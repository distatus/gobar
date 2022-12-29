[![Build Status](https://github.com/distatus/gobar/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/distatus/gobar/actions/workflows/tests.yml)

**gobar** is a minimalistic X status bar written in pure Go.

Supports xinerama, EWMH, font antialiasing and possibly some other fancy looking names and shortcuts.

## screenshot

Two **gobar** instances, both fed by [osop](https://github.com/distatus/osop).

For detailed configuration see [my](https://github.com/KenjiTakahashi/dotfiles/blob/master/dotxprofile) [dotfiles](https://github.com/KenjiTakahashi/dotfiles/tree/master/dotconfig/osop).

![screenshot](http://img.kenji.sx/gobar_dual.png)

## installation

First, you have to [get Go](http://golang.org/doc/install). Note that version >= 1.1 is required.

Then, just

```bash
$ go get github.com/distatus/gobar
```

should get you going.

## usage

Command line options reference follows:

**-h --help** displays help message and exits.

**--bottom** places bar on bottom of the screen *(defaults to false)*.

**--geometries** takes comma separated list of monitor geometries *(defaults to `0x16+0+0`)*.

Each geometry is in form of `<width>x<height>+<x>+<y>`. If `<width>`/`<height>` is `0`, screen width/height is used.

If geometry is empty, bar is not drawn on a respective monitor.

If there are less geometries than monitors, last geometry is used for subsequent monitors.

**--fonts** takes comma separated list of fonts.

Each font element is in form of `<ttf file path>[:<font size>]`.

If omitted, or if incorrect path is specified, defaults to whatever it can find in fontconfig configuration.

If `<font size>` part is omitted or incorrect, defaults to `12`.

**--fg** takes main foreground color. Should be in form `0xAARRGGBB` *(defaults to `0xFFFFFFFF`)*.

**--bg** takes main background color. Should be in form `0xAARRGGBB` *(defaults to `0xFF000000`)*.

Other than that, an input string should be piped into the **gobar** executable.

A really simple example could be displaying current date and time.
```bash
$ while :; do date; sleep 1; done | gobar
```

Special tokens can also be used in the input string to allow nice formatting.

#### Input string formatting syntax

Each token should be preceded with `{` and will be active until `}`. Note that `{text}` is also treated as valid token and will output `text`. Escaping with `\` will print bracket(s) literally.

**F&lt;num&gt;** sets active font, **&lt;num&gt;** should be index of one of the elements from fonts list specified in **--fonts=**.

**S&lt;num&gt;,&lt;num&gt;...** specifies monitors to draw on. Multiple, comma separated, numbers can be specified. If not specified, draws to all available monitors. Negative number can be specified to set on which monitors to *not* draw.

**CF0xAARRGGBB** sets active foreground color.

**CB0xAARRGGBB** sets active background color.

**AR** aligns next text piece to the right.
