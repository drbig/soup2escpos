package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	VERSION = `0.0.1`
)

var build = `UNKNOWN` // injected via Makefile

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
	for {
		tkn, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalln("Error reading next token:", err)
		}
		fmt.Printf("%T -> '%v': %s\n", tkn, tkn, tkn)
	}
}
