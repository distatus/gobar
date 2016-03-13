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
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/freetype-go/freetype/truetype"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/BurntSushi/xgbutil/xinerama"
	"github.com/BurntSushi/xgbutil/xwindow"
)

// fatal is a helper function to call when something terribly wrong
// had happened. Logs given error and terminates application.
func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func contains(slice []uint, item uint) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// headsEqual Checks whether Rects contained in xinerama.Heads are all equal.
func headsEqual(h1, h2 xinerama.Heads) bool {
	if len(h1) != len(h2) {
		return false
	}
	for i, h := range h1 {
		x1, y1, w1, h1 := h.Pieces()
		x2, y2, w2, h2 := h2[i].Pieces()
		if x1 != x2 || y1 != y2 || w1 != w2 || h1 != h2 {
			return false
		}
	}
	return true
}

// Position defines bar placement on the screen.
type Position uint8

const (
	BOTTOM Position = iota
	TOP
)

// Font stores font definition along with it's loaded truetype struct.
type Font struct {
	Path string
	Size float64
	Font *truetype.Font
}

func (f *Font) String() string {
	return fmt.Sprintf("%s:%f", f.Path, f.Size)
}

// FindFontError is returned when FindFontPath fails to fint any usable fonts.
type FindFontError struct {
	Action string
	Orig   error
}

func (f FindFontError) Error() string {
	return fmt.Sprintf("[fontconfig] Could not %s. Got `%s`", f.Action, f.Orig)
}

type FontFinder func() (string, error)

// FindFontPath tries hard to find any usable font(s) in the system.
// It does so by parsing fontconfig configuration and looking through
// the specified directories for anything that could possibly by used
// by freetype font parser.
func FindFontPath() (string, error) {
	log.Print("Trying to find usable font")

	fontsConf, err := os.Open("/etc/fonts/fonts.conf")
	if err != nil {
		return "", FindFontError{"open file", err}
	}
	defer fontsConf.Close()

	result := struct {
		Dirs []string `xml:"dir"`
	}{}
	decoder := xml.NewDecoder(fontsConf)
	if err := decoder.Decode(&result); err != nil {
		return "", FindFontError{"decode file", err}
	}

	for _, dir := range result.Dirs {
		files, err := filepath.Glob(fmt.Sprintf("%s/TTF/*.ttf", dir))
		if err == nil && files != nil {
			return files[0], nil
		}
	}
	return "", FindFontError{"find font files", err}
}

// FontError is returned when NewFont fails to create a font.
type FontError struct {
	Path string
	Orig error
}

func (f FontError) Error() string {
	return fmt.Sprintf("Could not open font `%s`. Got `%s`", f.Path, f.Orig)
}

type FontCreator func(path string, size float64) (*Font, error)

// NewFont opens a font file and parses it with truetype engine.
func NewFont(path string, size float64) (*Font, error) {
	fontReader, err := os.Open(path)
	if err != nil {
		return nil, FontError{path, err}
	}
	defer fontReader.Close()

	font, err := xgraphics.ParseFont(fontReader)
	if err != nil {
		return nil, FontError{path, err}
	}
	return &Font{path, size, font}, nil
}

// Geometry stores bars geometry on the screen (or actually monitor).
type Geometry struct {
	Width  uint16
	Height uint16
	X      uint16
	Y      uint16
}

func (g *Geometry) String() string {
	return fmt.Sprintf("%dx%d+%d+%d", g.Width, g.Height, g.X, g.Y)
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

	heads xinerama.Heads
}

// NewBar creates X windows for every monitor.
// Also sets proper EWMH information for docked windows and
// deals with dynamic geometry changes.
func NewBar(
	X *xgbutil.XUtil, geometries []*Geometry, position Position,
	fg uint64, bg uint64, fonts []*Font,
) *Bar {
	heads, err := xinerama.PhysicalHeads(X)
	fatal(err)

	bar := &Bar{
		X:          X,
		Windows:    []*xwindow.Window{},
		Geometries: []*Geometry{},
		Foreground: NewBGRA(fg),
		Background: NewBGRA(bg),
		Fonts:      fonts,
		heads:      heads,
	}

	bar.create(geometries, position)

	xproto.ChangeWindowAttributesChecked(
		X.Conn(), X.RootWin(), xproto.CwEventMask,
		[]uint32{xproto.EventMaskStructureNotify},
	)
	xevent.ConfigureNotifyFun(func(_ *xgbutil.XUtil, _ xevent.ConfigureNotifyEvent) {
		heads, err = xinerama.PhysicalHeads(X)
		if err != nil {
			log.Printf("Error `%s` getting updated heads, staying with the old ones\n", err)
			return
		}
		if !headsEqual(heads, bar.heads) {
			bar.destroy()
			bar.heads = heads
			bar.create(geometries, position)
		}
	}).Connect(X, X.RootWin())

	return bar
}

// destroy Destroys all existing windows and resets geometries.
func (b *Bar) destroy() {
	for i, window := range b.Windows {
		window.Destroy()
		b.Windows[i] = nil
	}
	b.Windows = []*xwindow.Window{}
	b.Geometries = []*Geometry{}
}

