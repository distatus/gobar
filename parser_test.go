// gobar
// Copyright (C) 2014 Karol 'Kenji Takahashi' Woźniak
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
// OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BurntSushi/xgbutil/xgraphics"

	"github.com/stretchr/testify/assert"
)

var TokenizeTests = []struct {
	input            string
	expected_advance int
	expected_token   string
}{
	//FIXME: Create []byte using make, with cap 1 hard
	{"t", 1, "t"},
	{"te", 1, "t"},
	{"tes", 1, "t"},
	{"test", 1, "t"},
	{"{Ftest", 2, "{F"},
	{"{Stest", 2, "{S"},
	{"{CFtest", 3, "{CF"},
	{"{CBtest", 3, "{CB"},
	{"{ARtest", 3, "{AR"},
	{"0xff1eF09atest", 10, "0xff1eF09a"},
	{"0xff1eF0test", 1, "0"},
	{"0312495test", 7, "0312495"},
	{"5942130", 7, "5942130"},
}

func TestTokenize(t *testing.T) {
	parser := NewTextParser()

	for _, tt := range TokenizeTests {
		// Do manual copy to ensure that cap(input) == len(tt.input)
		input := make([]byte, len(tt.input))
		for i, s := range tt.input {
			input[i] = byte(s)
		}

		actual_advance, actual_token, err := parser.Tokenize(input, false)

		assert.NoError(t, err)
		assert.Equal(t, tt.expected_advance, actual_advance)
		assert.Equal(t, []byte(tt.expected_token), actual_token)
	}
}

func TestTokenize_newline(t *testing.T) {
	parser := NewTextParser()

	advance, token, err := parser.Tokenize([]byte("\ntest"), false)

	assert.Equal(t, 0, advance)
	assert.Equal(t, []byte(nil), token)
	assert.EqualError(t, err, "EndScan")
}

var ScanTests = []struct {
	input    string
	expected []*TextPiece
}{
	{"test", []*TextPiece{
		&TextPiece{Text: "test"},
	}},
	{"{F1test}", []*TextPiece{
		&TextPiece{Text: "test", Font: 1},
	}},
	{"{CF0xFF00AA33test}", []*TextPiece{
		&TextPiece{
			Text:       "test",
			Foreground: &xgraphics.BGRA{B: 0x33, G: 0xAA, R: 0x00, A: 0xFF},
		},
	}},
	{"{CB0x33AA00FFtest}", []*TextPiece{
		&TextPiece{
			Text:       "test",
			Background: &xgraphics.BGRA{B: 0xFF, G: 0x00, R: 0xAA, A: 0x33},
		},
	}},
	{"{ARtest}", []*TextPiece{
		&TextPiece{Text: "test", Align: RIGHT},
	}},
	{"{ARtest1{F1test2}}", []*TextPiece{
		&TextPiece{Text: "test2", Font: 1, Align: RIGHT},
		&TextPiece{Text: "test1", Align: RIGHT},
	}},
	{"{AR{F1test1}test2}", []*TextPiece{
		&TextPiece{Text: "test2", Align: RIGHT},
		&TextPiece{Text: "test1", Font: 1, Align: RIGHT},
	}},
	{"{S1test}", []*TextPiece{
		&TextPiece{Text: "test", Screens: []uint{1}},
	}},
	{"{S1,2test}", []*TextPiece{
		&TextPiece{Text: "test", Screens: []uint{1, 2}},
	}},
	{"{F1test1}test2", []*TextPiece{
		&TextPiece{Text: "test1", Font: 1},
		&TextPiece{Text: "test2"},
	}},
	{"test1{F1test2}", []*TextPiece{
		&TextPiece{Text: "test1"},
		&TextPiece{Text: "test2", Font: 1},
	}},
	{"test1{F1test2}test3", []*TextPiece{
		&TextPiece{Text: "test1"},
		&TextPiece{Text: "test2", Font: 1},
		&TextPiece{Text: "test3"},
	}},
	{"{F1test1}{F2test2}", []*TextPiece{
		&TextPiece{Text: "test1", Font: 1},
		&TextPiece{Text: "test2", Font: 2},
	}},
	{"{F1test1}test2{F2test3}", []*TextPiece{
		&TextPiece{Text: "test1", Font: 1},
		&TextPiece{Text: "test2"},
		&TextPiece{Text: "test3", Font: 2},
	}},
	{"{F1{F2test1}}", []*TextPiece{
		&TextPiece{Text: "test1", Font: 2},
	}},
	{"{F1test1{F2test2}}", []*TextPiece{
		&TextPiece{Text: "test1", Font: 1},
		&TextPiece{Text: "test2", Font: 2},
	}},
	{"{F1{F2test1}test2}", []*TextPiece{
		&TextPiece{Text: "test1", Font: 2},
		&TextPiece{Text: "test2", Font: 1},
	}},
	{"{F1test1{F2test2}test3}", []*TextPiece{
		&TextPiece{Text: "test1", Font: 1},
		&TextPiece{Text: "test2", Font: 2},
		&TextPiece{Text: "test3", Font: 1},
	}},
	{"{S1test1}{F1{S1test2}test3}", []*TextPiece{
		&TextPiece{Text: "test1", Screens: []uint{1}},
		&TextPiece{Text: "test2", Font: 1, Screens: []uint{1}},
		&TextPiece{Text: "test3", Font: 1},
	}},
	{"}", []*TextPiece{
		&TextPiece{Text: "}"},
	}},
	{"\\{F", []*TextPiece{
		&TextPiece{Text: "{F"},
	}},
	{"\\{S", []*TextPiece{
		&TextPiece{Text: "{S"},
	}},
	{"\\{CF", []*TextPiece{
		&TextPiece{Text: "{CF"},
	}},
	{"\\{CB", []*TextPiece{
		&TextPiece{Text: "{CB"},
	}},
	{"\\}", []*TextPiece{
		&TextPiece{Text: "}"},
	}},
}

func TestScan(t *testing.T) {
	parser := NewTextParser()

	for i, tt := range ScanTests {
		actual, err := parser.Scan(strings.NewReader(tt.input))

		assert.NoError(t, err)
		assert.Equal(
			t, tt.expected, actual,
			fmt.Sprintf("%d: Scan(%q) => %q != %q", i, tt.input, actual, tt.expected),
		)
	}
}
