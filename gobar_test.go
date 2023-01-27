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
	"fmt"
	"log"
	"os"
	"testing"
)

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
