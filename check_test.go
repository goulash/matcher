// Copyright (c) 2015, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package matcher

import "testing"

func TestCheck(fw *testing.T) {
	tests := map[string]error{
		"abc":            nil,
		"*":              nil,
		"*c":             nil,
		"a*":             nil,
		"a*/b":           nil,
		"a*b*c*d*e*/f":   nil,
		"a*b?c*x":        nil,
		"ab[c]":          nil,
		"ab[b-d]":        nil,
		"ab[^c]":         nil,
		"ab[^b-d]":       nil,
		"a\\*b":          nil,
		"a[^a]b":         nil,
		"a???b":          nil,
		"a[^a][^a][^a]b": nil,
		"[a-ζ]*":         nil,
		"*[a-ζ]":         nil,
		"a?b":            nil,
		"a*b":            nil,
		"[\\]a]":         nil,
		"[\\-]":          nil,
		"[x\\-]":         nil,
		"[\\-x]":         nil,
		"[]a]":           ErrEmptyClass,
		"[-]":            ErrUnexpectedRune,
		"[x-]":           ErrUnexpectedRune,
		"[-x]":           ErrUnexpectedRune,
		"\\":             ErrTrailingEscape,
		"[a-b-c]":        ErrUnexpectedRune,
		"[":              ErrIncompleteClass,
		"[^":             ErrIncompleteClass,
		"[^bc":           ErrIncompleteClass,
		"a[":             ErrIncompleteClass,
		"*x":             nil,
		"":               ErrEmptyGlob,
		"  \t":           ErrTrailingWhitespace,
		" ":              ErrTrailingWhitespace,
		"ab ":            ErrTrailingWhitespace,
		"[\\--]":         ErrUnexpectedRune,
		"foo/**/bar":     ErrDualStar,
		"foo[]bar":       ErrEmptyClass,
		"[z-a]":          ErrNegativeRange,
		"]":              nil,
	}

	for k, v := range tests {
		err := Check(k)
		if err == nil {
			if v != nil {
				fw.Errorf("Check(%q) = nil, expected %q", k, v.Error())
			}
			continue
		}
		if e := err.(*BadPatternError); e.Err != v {
			s := "nil"
			if v != nil {
				s = v.Error()
			}
			fw.Errorf("Check(%q) = %q, expected %q", k, e.Error(), s)
		}
	}
}
