package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	VERSION = `0.6.0`
)

var build = `UNKNOWN` // injected via Makefile

type TagDefintion struct {
	StartBytes      string
	EndBytes        string
	EatsNextNewline bool
	ProcFunc        func(xml.StartElement) string
}

type ByteRange struct {
	Min byte
	Max byte
}

type BarcodeDefinition struct {
	Code        string
	LenMin      int
	LenMax      int
	ValidRanges []ByteRange
}

var BARCODE = map[string]BarcodeDefinition{
	"upc":    {"\x00", 11, 12, []ByteRange{{48, 57}}},
	"ean13":  {"\x02", 12, 13, []ByteRange{{48, 57}}},
	"ean8":   {"\x03", 7, 8, []ByteRange{{48, 57}}},
	"code39": {"\x04", 1, 0, []ByteRange{{48, 57}, {65, 90}, {32, 32}, {36, 37}, {43, 43}, {45, 47}}},
}

var BARCODE_HRI_POS = map[string]string{
	"none":  "\x00",
	"above": "\x01",
	"below": "\x02",
	"both":  "\x03",
}

var BARCODE_HRI_FONT = map[string]string{
	"small":  "\x01",
	"normal": "\x00",
}

const PAPER_WIDTH_IN = 2 // for images width sanity check only

type ImageDefintion struct {
	Code   string
	HorDPI int
}

var IMG_MODE = map[string]ImageDefintion{
	"normal": {"\x00", 180},
	"wide":   {"\x01", 90},
	"tall":   {"\x02", 180},
	"huge":   {"\x03", 90},
}

var ESCPOS = map[string]TagDefintion{
	"b":      {"\x1b\x45\x01", "\x1b\x45\x00", false, nil},
	"u":      {"\x1b\x2d\x01", "\x1b\x2d\x00", false, nil},
	"uu":     {"\x1b\x2d\x02", "\x1b\x2d\x00", false, nil},
	"inv":    {"\x1d\x42\x01", "\x1d\x42\x00", false, nil},
	"small":  {"\x1b\x4d\x01", "\x1b\x4d\x00", false, nil},
	"center": {"\x1b\x61\x01", "\n\x1b\x61\x00", true, nil},
	"right":  {"\x1b\x61\x02", "\n\x1b\x61\x00", true, nil},
	"tall":   {"\x1b\x21\x10", "\x1b\x21\x00", false, nil},
	"wide":   {"\x1b\x21\x20", "\x1b\x21\x00", false, nil},
	"huge":   {"\x1b\x21\x30", "\x1b\x21\x00", false, nil},
	"barcode": {"", "", false, func(e xml.StartElement) string {
		name := getAttr(e, "mode", true)
		mode, ok := BARCODE[name]
		if !ok {
			log.Fatalln("No supported barcode type:", name)
		}
		value := []byte(getAttr(e, "value", true))
		if len(value) < mode.LenMin {
			log.Fatalln("Value under minimum length of", mode.LenMin)
		}
		if mode.LenMax > 0 && (len(value) > mode.LenMax) {
			log.Fatalln("Value over maximum length of", mode.LenMax)
		}
		for p, b := range value {
			ok := false
			for _, br := range mode.ValidRanges {
				if (b >= br.Min) && (b <= br.Max) {
					ok = true
					break
				}
			}
			if !ok {
				log.Fatalf("Byte at pos %d is out of valid range\n", p)
			}
		}
		var preCodes, postCodes strings.Builder
		height := getAttr(e, "height", false)
		if height != "" {
			hint, err := strconv.Atoi(height)
			if err != nil {
				log.Fatalln("Failed to convert 'height' attr:", err)
			}
			h := byte(hint)
			if (h < 8) && (h > 162) {
				log.Fatalln("Height out of range")
			}
			//log.Println("Barcode will be of height:", hint)
			preCodes.WriteString("\x1d\x68")
			preCodes.WriteByte(h)
			postCodes.WriteString("\x1d\x68\xa2") // reset height to default 162
		}
		hripos := getAttr(e, "hri_pos", false)
		if hripos != "" {
			code, ok := BARCODE_HRI_POS[hripos]
			if !ok {
				log.Fatalln("Unsupported HRI position:", hripos)
			}
			preCodes.WriteString("\x1d\x48")
			preCodes.WriteString(code)
			postCodes.WriteString("\x1d\x48\x02") // reset to default below
		}
		hrifnt := getAttr(e, "hri_font", false)
		if hrifnt != "" {
			code, ok := BARCODE_HRI_FONT[hrifnt]
			if !ok {
				log.Fatalln("Unsupported HRI font:", hrifnt)
			}
			preCodes.WriteString("\x1d\x66")
			preCodes.WriteString(code)
			postCodes.WriteString("\x1d\x66\x00")
		}
		return fmt.Sprintf("%s\x1d\x6b%s%s\x00%s", preCodes.String(), mode.Code, value, postCodes.String())
	}},
	"img": {"", "", false, func(e xml.StartElement) string {
		src := getAttr(e, "src", true)
		ifh, err := os.Open(src)
		if err != nil {
			log.Fatalln("Error opening image file:", err)
		}
		defer ifh.Close()
		cfg, err := png.DecodeConfig(ifh)
		if err != nil {
			log.Fatalln("Error decoding config:", err)
		}
		var mode ImageDefintion
		var ok bool
		mname := getAttr(e, "mode", false)
		if mname == "" {
			mode = IMG_MODE["normal"]
		} else {
			mode, ok = IMG_MODE[mname]
			if !ok {
				log.Fatalln("No such image mode:", mname)
			}
		}
		maxWidth := PAPER_WIDTH_IN * mode.HorDPI
		if cfg.Width > maxWidth {
			log.Fatalln("Image width of", cfg.Width, "exceeds calculated max of", maxWidth)
		}
		if cfg.Height > 65535 { // max two bytes can hold
			log.Fatalln("Image height of", cfg.Height, "exceeds max of 0xffff")
		}
		ifh.Seek(0, io.SeekStart)
		img, err := png.Decode(ifh)
		if err != nil {
			log.Fatalln("Error decoding image:", err)
		}
		hX := ((uint16(cfg.Width) + 7) / 8)
		hY := uint16(cfg.Height)
		var code strings.Builder
		code.WriteString("\x1d\x76\x30")
		code.WriteString(mode.Code)
		code.WriteByte(byte(hX & 0xff))
		code.WriteByte(byte((hX >> 8) & 0xff))
		code.WriteByte(byte(hY & 0xff))
		code.WriteByte(byte((hY >> 8) & 0xff))
		var mask, i, current uint8
		mask = 0x80
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				c := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
				if c.Y < 128 {
					current |= mask
				}
				mask = mask >> 1
				i++
				if i == 8 {
					code.WriteByte(byte(current))
					mask = 0x80
					i = 0
					current = 0
				}
			}
			if i != 0 {
				code.WriteByte(byte(current))
				i = 0
			}
		}
		return code.String()
	}},
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s [file_path]
soup2escpos v%s by Piotr S. Staszewski, see LICENSE.txt
binary build by %s