func (b *Bar) create(geometries []*Geometry, position Position) {
	maxHeight := xwindow.RootGeometry(b.X).Height()

	if len(geometries) == 0 {
		geometries = append(geometries, &Geometry{Height: 16})
	}
	for i, head := range b.heads {
		var geometry *Geometry
		if i >= len(geometries) {
			if geometries[len(geometries)-1] == nil {
				break
			}
			geometry = geometries[len(geometries)-1]
		} else {
			if geometries[i] == nil {
				continue
			}
			geometry = geometries[i]
		}
		win, err := xwindow.Generate(b.X)
		if err != nil {
			log.Printf("Could not generate window for geometry `%s`", geometry)
			continue
		}

		width := int(geometry.Width)
		if width == 0 {
			width = head.Width()
		}
		height := int(geometry.Height)
		if height == 0 {
			height = head.Height()
		}
		y := int(geometry.Y)

		strutP := ewmh.WmStrutPartial{}
		strut := ewmh.WmStrut{}
		if position == BOTTOM {
			y = head.Height() - height - y
			bottom := uint(maxHeight - y)

			strutP.BottomStartX = uint(geometry.X)
			strutP.BottomEndX = uint(geometry.X + uint16(width))
			strutP.Bottom = bottom
			strut.Bottom = bottom
		} else {
			strutP.TopStartX = uint(geometry.X)
			strutP.TopEndX = uint(geometry.X + uint16(width))
			strutP.Top = uint(height)
			strut.Top = uint(height)
		}

		win.Create(
			b.X.RootWin(),
			int(geometry.X)+head.X(),
			y+head.Y(),
			width, height, 0,
		)

		ewmh.WmWindowTypeSet(b.X, win.Id, []string{"_NET_WM_WINDOW_TYPE_DOCK"})
		ewmh.WmStateSet(b.X, win.Id, []string{"_NET_WM_STATE_STICKY"})
		ewmh.WmDesktopSet(b.X, win.Id, 0xFFFFFFFF)
		ewmh.WmStrutPartialSet(b.X, win.Id, &strutP)
		ewmh.WmStrutSet(b.X, win.Id, &strut)

		b.Windows = append(b.Windows, win)
		b.Geometries = append(b.Geometries, &Geometry{
			X:      geometry.X,
			Y:      uint16(y),
			Width:  uint16(width),
			Height: uint16(height),
		})
	}
}

// Draw draws TextPieces into X monitors.
func (b *Bar) Draw(text []*TextPiece) {
	imgs := make([]*xgraphics.Image, len(b.Windows))
	for i, geometry := range b.Geometries {
		imgs[i] = xgraphics.New(b.X, image.Rect(
			0, 0, int(geometry.Width), int(geometry.Height),
		))
		imgs[i].For(func(x, y int) xgraphics.BGRA { return *b.Background })
	}

	xsl := make([]int, len(b.Windows))
	xsr := make([]int, len(b.Windows))
	for i := range xsr {
		xsr[i] = int(b.Geometries[i].Width)
	}
	for _, piece := range text {
		if piece.Background == nil {
			piece.Background = b.Background
		}
		if piece.Foreground == nil {
			piece.Foreground = b.Foreground
		}

		if piece.Font > uint(len(b.Fonts))-1 {
			log.Printf("Invalid font index `%d`, using 0", piece.Font)
			piece.Font = 0
		}
		font := b.Fonts[piece.Font]
		width, _ := xgraphics.Extents(font.Font, font.Size, piece.Text)

		screens := []uint{}
		if piece.Screens == nil {
			for i := range imgs {
				if !contains(piece.NotScreens, uint(i)) {
					screens = append(screens, uint(i))
				}
			}
		} else {
			for _, screen := range piece.Screens {
				if int(screen) < len(xsl) && !contains(piece.NotScreens, screen) {
					screens = append(screens, screen)
				}
			}
		}

		for _, screen := range screens {
			xs := xsl[screen]
			if piece.Align == RIGHT {
				xs = xsr[screen] - width
			}

			subimg := imgs[screen].SubImage(image.Rect(
				xs, 0, xs+width, int(b.Geometries[screen].Height),
			))
			if subimg == nil {
				log.Printf(
					"Cannot create Subimage for coords `%dx%dx%dx%d`\n",
					xs, 0, xs+width, int(b.Geometries[screen].Height),
				)
				continue
			}
			subximg := subimg.(*xgraphics.Image)

			subximg.For(func(x, y int) xgraphics.BGRA { return *piece.Background })

			xsNew, _, err := subximg.Text(
				xs, 0, piece.Foreground, font.Size, font.Font, piece.Text,
			)
			if err != nil {
				log.Print(err) // TODO: Better logging
			}

			if piece.Align == LEFT {
				xsl[screen] = xsNew
			} else if piece.Align == RIGHT {
				xsr[screen] -= width
			}

			subximg.XPaint(b.Windows[screen].Id)
			subximg.Destroy()
		}
	}

	for i, img := range imgs {
		img.XSurfaceSet(b.Windows[i].Id)
		img.XDraw()
		img.XPaint(b.Windows[i].Id)
		img.Destroy()

		b.Windows[i].Map()
	}
}

