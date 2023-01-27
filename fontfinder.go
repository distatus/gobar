// gobar
//
// Copyright (C) 2022 Karol 'Kenji Takahashi' Woźniak
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
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/adrg/sysfont"
	"github.com/flopp/go-findfont"
	"github.com/jezek/xgbutil/xgraphics"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/font/opentype"
)

func findFont(def string) font.Face {
	i := strings.LastIndexByte(def, ':')
	name, size := parseSize(def, i)

	fontPath, err := findfont.Find(name)
	if err != nil {
		log.Printf("Could not find font `%s`, trying alternate method: %s", def, err)
		return findFontFallback(def, size)
	}
	fontFile, err := os.Open(fontPath)
	if err != nil {
		log.Printf("Could not open font `%s`, trying to find another one: %s", fontPath, err)
		return findFontFallback(def, size)
	}
	face, err := parseFontFace(fontFile, size)
	if err != nil {
		log.Printf("Could not parse font `%s`, trying to find another one: %s", fontPath, err)
		return findFontFallback(def, size)
	}
	return face
}

var fallbackFinder *sysfont.Finder = nil

func findFontFallback(def string, size float64) font.Face {
	if fallbackFinder == nil {
		fallbackFinder = sysfont.NewFinder(nil)
	}

	fontDef := fallbackFinder.Match(def)
	if fontDef == nil {
		log.Printf("Could not find font `%s`, using `inconsolata regular 8x16`", def)
		return inconsolata.Regular8x16
	}
	fontFile, err := os.Open(fontDef.Filename)
	if err != nil {
		log.Printf("Could not open font `%s`, using `inconsolata regular 8x16`: %s", fontDef.Filename, err)
		return inconsolata.Regular8x16
	}
	face, err := parseFontFace(fontFile, size)
	if err != nil {
		log.Printf("Could not parse font `%s`, using `inconsolata regular 8x16`: %s", fontDef.Filename, err)
		return inconsolata.Regular8x16
	}
	log.Printf("Found fallback font `%s`", fontDef.Filename)
	return face
}

func parseFontFace(file io.Reader, size float64) (font.Face, error) {
	otf, err := xgraphics.ParseFont(file)
	if err != nil {
		return nil, err
	}
	// XXX Can we somehow figure out DPI?
	face, err := opentype.NewFace(otf, &opentype.FaceOptions{Size: size, DPI: 72})
	if err != nil {
		return nil, err
	}
	return face, nil
}

func parseSize(def string, i int) (string, float64) {
	if i == -1 {
		log.Printf("Font size not specified for `%s`, using `12`", def)
		return def, 12
	}
	name, sizeStr := def[:i], def[i+1:]
	size, err := strconv.ParseFloat(sizeStr, 32)
	if err != nil {
		log.Printf("Invalid font size `%s` for `%s`, using `12`: `%s`", sizeStr, name, err)
		size = 12
	}
	return name, size
}
