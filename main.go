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
	VERSION = `0.0.2`
)

var build = `UNKNOWN` // injected via Makefile

type TagDefintion struct {
	StartBytes      string
	EndBytes        string
	EatsNextNewline bool
}

var ESCPOS = map[string]TagDefintion{
	"b":      {"\x1b\x45\x01", "\x1b\x45\x00", false},
	"u":      {"\x1b\x2d\x01", "\x1b\x2d\x00", false},
	"uu":     {"\x1b\x2d\x02", "\x1b\x2d\x00", false},
	"small":  {"\x1b\x4d\x01", "\x1b\x4d\x00", false},
	"center": {"\x1b\x61\x01", "\n\x1b\x61\x00", true},
	"right":  {"\x1b\x61\x02", "\n\x1b\x61\x00", true},
	"tall":   {"\x1b\x21\x10", "\x1b\x21\x00", false},
	"wide":   {"\x1b\x21\x20", "\x1b\x21\x00", false},
	"huge":   {"\x1b\x21\x30", "\x1b\x21\x00", false},
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
			fmt.Print(getTagDef(tt.Name.Local).StartBytes)
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
