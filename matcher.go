// Copyright (c) 2015, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// Package matcher provides matching files akin to gitignore.
// This package does not implement the gitignore standard, but it aspires to one day.
// If there are any bugs, please report them!
//
// This package does not define what the matching means. Whether it is to ignore
// files or not is up to the user.
//
// Pattern Format
//
// A blank line matches no files, so it can serve as a separator for
// readability.
//
// A line starting with # serves as a comment. Put a backslash ("\") in front
// of the first hash for patterns that begin with a hash.
//
// Trailing and leading spaces are ignored unless they are quoted with backslash ("\").
// Any character that is quoted with a backslash is interpreted as is.
//
// If the pattern does not contain a slash /, it is treated as a shell glob
// applicable to only the basename of files. Otherwise, it is matched against
// the full filename.
//
// Otherwise, the pattern is as defined in filepath.Match:
//
//  pattern:
//      { term }
//  term:
//      '*'         matches any sequence of non-Separator characters
//      '?'         matches any single non-Separator character
//      '[' [ '^' ] { character-range } ']'
//                  character class (must be non-empty)
//      c           matches character c (c != '*', '?', '\\', '[')
//      '\\' c      matches character c
//
//  character-range:
//      c           matches character c (c != '\\', '-', ']')
//      '\\' c      matches character c
//      lo '-' hi   matches character c for lo <= c <= hi
//
// Unfortunately, filepath.Match may or may not fail, depending on the glob and the string.
// This package defines the Check function, which attempts to validate a glob beforehand.
// If there is an error during matching, the function panics with the error. This indicates
// a bug in the matcher package. Please report it!
package matcher

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrMissingDir  = errors.New("need path to current directory for worker")
	ErrGlobIsPath  = errors.New("glob cannot contain path separators")
	ErrConfigUnset = errors.New("config is unset")
)

// Matcher is the starting point for matching. When creating a matcher,
// the configuration filename is specified. When creating a Worker,
// the configuration filename is looked for in the working directory
// of the Worker, as well as all parent directories.
//
// Nothing is matched by default, not even the configuration file.
// It is therefore recommended to add this, if necessary.
//
// Example:
//
//      m := NewMatcher(".dunignore")
//      m.Add(".dun*")
//      ...
//      w, err := m.NewWorker(cwd)
//      if err != nil {
//          log.Errorln(err)
//      }
//      ...
//      if w.Matches(filename) {
//          fmt.Printf("File %q matches!\n", filename)
//      }
//
type Matcher struct {
	// ErrHandler handles various errors that arise when creating a Worker.
	// In particular, the errors that arise when trying to read all configuration
	// files.
	//
	// It may be convenient to simply ignore the errors, in which case ErrHandler
	// can be left nil. If an error is returned, NewWorker will abort.
	ErrHandler func(error) error

	config string
	global []string
}

// New creates a new Matcher, which contains only global globs.
// A global glob is a glob that only applies to the basename,
// and hence does not have any slashes ("/").
//
// Matcher is safe to use concurrently, as long as you don't add any globs.
// If it is necessary to add local globs, use a Worker.
func New(config string) *Matcher {
	return &Matcher{
		config: config,
		global: make([]string, 0),
	}
}

// Add adds the globs to the global matcher.
// None of the globs may contain a path character.
func (m *Matcher) Add(globs ...string) error {
	return addAll(&m.global, globs)
}

// Matches returns true if any of the global globs matches.
//
// There should be no errors in matching, because globs are checked with the
// Check function. If there is an error, however, the function panics with the
// error.
func (m *Matcher) Matches(path string) bool {
	return matchAll(m.global, filepath.Base(path))
}

// Worker is derived from Matcher, and loads globs from configurations.
// Globs in configurations may be paths.
//
// For each concurrent use, a separate Worker is required.
type Worker struct {
	cwd    string
	local  []string
	global []string
}

// NewWorker creates a new Worker.
func (m *Matcher) NewWorker(dir string) (*Worker, error) {
	var err error

	dir = filepath.Clean(dir)
	if dir == "" {
		return nil, ErrMissingDir
	}
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
	}

	w := &Worker{
		cwd:    dir,
		local:  make([]string, 0),
		global: m.global,
	}

	// Read configuration files in each directory from
	// the current till we reach the root.
	// If m.config is not set, we skip this.
	if m.config != "" {
		for {
			err := w.AddFile(filepath.Join(dir, m.config))
			if err != nil && !os.IsNotExist(err) && m.ErrHandler != nil {
				err = m.ErrHandler(err)
				if err != nil {
					return nil, err
				}
			}

			dir = filepath.Clean(filepath.Join(dir, ".."))
			if dir == "/" {
				break
			}
		}
	}
	return w, nil
}

// Add adds the globs to the local matcher.
// None of the globs may contain a path character.
func (w *Worker) Add(glob ...string) error {
	return addAll(&w.local, glob)
}

// AddFile loads a file containing globs. The format of the file
// is similar to gitignore.
func (w *Worker) AddFile(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	f, err := os.Open(abs)
	if err != nil {
		return err
	}

	var line int
	base := filepath.Dir(abs)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line++

		s := Clean(sc.Text())
		if s == "" {
			continue
		}
		err = Check(s)
		if err != nil {
			pe := err.(*BadPatternError)
			pe.Line = line
			pe.File = path
			return pe
		}

		if strings.Contains(s, "/") {
			s = filepath.Join(base, s)
		}
		w.local = append(w.local, s)
	}
	return sc.Err()
}

// Reset clears the set of local globs,
// i.e. the globs that are added by AddFile, or are read
// through loading configs.
func (w *Worker) Reset() {
	w.local = w.local[:0]
}

// Matches returns true if any of the global or local globs matches.
//
// There should be no errors in matching, because globs are checked with the
// Check function. If there is an error, however, the function panics with the
// error.
func (w *Worker) Matches(path string) bool {
	path = filepath.Clean(path)
	if path == "" {
		return false
	}
	if !filepath.IsAbs(path) {
		path = filepath.Clean(filepath.Join(w.cwd, path))
	}

	for _, l := range [][]string{w.global, w.local} {
		if matchAll(l, path) {
			return true
		}
	}
	return false
}

func match(pattern, s string) bool {
	if pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "/") {
		s = filepath.Base(s)
	}
	m, err := filepath.Match(pattern, s)
	if err != nil {
		panic(err)
	}
	return m
}

func matchAll(patterns []string, s string) bool {
	for _, p := range patterns {
		if match(p, s) {
			return true
		}
	}
	return false
}

func add(list *[]string, glob string) error {
	err := Check(glob)
	if err != nil {
		return err
	}
	if strings.Contains(glob, "/") {
		return ErrGlobIsPath
	}
	*list = append(*list, glob)
	return nil
}

func addAll(list *[]string, globs []string) error {
	for _, g := range globs {
		err := add(list, g)
		if err != nil {
			return err
		}
	}
	return nil
}
