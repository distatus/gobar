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
	"image"
	"log"
	"os"
	"strconv"
	"strings"

	"code.google.com/p/jamslam-freetype-go/freetype/truetype"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/BurntSushi/xgbutil/xwindow"
	"github.com/BurntSushi/xgbutil/xinerama"
	"github.com/BurntSushi/xgbutil/xrect"

	"github.com/docopt/docopt-go"
)

// fatal is a helper function to call when something terribly wrong
// might happen. Logs given error and terminates application.
func fatal(err error) {
	if err != nil {
		log.Panic(err)
	}
}

// Position defines bar placement on the screen.
type Position uint8

const (
	BOTTOM Position = iota
	TOP    Position = iota
)

// Font stores font definition along with it's loaded truetype struct.
type Font struct {
	Path string
	Size float64
	Font *truetype.Font
}

// NewFont opens a font file and parses it with truetype engine.
// TODO(Kenji): Fallback when path doesn't exist.
func NewFont(path string, size float64) *Font {
	fontReader, err := os.Open(path)
	fatal(err)
	font, err := xgraphics.ParseFont(fontReader)
	fatal(err)
	return &Font{path, size, font}
}

// Geometry stores bars geometry on the screen (or actually monitor).
type Geometry struct {
	Width  uint16
	Height uint16
	X      uint16
	Y      uint16
}

// NewGeometry parses geometry from textual definition.
// Input should be formatted as <width>x<height>+<x>+<y>.
// A special "M" value is allowed as <width>, to represent 100%.
func NewGeometry(
	geostr string, head xrect.Rect, position Position,
) *Geometry {
	geometry := Geometry{}
	var widthS string
	_, err := fmt.Sscanf(
		geostr, "%1sx%d+%d+%d",
		&widthS, &geometry.Height, &geometry.X, &geometry.Y,
	)

	geometry.X += uint16(head.X())

	hwidth := uint16(head.Width())
	hheight := uint16(head.Height())
	if err != nil {
		log.Print("bad geometry, ", geometry, ", using default")
		geometry.Width = hwidth - (geometry.X - uint16(head.X()))
		geometry.Height = 16
	} else {
		width, err := strconv.Atoi(widthS)
		if err != nil {
			if widthS != "M" {
				// Make people feel guilty
				log.Print("wrong geometry width")
			}
			geometry.Width = hwidth - (geometry.X - uint16(head.X()))
		} else {
			geometry.Width = uint16(width)
		}
	}

	if position == BOTTOM {
		geometry.Y = uint16(head.Y()) + hheight - geometry.Height - geometry.Y
	}

	return &geometry
}

// Bar stores and manages all X related stuff and configuration.
type Bar struct {
	X          *xgbutil.XUtil
	Windows    []*xwindow.Window
	Geometries []*Geometry
	Foreground *xgraphics.BGRA
	Background *xgraphics.BGRA
	Colors     []*xgraphics.BGRA
	Fonts      []*Font
}

// NewBar creates X windows for every monitor.
// Also sets proper EWMH information for docked windows.
func NewBar(
	X *xgbutil.XUtil, geometries []*Geometry, position Position,
	fg uint64, bg uint64, fonts []*Font,
) *Bar {
	windows := make([]*xwindow.Window, len(geometries))
	for i, geometry := range geometries {
		win, err := xwindow.Generate(X)
		fatal(err)
		win.Create(
			X.RootWin(),
			int(geometry.X), int(geometry.Y),
			int(geometry.Width), int(geometry.Height),
			0,
		)

		ewmh.WmWindowTypeSet(X, win.Id, []string{"_NET_WM_WINDOW_TYPE_DOCK"})
		ewmh.WmStateSet(X, win.Id, []string{"_NET_WM_STATE_STICKY"})
		ewmh.WmDesktopSet(X, win.Id, 0xFFFFFFFF)
		strutP := ewmh.WmStrutPartial{}
		strut := ewmh.WmStrut{}
		if position == BOTTOM {
			strutP.BottomStartX = uint(geometry.X)
			strutP.BottomEndX = uint(geometry.X + geometry.Height)
			strutP.Bottom = uint(geometry.Height)
			strut.Bottom = uint(geometry.Height)
		} else {
			strutP.TopStartX = uint(geometry.X)
			strutP.TopEndX = uint(geometry.X + geometry.Height)
			strutP.Top = uint(geometry.Height)
			strut.Top = uint(geometry.Height)
		}
		ewmh.WmStrutPartialSet(X, win.Id, &strutP)
		ewmh.WmStrutSet(X, win.Id, &strut)

		windows[i] = win
	}

	return &Bar{
		X:          X,
		Windows:    windows,
		Geometries: geometries,
		Foreground: NewBGRA(fg),
		Background: NewBGRA(bg),
		Fonts:      fonts,
	}
}

