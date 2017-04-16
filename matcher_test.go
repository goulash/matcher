// Copyright (c) 2015, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package matcher

import (
	"path/filepath"
	"testing"
)

func TestMatch(fw *testing.T) {
	type io struct {
		Path  string
		Match bool
	}
	type test struct {
		Pattern string
		IO      []io
	}

	tests := []test{
		test{"", []io{
			{"", false},
			{"/", false},
			{"/home", false},
			{"local", false},
		}},
		test{" \t", []io{
			{"", false},
			{"/", false},
			{"/home", false},
			{"local", false},
		}},
		test{"foo", []io{
			{"foo", true},
			{"bar/foo", true},
			{"foobar", false},
		}},
		test{"bar/*", []io{
			{"foo", false},
			{"bar/foo", true},
			{"bar", false},
		}},
		test{"Documentation/*.html", []io{
			{"Documentation/git.html", true},
			{"Documentation/xyz/git.html", false},
			{"tools/Documentation/perf.html", false},
		}},
	}

	for _, t := range tests {
		for _, p := range t.IO {
			if m := match(t.Pattern, p.Path); m != p.Match {
				fw.Errorf("match(%q, %q) = %v, expected %v", t.Pattern, p.Path, m, p.Match)
			}
		}
	}
}

func TestMatcher(fw *testing.T) {
	// Inside tests directory
	var tests = map[string]bool{
		"bar":                  true,
		"brain":                false,
		"brain/bar":            false,
		"brain/foo":            true,
		"brain/yahoo":          false,
		"dead":                 false,
		"dead/bad":             false,
		"dead/bad/shubarf":     true,
		"dead/bad/somefoo":     true,
		"dead/good":            false,
		"dead/good/1":          false,
		"dead/good/2":          true,
		"dead/good/3":          true,
		"dead/good/4":          true,
		"dead/good/5":          true,
		"dead/good/6":          false,
		"dead/good/7":          false,
		"dead/good/8":          false,
		"dead/good/9":          false,
		"dead/good/cache":      false,
		"dead/good/hit":        false,
		"dead/good/match.conf": true,
		"dead/good/yala":       false,
		"dead/match.conf":      true,
		"dead/never":           false,
		"dead/ok":              true,
		"dead/somewhere":       false,
		"dead/ugly":            false,
		"dead/ugly/bar":        true,
		"dead/ugly/foo":        true,
		"dead/ugly/foobar":     true,
		"foo":                  true,
		"jack":                 false,
		"lucy":                 false,
		"match.conf":           true,
	}

	m := New("match.conf")
	m.ErrHandler = func(err error) error {
		fw.Fatalf("Error: %s", err)
		return nil
	}
	err := m.Add("match.conf")
	if err != nil {
		fw.Fatalf("Adding glob %q failed: %s", "match.conf", err)
	}

	for k, v := range tests {
		b := filepath.Base(k)
		w, err := m.NewWorker(filepath.Join("tests", filepath.Dir(k)))
		if err != nil {
			fw.Fatalf("Creating new Worker failed: %s", err)
		}

		if u := w.Matches(b); u != v {
			path, _ := filepath.Abs(filepath.Join("tests", filepath.Dir(k)))
			fw.Errorf("w.Matches(%q) = %v, expected %v", b, u, v)
			fw.Errorf("  Variables (dir,file) = (%q,%q)", path, b)
			fw.Errorf("  Available globs are:\n%s\n%s", w.global, w.local)
		}
	}
}
