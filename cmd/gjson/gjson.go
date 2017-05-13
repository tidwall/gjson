package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/tidwall/gjson"
)

const version = "0.0.1"

var options struct {
	in          string
	out         string
	includePath bool
	version     bool
}

func readInput() ([]byte, error) {
	if len(options.in) > 0 {
		return ioutil.ReadFile(options.in)
	}

	// Check that stdin is not empty.
	stat, err := os.Stdin.Stat()
	if err != nil {
		return []byte{}, err
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return []byte{}, errors.New("expected JSON data from stdin (run gjson -help)")
	}

	return ioutil.ReadAll(os.Stdin)
}

func printOutput(results []gjson.Result) error {
	var (
		out *os.File
		err error
	)

	if len(options.out) > 0 {
		out, err = os.Create(options.out)
		if err != nil {
			return err
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	for i, result := range results {
		if options.includePath {
			fmt.Fprintf(out, "\"%s\" : ", flag.Args()[i])
		}
		fmt.Fprintln(out, result)
	}

	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] [path ...]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&options.in, "in", "", "read JSON data from this file instead of stdin")
	flag.StringVar(&options.out, "out", "", "write result to this file instead of stdout")
	flag.BoolVar(&options.includePath, "include-path", false, "include paths in output")
	flag.BoolVar(&options.version, "version", false, "print version and exit")
	flag.Parse()

	if options.version {
		fmt.Printf("gjson v%v\n", version)
		os.Exit(0)
	}

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	bytes, err := readInput()
	if err != nil {
		log.Fatal(err.Error())
	}

	results := gjson.GetMany(string(bytes), flag.Args()...)
	err = printOutput(results)
	if err != nil {
		log.Fatal(err.Error())
	}
}