// ParseFonts parses a list of stringified font definitions
// into a list of Font structures.
// Also handles all kinds of bad input and tries hard to recover from it.
// Returns error only if not a single usable font is found in the end.
func ParseFonts(
	fontSpecs []string, createFont FontCreator, findFont FontFinder,
) (fonts []*Font, err error) {
	fonts = make([]*Font, 0, len(fontSpecs))
	fontSize := 12.0
	for _, fontSpec := range fontSpecs {
		fontSpecSplit := strings.Split(fontSpec, ":")
		fontPath := fontSpecSplit[0]
		fontSize = 12.0
		if len(fontSpecSplit) < 2 {
			log.Printf("No font size for `%s`, using `12`", fontPath)
		} else {
			fontSizeStr := fontSpecSplit[1]
			possibleFontSize, err := strconv.ParseFloat(fontSizeStr, 32)
			if err == nil {
				fontSize = possibleFontSize
			} else {
				log.Printf(
					"Invalid font size `%s` for `%s`, using `12`. Got `%s`",
					fontSizeStr, fontPath, err,
				)
			}
		}
		font, err := createFont(fontPath, fontSize)
		if err != nil {
			log.Print(err)
		} else {
			fonts = append(fonts, font)
		}
	}
	if len(fonts) == 0 {
		fontPath, err := findFont()
		if err != nil {
			return fonts, err
		}

		font, err := createFont(fontPath, fontSize)
		if err != nil {
			return fonts, err
		}
		fonts = append(fonts, font)
	}
	return
}

type Fonts []*Font

func (f *Fonts) String() string {
	str := make([]string, len(*f))
	for i, f := range *f {
		str[i] = f.String()
	}
	return fmt.Sprintf("%q", strings.Join(str, ","))
}

func (f *Fonts) Set(value string) error {
	fonts, err := ParseFonts(strings.Split(value, ","), NewFont, FindFontPath)
	if err != nil {
		return fmt.Errorf("No usable fonts found")
	}
	*f = append(*f, fonts...)
	return nil
}

type Geometries []*Geometry

func (g *Geometries) String() string {
	str := make([]string, len(*g))
	for i, g := range *g {
		str[i] = g.String()
	}
	j := strings.Join(str, ",")
	if j == "" {
		j = "0x16+0+0"
	}
	return fmt.Sprintf("%q", j)
}

func (g *Geometries) Set(value string) error {
	if len(*g) > 0 {
		return fmt.Errorf("geometries flag already set")
	}
	if value == "" {
		return nil
	}
	for _, geometry := range strings.Split(value, ",") {
		if geometry == "" {
			*g = append(*g, nil)
		} else {
			geom := &Geometry{}
			_, err := fmt.Sscanf(
				geometry, "%dx%d+%d+%d",
				&geom.Width, &geom.Height, &geom.X, &geom.Y,
			)
			if err != nil {
				geom = &Geometry{Height: 16}
				log.Printf("Bad geometry `%s`, using default", geometry)
			}
			*g = append(*g, geom)
		}
	}
	return nil
}

// main gets command line arguments, creates X connection and initializes Bar.
// This is also where X event loop and Stdin reading lies.
func main() {
	bottom := flag.Bool("bottom", false, "Place bar at the bottom of the screen")
	fgColor := flag.Uint64("fg", 0xFFFFFFFF, "Foreground color (0xAARRGGBB)")
	flag.Lookup("fg").DefValue = "0xFFFFFFFF"
	bgColor := flag.Uint64("bg", 0xFF000000, "Background color (0xAARRGGBB)")
	flag.Lookup("bg").DefValue = "0xFF000000"
	var fonts Fonts
	flag.Var(&fonts, "fonts", "Comma separated list of fonts in form of path[:size]")
	var geometries Geometries
	flag.Var(&geometries, "geometries", "Comma separated list of monitor geometries (<w>x<h>+<x>+<y>), for <w> and <h>, 0 means 100%")
	flag.Parse()

	position := TOP
	if *bottom {
		position = BOTTOM
	}

	X, err := xgbutil.NewConn()
	fatal(err)

	bar := NewBar(X, geometries, position, *fgColor, *bgColor, fonts)
	parser := NewTextParser()

	stdin := make(chan []*TextPiece)
	go func() {
		defer close(stdin)
		reader := bufio.NewReader(os.Stdin)

		for {
			str, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Error reading stdin. Got `%s`", err)
			} else {
				stdin <- parser.Scan(strings.NewReader(str))
			}
		}
	}()

	pingBefore, pingAfter, pingQuit := xevent.MainPing(X)
	for {
		select {
		case <-pingBefore:
			<-pingAfter
		case text := <-stdin:
			bar.Draw(text)
		case <-pingQuit:
			return
		}
	}
}