// Draw draws TextPieces into X monitors.
func (self *Bar) Draw(text []*TextPiece) {
	imgs := make([]*xgraphics.Image, len(self.Windows))
	for i, geometry := range self.Geometries {
		imgs[i] = xgraphics.New(self.X, image.Rect(
			0, 0, int(geometry.Width), int(geometry.Height),
		))
		imgs[i].For(func(x, y int) xgraphics.BGRA { return *self.Background })
	}

	var err error
	xs := make([]int, len(self.Windows))
	for _, piece := range text {
		if piece.Background == nil {
			piece.Background = self.Background
		}
		if piece.Foreground == nil {
			piece.Foreground = self.Foreground
		}

		font := self.Fonts[piece.Font]
		width, _ := xgraphics.Extents(font.Font, font.Size, piece.Text)

		screens := piece.Screens
		if screens == nil {
			screens = make([]uint, len(imgs))
			for i := range imgs {
				screens[i] = uint(i)
			}
		}

		for i, screen := range screens {
			subimg := imgs[screen].SubImage(image.Rect(
				xs[screen], 0,
				xs[screen] + width, int(self.Geometries[screen].Height),
			))
			subimg.For(func(x, y int) xgraphics.BGRA { return *piece.Background })

			xs[screen], _, err = subimg.Text(
				xs[screen], 0,
				piece.Foreground, font.Size, font.Font, piece.Text,
			)
			if err != nil {
				log.Print(err) // TODO: Better logging
			}

			subimg.XDraw()
			subimg.XPaint(self.Windows[i].Id)
			subimg.Destroy()
		}
	}

	for i, img := range imgs {
		img.XSurfaceSet(self.Windows[i].Id)
		img.XDraw()
		img.Destroy()

		self.Windows[i].Map()
	}
}

// main gets command line arguments, creates X connection and initializes Bar.
// This is also where X event loop and Stdin reading lies.
func main() {
	cli := `gobar.

Usage:
  gobar [options]
  gobar -h | --help
  gobar --version

Options:
  -h --help              Show this screen.
  --bottom               Place bar at the bottom of the screen.
  --geometry=<GEOMETRY>  Comma separated list of monitor geometries in form of
                         <w>x<h>+<x>+<y>. If not specified, uses Mx16+0+0 for each screen.
                         M is a special case for <w> only, taking all available space.
  --fonts=<FONTS>        Comma seperated list of fonts in form of path:size.
                         [default: /usr/share/fonts/TTF/LiberationMono-Regular.ttf:12].
  --fg=<COLOR>           Foreground color (0xAARRGGBB) [default: 0xFFFFFFFF].
  --bg=<COLOR>           Background color (0xAARRGGBB) [default: 0xFF000000].
	`

	arguments, err := docopt.Parse(cli, nil, true, "", false)
	fatal(err)
	fgColor, err := strconv.ParseUint(arguments["--fg"].(string), 0, 32)
	fatal(err)
	bgColor, err := strconv.ParseUint(arguments["--bg"].(string), 0, 32)
	fatal(err)
	bottom := arguments["--bottom"].(bool)
	position := TOP
	if bottom {
		position = BOTTOM
	}
	fontSpecs := strings.Split(arguments["--fonts"].(string), ",")
	fonts := make([]*Font, len(fontSpecs))
	for i, fontSpec := range fontSpecs {
		fontSpecSplit := strings.Split(fontSpec, ":")
		fontPath := fontSpecSplit[0]
		fontSize, _ := strconv.ParseFloat(fontSpecSplit[1], 32)
		fonts[i] = NewFont(fontPath, fontSize)
	}

	X, err := xgbutil.NewConn()
	fatal(err)
	heads, err := xinerama.PhysicalHeads(X)
	fatal(err)

	geometryStr := arguments["--geometry"]
	var geometrySpec []string
	if geometryStr != nil {
		geometrySpec = strings.Split(geometryStr.(string), ",")
	}
	if len(geometrySpec) < len(heads) {
		for i := len(geometrySpec); i < len(heads); i++ {
			geometrySpec = append(
				geometrySpec, "Mx16+0+0",
			)
		}
	}

	geometries := make([]*Geometry, len(heads))
	for i, head := range heads {
		geometries[i] = NewGeometry(geometrySpec[i], head, position)
	}

	bar := NewBar(X, geometries, position, fgColor, bgColor, fonts)
	parser := NewTextParser()

	stdin := make(chan []*TextPiece, 0)
	go func() {
		defer close(stdin)

		for {
			parsed, err := parser.Scan(os.Stdin)
			if err != nil {
				log.Print(err) //TODO: Better logging
				continue
			}
			stdin <- parsed
		}
	}()

	_, _, pingQuit := xevent.MainPing(X)
	for {
		select {
		case text := <-stdin:
			bar.Draw(text)
		case <-pingQuit:
			return
		}
	}
}
