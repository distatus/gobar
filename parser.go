// gobar
//
// Copyright (C) 2014,2022 Karol 'Kenji Takahashi' Woźniak
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
	"bufio"
	"io"
	"log"
	"regexp"
	"strconv"

	"github.com/jezek/xgbutil/xgraphics"
)

// Align defines text piece alignment on the screen.
type Align uint8

const (
	LEFT Align = iota
	RIGHT
)

// EndScan is an artifical Error.
// Raised when parser should stop scanning.
type EndScan struct{}

func (e EndScan) Error() string { return "EndScan" }

// NewBGRA returns a new color definition in X compatible format.
// Input should be a hexagonal representation with alpha, i.e 0xAARRGGBB.
func NewBGRA(color uint64) *xgraphics.BGRA {
	a := uint8(color >> 24)
	r := uint8((color & 0x00ff0000) >> 16)
	g := uint8((color & 0x0000ff00) >> 8)
	b := uint8(color & 0x000000ff)
	return &xgraphics.BGRA{B: b, G: g, R: r, A: a}
}

// TextPiece stores formatting information for a text
// within single pair of brackets.
type TextPiece struct {
	Text       string
	Font       uint
	Align      Align
	Foreground *xgraphics.BGRA
	Background *xgraphics.BGRA
	Screens    []uint
	NotScreens []uint

	Origin *TextPiece
}

// TextParser is used to create a set of TextPieces from a textual definition.
type TextParser struct {
	rgbPattern *regexp.Regexp
}

// NewTextParser creates TextParser instance with
// correct necessary regexp definitions.
func NewTextParser() *TextParser {
	return &TextParser{regexp.MustCompile(`^0[xX][0-9a-fA-F]{8}$`)}
}

// Tokenize turns textual definition into a series of valid tokens.
// If no valid token is found at given place, char at 0 position is returned.
func (tp *TextParser) Tokenize(
	data []byte, EOF bool,
) (advance int, token []byte, err error) {
	if EOF {
		return
	}
	switch {
	case data[0] == '\n':
		err = EndScan{}
	case len(data) < 2:
		advance, token, err = 1, data[:1], nil
	case string(data[:2]) == "{F":
		advance, token, err = 2, data[:2], nil
	case string(data[:2]) == "{S":
		advance, token, err = 2, data[:2], nil
	case len(data) < 3:
		advance, token, err = 1, data[:1], nil
	case string(data[:3]) == "{CF":
		advance, token, err = 3, data[:3], nil
	case string(data[:3]) == "{CB":
		advance, token, err = 3, data[:3], nil
	case string(data[:3]) == "{AR":
		advance, token, err = 3, data[:3], nil
	case len(data) >= 10 && tp.rgbPattern.Match(data[:10]):
		advance, token, err = 10, data[:10], nil
	case ('0' <= data[0] && data[0] <= '9') || data[0] == '-':
		i := 0
		if data[0] == '-' {
			i = 1
		}
		for _, n := range data[i:] {
			if !('0' <= n && n <= '9') {
				break
			}
			i++
		}
		advance, token, err = i, data[:i], nil
	default: // Also contains '}' and ','
		// TODO: Parsing whole text piece here, instead of returning
		// char-by-char, should perform better
		advance, token, err = 1, data[:1], nil
	}
	return
}

// Scan scans textual definition and returns array of TextPieces.
// Possible empty pieces are omitted in the returned array.
func (tp *TextParser) Scan(r io.Reader) []*TextPiece {
	var text []*TextPiece

	scanner := bufio.NewScanner(r)

	scanner.Split(tp.Tokenize)

	currentText := &TextPiece{}
	text = append(text, currentText)

	currentIndex := func() int {
		for i, t := range text {
			if t == currentText {
				return i
			}
		}
		return 0
	}

	moveCurrent := func(end bool) *TextPiece {
		newCurrent := &TextPiece{}
		if end {
			*newCurrent = *currentText.Origin
		} else {
			*newCurrent = *currentText
			newCurrent.Origin = currentText
		}
		newCurrent.Text = ""
		if currentText.Align == RIGHT {
			i := currentIndex()
			text = append(text, &TextPiece{})
			copy(text[i+1:], text[i:])
			text[i] = newCurrent
		} else {
			text = append(text, newCurrent)
		}
		currentText = newCurrent
		return newCurrent
	}

	logPieceError := func(err error, pieces ...string) {
		log.Printf("Problem parsing `%q`: %s", pieces, err)
		for _, piece := range pieces {
			currentText.Text += piece
		}
	}

	screening := false
	escaping := false
	bracketing := 0
	for scanner.Scan() {
		stext := scanner.Text()
		switch {
		case stext == "\\":
			escaping = true
			continue
		case !escaping && stext == "{F":
			scanner.Scan()
			text := scanner.Text()
			font, err := strconv.Atoi(text)
			if err != nil {
				logPieceError(err, stext, text)
			}
			newCurrent := moveCurrent(false)
			newCurrent.Font = uint(font)
		case !escaping && stext == "{S":
			scanner.Scan()
			text := scanner.Text()
			screen, err := strconv.Atoi(text)
			if err != nil {
				logPieceError(err, stext, text)
			}
			newCurrent := moveCurrent(false)
			if text[0] == '-' {
				newCurrent.NotScreens = append(newCurrent.NotScreens, uint(-screen))
			} else {
				newCurrent.Screens = append(newCurrent.Screens, uint(screen))
			}
			screening = true
		case !escaping && stext == "{CF":
			scanner.Scan()
			text := scanner.Text()
			fg, err := strconv.ParseUint(text, 0, 32)
			if err != nil {
				logPieceError(err, stext, text)
			}
			newCurrent := moveCurrent(false)
			newCurrent.Foreground = NewBGRA(fg)
		case !escaping && stext == "{CB":
			scanner.Scan()
			text := scanner.Text()
			bg, err := strconv.ParseUint(text, 0, 32)
			if err != nil {
				logPieceError(err, stext, text)
			}
			newCurrent := moveCurrent(false)
			newCurrent.Background = NewBGRA(bg)
		case !escaping && stext == "{AR":
			newCurrent := moveCurrent(false)
			newCurrent.Align = RIGHT
		case !escaping && stext == "{":
			bracketing++
		case !escaping && stext == "}":
			if bracketing > 0 {
				bracketing--
				continue
			}
			screening = false
			if currentText.Origin != nil {
				moveCurrent(true)
				continue
			}
			fallthrough
		default:
			if screening && stext == "," {
				scanner.Scan()
				text := scanner.Text()
				screen, err := strconv.Atoi(text)
				if err != nil {
					logPieceError(err, stext, text)
				}
				currentText.Screens = append(currentText.Screens, uint(screen))
			} else {
				currentText.Text += stext
			}
			escaping = false
		}
	}

	//Remove possible empty pieces.
	var text2 []*TextPiece
	for _, piece := range text {
		if piece.Text != "" {
			text2 = append(text2, piece)
		}
	}

	return text2
}