file_path - path to file to process, otherwise will read from stdin
`, os.Args[0], VERSION, build)
	}
}

func main() {
	var err error
	var ifh *os.File

	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}
	ifh = os.Stdin
	if flag.NArg() == 1 {
		ifh, err = os.Open(flag.Arg(0))
		if err != nil {
			log.Fatalln("Error opening input file:", err)
		}
	}
	defer ifh.Close()
	decoder := xml.NewDecoder(ifh)
	enl := false // eat next new line
	for {
		tkn, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalln("Error reading next token:", err)
		}
		//log.Printf("%T -> '%v': %s\n", tkn, tkn, tkn)
		switch tt := tkn.(type) {
		case xml.StartElement:
			tag := getTagDef(tt.Name.Local)
			if tag.ProcFunc != nil {
				fmt.Print(tag.ProcFunc(tt))
			} else {
				fmt.Print(tag.StartBytes)
			}
		case xml.EndElement:
			tag := getTagDef(tt.Name.Local)
			fmt.Print(tag.EndBytes)
			enl = tag.EatsNextNewline
		case xml.CharData:
			start := 0
			if enl && tt[0] == 0x0A {
				start = 1
				enl = false
			}
			fmt.Printf("%s", tt[start:])
		case xml.Comment:
		default:
			log.Fatalf("Unhandled token type: %T\n", tkn)
		}
	}
}

func getTagDef(rawName string) TagDefintion {
	name := strings.ToLower(rawName)
	def, ok := ESCPOS[name]
	if !ok {
		log.Fatalln("Unknown token:", name)
	}
	return def
}

func getAttr(e xml.StartElement, name string, required bool) (mode string) {
	for _, a := range e.Attr {
		if strings.ToLower(a.Name.Local) == name {
			mode = a.Value
			break
		}
	}
	if (mode == "") && required {
		log.Fatalf("Required attr '%s' not found\n", name)
	}
	return mode
}
