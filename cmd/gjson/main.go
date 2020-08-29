package main

import (
	"flag"
	"fmt"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	logerr    *log.Logger
	query     string
	delimiter string
	quote     bool
	inputs    []string
	errorcode int
)

func configure() {
	logerr = log.New(os.Stdout, "gjson: ", 0)
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), "gjson [-d DELIMITER] [-q] QUERY [FILE...]\n")
		flag.PrintDefaults()
	}

	quoteflag := flag.Bool("q", false, "add quotations around objects in an array")
	delimflag := flag.String("d", ", ", "delimiter between objects in an array")
	flag.Parse()
	args := flag.Args()

	quote = *quoteflag
	delimiter = *delimflag

	if len(args) < 1 {
		logerr.Println("Query not provided")
		flag.Usage()
		os.Exit(1)
	}

	query = args[0]

	if len(args) > 1 {
		inputs = args[1:]
		return
	}

	waiting, err := stdinWaiting()
	if err != nil {
		logerr.Fatalf("Error checking for stdin: %s", err.Error())
	}
	if !waiting {
		logerr.Println("No files to process")
		flag.Usage()
		os.Exit(1)
	}
}

func stdinWaiting() (result bool, err error) {
	var instat os.FileInfo
	instat, err = os.Stdin.Stat()
	result = err == nil && instat.Mode()&os.ModeNamedPipe != 0
	return
}

func main() {
	configure()

	if len(inputs) < 1 {
		if err := process(os.Stdin); err != nil {
			logerr.Fatalf("Processing stdin: %s", err.Error())
		}
		return
	}

	for _, input := range inputs {
		fo, err := os.Open(input)
		if err != nil {
			logerr.Printf("Error opening file %q: %s", input, err.Error())
			errorcode = 2
			continue
		}
		defer fo.Close()
		if perr := process(fo); perr != nil {
			logerr.Printf("Error processing %q: %s", input, perr.Error())
			errorcode = 3
		}
	}
	os.Exit(errorcode)
}

func process(input io.Reader) error {
	all, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	result := gjson.GetBytes(all, query)

	if !result.IsArray() {
		fmt.Println(result)
		return nil
	}

	resulta := result.Array()
	results := make([]string, len(resulta))

	for i := range results {
		if quote {
			results[i] = fmt.Sprintf("%q", resulta[i].String())
		} else {
			results[i] = resulta[i].String()
		}
	}

	fmt.Println(strings.Join(results, delimiter))
	return nil
}
