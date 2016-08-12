<p align="center">
<img 
    src="logo.png" 
    width="240" height="78" border="0" alt="GJSON">
<br>
<a href="https://travis-ci.org/tidwall/gjson"><img src="https://img.shields.io/travis/tidwall/gjson.svg?style=flat-square" alt="Build Status"></a><!--
<a href="http://gocover.io/github.com/tidwall/gjson"><img src="https://img.shields.io/badge/coverage-97%25-brightgreen.svg?style=flat-square" alt="Code Coverage"></a>
-->
<a href="https://godoc.org/github.com/tidwall/gjson"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>
</p>

<p align="center">get a json value quickly</a></p>

GJSON is a Go package the provides a [very fast](#performance) and simple way to get a value from a json document. The reason for this library it to give efficient json indexing for the [BuntDB](https://github.com/tidwall/buntdb) project. 

Getting Started
===============

## Installing

To start using GJSON, install Go and run `go get`:

```sh
$ go get -u github.com/tidwall/gjson
```

This will retrieve the library.

## Get a value
Get searches json for the specified path. A path is in dot syntax, such as "name.last" or "age". This function expects that the json is well-formed and validates. Invalid json will not panic, but it may return back unexpected results. When the value is found it's returned immediately.

```go
package main

import "github.com/tidwall/gjson"

const json = `{"name":{"first":"Janet","last":"Prichard"},"age":47}`

func main() {
	value := gjson.Get(json, "name.last")
	println(value.String())
}
```

This will print:

```
Prichard
```

A path is a series of keys separated by a dot. 
A key may contain special wildcard characters '\*' and '?'. 
To access an array value use the index as the key. 
To get the number of elements in an array use the '#' character.
The dot and wildcard characters can be escaped with '\'.
```
{
  "name": {"first": "Tom", "last": "Anderson"},
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter"
}
"name.last"          >> "Anderson"
"age"                >> 37
"children.#"         >> 3
"children.1"         >> "Alex"
"child*.2"           >> "Jack"
"c?ildren.0"         >> "Sara"
"fav\.movie"         >> "Deer Hunter"
```

## Result Type

GJSON supports the json types `string`, `number`, `bool`, and `null`. 
Arrays and Objects are returned as their raw json types. 

The `Result` type holds one of these:

```
bool, for JSON booleans
float64, for JSON numbers
string, for JSON string literals
nil, for JSON null
```

To get the Go value call the `Value()` method:


```go
result.Value()  // interface{} which may be nil, string, float64, or bool

// Or just get the value in one step.
gjson.Get(json, "name.last").Value()
```

To directly access the value:

```go
result.Type    // can be String, Number, True, False, Null, or JSON
result.Str     // holds the string
result.Num     // holds the float64 number
result.Raw     // holds the raw json
```

## Check for the existence of a value

Sometimes you may want to see if the value actually existed in the json document.

```
value := gjson.Get(json, "name.last")
if !value.Exists() {
	println("no last name")
} else {
	println(value.String())
}

// Or as one step
if gjson.Get(json, "name.last").Exists(){
	println("has a last name")
}
```


## Performance

Benchmarks of GJSON alongside [encoding/json](https://golang.org/pkg/encoding/json/), 
[ffjson](https://github.com/pquerna/ffjson), 
[EasyJSON](https://github.com/mailru/easyjson),
and [jsonparser](https://github.com/buger/jsonparser)

```
BenchmarkGJSONGet-8             3000000    440 ns/op     0 B/op    0 allocs/op
BenchmarkJSONUnmarshalMap-8      600000  10738 ns/op  3176 B/op   69 allocs/op
BenchmarkJSONUnmarshalStruct-8   600000  11635 ns/op  1960 B/op   69 allocs/op
BenchmarkJSONDecoder-8           300000  17193 ns/op  4864 B/op  184 allocs/op
BenchmarkFFJSONLexer-8          1500000   3773 ns/op  1024 B/op    8 allocs/op
BenchmarkEasyJSONLexer-8        3000000   1134 ns/op   741 B/op    6 allocs/op
BenchmarkJSONParserGet-8        3000000    596 ns/op    21 B/op    0 allocs/op
```

JSON document used:

```json
{
  "widget": {
    "debug": "on",
    "window": {
      "title": "Sample Konfabulator Widget",
      "name": "main_window",
      "width": 500,
      "height": 500
    },
    "image": { 
      "src": "Images/Sun.png",
      "hOffset": 250,
      "vOffset": 250,
      "alignment": "center"
    },
    "text": {
      "data": "Click Here",
      "size": 36,
      "style": "bold",
      "vOffset": 100,
      "alignment": "center",
      "onMouseUp": "sun1.opacity = (sun1.opacity / 100) * 90;"
    }
  }
}    
```

Each operation was rotated though one of the following search paths:

```
widget.window.name
widget.image.hOffset
widget.text.onMouseUp
```


*These are the results from running the benchmarks on a MacBook Pro 15" 2.8 GHz Intel Core i7:*

## Contact
Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

GJSON source code is available under the MIT [License](/LICENSE).
