package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const (
	VERSION = `0.1.0`
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

var BARCODES = map[string]BarcodeDefinition{
	"upc":    {"\x00", 11, 12, []ByteRange{{48, 57}}},
	"ean13":  {"\x02", 12, 13, []ByteRange{{48, 57}}},
	"ean8":   {"\x03", 7, 8, []ByteRange{{48, 57}}},
	"code39": {"\x04", 1, 0, []ByteRange{{48, 57}, {65, 90}, {32, 32}, {36, 37}, {43, 43}, {45, 47}}},
}

var ESCPOS = map[string]TagDefintion{
	"b":      {"\x1b\x45\x01", "\x1b\x45\x00", false, nil},
	"u":      {"\x1b\x2d\x01", "\x1b\x2d\x00", false, nil},
	"uu":     {"\x1b\x2d\x02", "\x1b\x2d\x00", false, nil},
	"small":  {"\x1b\x4d\x01", "\x1b\x4d\x00", false, nil},
	"center": {"\x1b\x61\x01", "\n\x1b\x61\x00", true, nil},
	"right":  {"\x1b\x61\x02", "\n\x1b\x61\x00", true, nil},
	"tall":   {"\x1b\x21\x10", "\x1b\x21\x00", false, nil},
	"wide":   {"\x1b\x21\x20", "\x1b\x21\x00", false, nil},
	"huge":   {"\x1b\x21\x30", "\x1b\x21\x00", false, nil},
	"barcode": {"", "", false, func(e xml.StartElement) string {
		name := getAttr(e, "mode", true)
		mode, ok := BARCODES[name]
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
		return fmt.Sprintf("\x1d\x6b%s%s\x00", mode.Code, value)
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
		log.Printf("%T -> '%v': %s\n", tkn, tkn, tkn)
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