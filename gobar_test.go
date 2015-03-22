// gobar
// Copyright (C) 2014-2015 Karol 'Kenji Takahashi' Woźniak
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
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
)

func newFontMock(path string, size float64) (*Font, error) {
	if strings.Contains(path, "invalid") {
		return nil, errors.New("new font mock")
	}
	return &Font{Path: path, Size: size}, nil
}

func findFontMockFactory(value string) FontFinder {
	return func() (string, error) {
		if strings.Contains(value, "wrong") {
			return "", errors.New("find font mock")
		}
		return value, nil
	}
}

var ParseFontsTests = []struct {
	findFontValue string
	input         []string
	fontExpected  []*Font
	logExpected   []string
	errExpected   error
}{
	{
		"mock1", []string{"test1:14"},
		[]*Font{{Path: "test1", Size: 14}},
		[]string{""}, nil,
	},
	{
		"mock1", []string{"test1:14", "test2:10"},
		[]*Font{{Path: "test1", Size: 14}, {Path: "test2", Size: 10}},
		[]string{""}, nil,
	},
	{
		"mock1", []string{"test1"},
		[]*Font{{Path: "test1", Size: 12}},
		[]string{"No font size for `test1`, using `12`"}, nil,
	},
	{
		"mock1", []string{"test1:size1"},
		[]*Font{{Path: "test1", Size: 12}},
		[]string{"Invalid font size `size1` for `test1`, using `12`. Got"}, nil,
	},
	{
		"mock1", []string{"test1:14", "invalid1:10"},
		[]*Font{{Path: "test1", Size: 14}},
		[]string{"new font mock"}, nil,
	},
	{
		"mock1", []string{"invalid1:10"},
		[]*Font{{Path: "mock1", Size: 10}},
		[]string{"new font mock"}, nil,
	},
	{
		"mock1", []string{"invalid1"},
		[]*Font{{Path: "mock1", Size: 12}},
		[]string{"No font size for `invalid1`, using `12`", "new font mock"}, nil,
	},
	{
		"wrong1", []string{"invalid1:12"},
		[]*Font{},
		[]string{"new font mock"}, errors.New("find font mock"),
	},
	{
		"invalid1", []string{"invalid1:12"},
		[]*Font{},
		[]string{"new font mock"}, errors.New("new font mock"),
	},
}

func TestParseFonts(t *testing.T) {
	var stderr bytes.Buffer
	log.SetOutput(&stderr)

	for i, tt := range ParseFontsTests {
		findFontMock := findFontMockFactory(tt.findFontValue)

		actual, err := ParseFonts(tt.input, newFontMock, findFontMock)

		assertEqual(t, tt.input, tt.fontExpected, actual, "ParseFonts", i)
		assertEqualError(t, tt.errExpected, err, "ParseFonts", i)

		for _, logExpected := range tt.logExpected {
			logActual, err := stderr.ReadString('\n')
			if err != nil && err.Error() != "EOF" {
				t.Errorf("ERROR `%s` READING STDERR", err)
			}

			if len(logActual) > 0 {
				gotIdx := strings.Index(logActual, ". Got")
				if gotIdx == -1 {
					logActual = logActual[20 : len(logActual)-1]
				} else {
					logActual = logActual[20 : gotIdx+5]
				}
			}

			assertEqual(t, tt.input, logActual, logExpected, "ParseFonts", i)
		}
	}
}

func TestGeometriesSet(t *testing.T) {
	tests := []struct {
		input  string
		logs   string
		output Geometries
	}{
		{"", "", Geometries{}},
		{"0x16+0+0", "", Geometries{
			{0, 16, 0, 0},
		}},
		{",0x16+0+0", "", Geometries{
			nil,
			{0, 16, 0, 0},
		}},
		{"0x16+0+0,", "", Geometries{
			{0, 16, 0, 0},
			nil,
		}},
		{",0x16+0+0,", "", Geometries{
			nil,
			{0, 16, 0, 0},
			nil,
		}},
		{"22x01+20+15", "", Geometries{
			{22, 1, 20, 15},
		}},
		{",0x16+0+0,22x01+20+15,", "", Geometries{
			nil,
			{0, 16, 0, 0},
			{22, 1, 20, 15},
			nil,
		}},
		{",0x16+0+0,,22x01+20+15,", "", Geometries{
			nil,
			{0, 16, 0, 0},
			nil,
			{22, 1, 20, 15},
			nil,
		}},
		{"wrongo", "Bad geometry `wrongo`, using default\n", Geometries{
			{0, 16, 0, 0},
		}},
	}

	var stderr bytes.Buffer
	log.SetOutput(&stderr)

	for i, test := range tests {
		geometries := Geometries{}

		geometries.Set(test.input)
		logs, _ := stderr.ReadString('\n')

		assertEqual(t, test.input, test.output, geometries, "GeometriesSet", i)
		if logs != "" {
			assertEqual(t, test.input, test.logs, logs[20:], "GeometriesSet", i)
		}
	}

	geometries := Geometries{{0, 16, 0, 0}}
	err := geometries.Set("")
	assertEqualError(t, fmt.Errorf("geometries flag already set"), err, "GeometriesSet", -1)

	log.SetOutput(os.Stderr)
}
