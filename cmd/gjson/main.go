package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/tidwall/gjson"
)

/*Commands The list of commands supported by this cli.*/
var Commands = []string{"help", "get"}

const description = `
Query and print to stdout values from a json file.

commands:
	help - 	display this prompt
	
	get - 	Query a JSON object from a file with a path in 'dot syntax' (see readme)
		usage: get -j FILE.json -q DOT.PATH
`

func main() {

	args := os.Args

	if len(args) < 2 {
		log.Fatal("Must specify a command. ", Commands)
	}

	switch args[1] {
	case "get":
		get()
	case "help":
		help()
	default:
		log.Fatal("Invalid command. ", Commands)
	}

}

func help() {
	fmt.Println(description)
}

func get() {
	os.Args = os.Args[1:]
	file := flag.String("j", "", "A json file containing valid JSON.")
	query := flag.String("q", "", "The json query you want to preform")
	flag.Parse()

	if *file == "" {
		log.Fatal("json file required. -j")
	}

	if *query == "" {
		log.Fatal("query required. -q")
	}

	json, err := ioutil.ReadFile(*file)

	if err != nil {
		log.Fatal("Something is wrong with that file.")
	}

	if !gjson.Valid(string(json)) {
		log.Fatal("That JSON file contains invalid JSON.")
	}

	value := gjson.Get(string(json), *query)
	println(value.String())
}
