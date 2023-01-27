// gobar
//
// Copyright (C) 2014-2015,2022 Karol 'Kenji Takahashi' Woźniak
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
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jezek/xgbutil/xgraphics"
)

var TokenizeTests = []struct {
	input           string
	advanceExpected int
	tokenExpected   string
}{
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

func assertEqual(t *testing.T, input, expected, actual interface{}, name string, i int) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%s:%d(%v) == %v != %v\n", name, i, input, actual, expected)
	}
}

func assertEqualError(t *testing.T, expected, actual error, name string, i int) {
	if actual == nil && expected == nil {
		return
	}
	if (actual == nil && expected != nil) || expected.Error() != actual.Error() {
		t.Errorf("%s:%d expected error `%s`, got `%s`\n", name, i, expected, actual)
	}
}

func TestTokenize(t *testing.T) {
	parser := NewTextParser()

	for _, tt := range TokenizeTests {
		// Do manual copy to ensure that cap(input) == len(tt.input)
		input := make([]byte, len(tt.input))
		for i, s := range tt.input {
			input[i] = byte(s)
		}

		advanceActual, tokenActual, err := parser.Tokenize(input, false)

		assertEqualError(t, nil, err, "Tokenize", 0)
		assertEqual(t, tt.input, tt.advanceExpected, advanceActual, "Tokenize", 0)
		assertEqual(t, tt.input, []byte(tt.tokenExpected), tokenActual, "Tokenize", 0)
	}
}

func TestTokenize_newline(t *testing.T) {
	parser := NewTextParser()

	input := "\ntest"
	advance, token, err := parser.Tokenize([]byte(input), false)

	assertEqual(t, input, 0, advance, "Tokenize_newline", 0)
	assertEqual(t, input, []byte(nil), token, "Tokenize_newline", 0)
	assertEqualError(t, errors.New("EndScan"), err, "Tokenize_newline", 0)
}

var ScanTests = []struct {
	input    string
	expected []*TextPiece
}{
	{"test", []*TextPiece{
		{Text: "test"},
	}},
	{"{F1test}", []*TextPiece{
		{Text: "test", Font: 1},
	}},
	{"{CF0xFF00AA33test}", []*TextPiece{
		{Text: "test", Foreground: &xgraphics.BGRA{B: 0x33, G: 0xAA, R: 0x00, A: 0xFF}},
	}},
	{"{CB0x33AA00FFtest}", []*TextPiece{
		{Text: "test", Background: &xgraphics.BGRA{B: 0xFF, G: 0x00, R: 0xAA, A: 0x33}},
	}},
	{"{ARtest}", []*TextPiece{
		{Text: "test", Align: RIGHT},
	}},
	{"{ARtest1{F1test2}}", []*TextPiece{
		{Text: "test2", Font: 1, Align: RIGHT}, {Text: "test1", Align: RIGHT},
	}},
	{"{AR{F1test1}test2}", []*TextPiece{
		{Text: "test2", Align: RIGHT}, {Text: "test1", Font: 1, Align: RIGHT},
	}},
	{"{S1test}", []*TextPiece{
		{Text: "test", Screens: []uint{1}},
	}},
	{"{S1,2test}", []*TextPiece{
		{Text: "test", Screens: []uint{1, 2}},
	}},
	{"{F1test1}test2", []*TextPiece{
		{Text: "test1", Font: 1}, {Text: "test2"},
	}},
	{"test1{F1test2}", []*TextPiece{
		{Text: "test1"}, {Text: "test2", Font: 1},
	}},
	{"test1{F1test2}test3", []*TextPiece{
		{Text: "test1"}, {Text: "test2", Font: 1}, {Text: "test3"},
	}},
	{"{F1test1}{F2test2}", []*TextPiece{
		{Text: "test1", Font: 1}, {Text: "test2", Font: 2},
	}},
	{"{F1test1}test2{F2test3}", []*TextPiece{
		{Text: "test1", Font: 1}, {Text: "test2"}, {Text: "test3", Font: 2},
	}},
	{"{F1{F2test1}}", []*TextPiece{
		{Text: "test1", Font: 2},
	}},
	{"{F1test1{F2test2}}", []*TextPiece{
		{Text: "test1", Font: 1}, {Text: "test2", Font: 2},
	}},
	{"{F1{F2test1}test2}", []*TextPiece{
		{Text: "test1", Font: 2}, {Text: "test2", Font: 1},
	}},
	{"{F1test1{F2test2}test3}", []*TextPiece{
		{Text: "test1", Font: 1}, {Text: "test2", Font: 2}, {Text: "test3", Font: 1},
	}},
	{"{S1test1}{F1{S1test2}test3}", []*TextPiece{
		{Text: "test1", Screens: []uint{1}},
		{Text: "test2", Font: 1, Screens: []uint{1}},
		{Text: "test3", Font: 1},
	}},
	{"}", []*TextPiece{
		{Text: "}"},
	}},
	{"\\{F", []*TextPiece{
		{Text: "{F"},
	}},
	{"\\{S", []*TextPiece{
		{Text: "{S"},
	}},
	{"\\{CF", []*TextPiece{
		{Text: "{CF"},
	}},
	{"\\{CB", []*TextPiece{
		{Text: "{CB"},
	}},
	{"\\}", []*TextPiece{
		{Text: "}"},
	}},
	{"{test1}", []*TextPiece{
		{Text: "test1"},
	}},
	{"\\{test1}", []*TextPiece{
		{Text: "{test1}"},
	}},
	{"\\{test1}{test2}", []*TextPiece{
		{Text: "{test1}test2"},
	}},
	{"{F1test1}{test2}{ARtest3}", []*TextPiece{
		{Text: "test1", Font: 1}, {Text: "test2"}, {Text: "test3", Align: RIGHT},
	}},
	{"{F1test1}test2", []*TextPiece{
		{Text: "test1", Font: 1}, {Text: "test2"},
	}},
	{"{F1{S2test1}}test2", []*TextPiece{
		{Text: "test1", Font: 1, Screens: []uint{2}}, {Text: "test2"},
	}},
	{"{F1{S2test1}test2}test3", []*TextPiece{
		{Text: "test1", Font: 1, Screens: []uint{2}}, {Text: "test2", Font: 1}, {Text: "test3"},
	}},
	{"{S-0test1}", []*TextPiece{
		{Text: "test1", NotScreens: []uint{0}},
	}},
	{"{S-1test1}", []*TextPiece{
		{Text: "test1", NotScreens: []uint{1}},
	}},
}

func TestScan(t *testing.T) {
	parser := NewTextParser()

	for i, tt := range ScanTests {
		actual := parser.Scan(strings.NewReader(tt.input))
		// We don't care about Origin
		for _, t := range actual {
			t.Origin = nil
		}

		assertEqual(t, tt.input, tt.expected, actual, "Scan", i)
	}
}

func BenchmarkScan(b *testing.B) {
	parser := NewTextParser()

	for i := 0; i < b.N; i++ {
		parser.Scan(strings.NewReader("{F1{S2test1}test2}test3"))
	}
}
