// Copyright (c) 2015, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package matcher

import (
	"bytes"
	"errors"
	"fmt"
)

// The (above) error variables are returned by Check in BadPatternError.
var (
	ErrUnexpectedRune     = errors.New("unexpected rune")
	ErrNegativeRange      = errors.New("negative range")
	ErrDualStar           = errors.New("dual stars not supported")
	ErrEmptyClass         = errors.New("character class empty")
	ErrEmptyGlob          = errors.New("glob empty")
	ErrIncompleteClass    = errors.New("character class incomplete")
	ErrTrailingEscape     = errors.New("trailing escape character")
	ErrTrailingWhitespace = errors.New("trailing whitespace")
)

// BadPatternError is what is returned by Check.
//
// If not nil, the Err field can only be one of the following errors:
//
//     ErrUnexpectedRune
//     ErrNegativeRange
//     ErrDualStar
//     ErrEmptyClass
//     ErrEmptyGlob
//     ErrIncompleteClass
//     ErrTrailingEscape
//     ErrTrailingWhitespace
//
type BadPatternError struct {
	Err    error
	Column int
	Line   int
	File   string
}

func (pe *BadPatternError) Error() string {
	if pe.Line < 0 {
		return fmt.Sprintf("column %d: %s", pe.Column, pe.Err)
	}
	return fmt.Sprintf("%s:%d:%d: %s", pe.File, pe.Line, pe.Column, pe.Err)
}

// Check returns nil when the glob pattern is okay.
// The pattern syntax is:
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
// The only possible returned error is BadPatternError, when pattern
// is malformed.
func Check(glob string) error {
	type State int
	const (
		Initial State = iota
		Regular
		ClassBegin
		ClassMiddle
		ClassRange
		ClassRequire
		Star
		DualStar
		Escape
		Whitespace
	)

	column := -1
	give := func(e error) error {
		return &BadPatternError{
			Err:    e,
			Column: column,
			Line:   -1,
			File:   "",
		}
	}

	var last rune
	var state State
	var next State
	for _, r := range glob {
		column++
		switch state {
		case Initial:
			state = Regular
			fallthrough
		case Regular:
			switch r {
			case '[':
				state = ClassBegin
			case '*':
				state = Star
			case '\\':
				state = Escape
				next = Regular
			case ' ', '\t', '\n': // find out if this is the end
				state = Whitespace
			default:
			}
		case ClassBegin:
			switch r {
			case ']':
				return give(ErrEmptyClass)
			case '-':
				return give(ErrUnexpectedRune)
			case '\\':
				state = Escape
				next = ClassRequire
			default:
				last = r
				state = ClassMiddle
			}
		case ClassRequire:
			if r == '-' {
				return give(ErrUnexpectedRune)
			}
			state = ClassMiddle
			fallthrough
		case ClassMiddle:
			switch r {
			case '\\':
				state = Escape
				next = ClassMiddle
			case ']':
				state = Regular
			case '-':
				state = ClassRange
			default:
				last = r
			}
		case ClassRange:
			switch r {
			// TODO: following case may be unnecessary
			case '-', '\\', ']':
				return give(ErrUnexpectedRune)
			default:
				if r-last < 0 {
					return give(ErrNegativeRange)
				}
				state = ClassRequire
			}
		case Escape:
			state = next
		case Star:
			switch r {
			case '*':
				state = DualStar
			default:
				state = Regular
			}
		case DualStar:
			return give(ErrDualStar)
		case Whitespace:
			switch r {
			case ' ', '\t', '\n':
			default:
				state = Regular
			}
		}
	}

	switch state {
	case Initial:
		return give(ErrEmptyGlob)
	case ClassBegin, ClassMiddle, ClassRange:
		return give(ErrIncompleteClass)
	case Escape:
		return give(ErrTrailingEscape)
	case Whitespace:
		return give(ErrTrailingWhitespace)
	default:
		return nil
	}
}

// Clean discards parts of s that are not needed, as gitignore does.
// If the returned string is not empty, then s parsed OK.
//
// If the string starts with a hash ("#"), then "" is returned.
// Trailing whitespace is removed unless escaped.
// A sole trailing escape character is removed.
func Clean(s string) string {
	type State int
	const (
		Initial State = iota
		Regular
		Escape
		Whitespace
	)

	var state State
	var buf bytes.Buffer
	var sp bytes.Buffer
	for _, r := range s {
		switch state {
		case Initial:
			if r == '#' {
				return ""
			}
			state = Regular
			fallthrough
		case Regular:
			switch r {
			case ' ', '\t', '\n':
				state = Whitespace
				sp.WriteRune(r)
			case '\\':
				state = Escape
			default:
				buf.WriteRune(r)
			}
		case Escape:
			state = Regular
			buf.WriteRune('\\')
			buf.WriteRune(r)
		case Whitespace:
			switch r {
			case ' ', '\t', '\n':
				sp.WriteRune(r)
			default:
				state = Regular
				buf.Write(sp.Bytes())
				sp.Reset()
			}
		}
	}

	return buf.String()
}
