package gjson

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/mailru/easyjson/jlexer"
	fflib "github.com/pquerna/ffjson/fflib/v1"
)

// TestRandomData is a fuzzing test that throws random data at the Parse
// function looking for panics.
func TestRandomData(t *testing.T) {
	var lstr string
	defer func() {
		if v := recover(); v != nil {
			println("'" + hex.EncodeToString([]byte(lstr)) + "'")
			println("'" + lstr + "'")
			panic(v)
		}
	}()
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 200)
	for i := 0; i < 2000000; i++ {
		n, err := rand.Read(b[:rand.Int()%len(b)])
		if err != nil {
			t.Fatal(err)
		}
		lstr = string(b[:n])
		Get(lstr, "zzzz")
	}
}

func TestRandomValidStrings(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 200)
	for i := 0; i < 100000; i++ {
		n, err := rand.Read(b[:rand.Int()%len(b)])
		if err != nil {
			t.Fatal(err)
		}
		sm, err := json.Marshal(string(b[:n]))
		if err != nil {
			t.Fatal(err)
		}
		var su string
		if err := json.Unmarshal([]byte(sm), &su); err != nil {
			t.Fatal(err)
		}
		token := Get(`{"str":`+string(sm)+`}`, "str")
		if token.Type != String || token.Str != su {
			println("["+token.Raw+"]", "["+token.Str+"]", "["+su+"]", "["+string(sm)+"]")
			t.Fatal("string mismatch")
		}
	}
}
func testEscapePath(t *testing.T, json, path, expect string) {
	if Get(json, path).String() != expect {
		t.Fatalf("expected '%v', got '%v'", expect, Get(json, path).String())
	}
}

func TestEscapePath(t *testing.T) {
	json := `{
		"test":{
			"*":"valZ",
			"*v":"val0",
			"keyv*":"val1",
			"key*v":"val2",
			"keyv?":"val3",
			"key?v":"val4",
			"keyv.":"val5",
			"key.v":"val6",
			"keyk*":{"key?":"val7"}
		}
	}`

	testEscapePath(t, json, "test.\\*", "valZ")
	testEscapePath(t, json, "test.\\*v", "val0")
	testEscapePath(t, json, "test.keyv\\*", "val1")
	testEscapePath(t, json, "test.key\\*v", "val2")
	testEscapePath(t, json, "test.keyv\\?", "val3")
	testEscapePath(t, json, "test.key\\?v", "val4")
	testEscapePath(t, json, "test.keyv\\.", "val5")
	testEscapePath(t, json, "test.key\\.v", "val6")
	testEscapePath(t, json, "test.keyk\\*.key\\?", "val7")
}

// this json block is poorly formed on purpose.
var basicJSON = `{"age":100, "name":{"here":"B\\\"R"},
	"noop":{"what is a wren?":"a bird"},
	"happy":true,"immortal":false,
	"items":[1,2,3,{"tags":[1,2,3],"points":[[1,2],[3,4]]},4,5,6,7],
	"arr":["1",2,"3",{"hello":"world"},"4",5],
	"vals":[1,2,3,{"sadf":sdf"asdf"}],"name":{"first":"tom","last":null},
	"loggy":{
		"programmers": [
    	    {
    	        "firstName": "Brett", 
    	        "lastName": "McLaughlin", 
    	        "email": "aaaa"
    	    }, 
    	    {
    	        "firstName": "Jason", 
    	        "lastName": "Hunter", 
    	        "email": "bbbb"
    	    }, 
    	    {
    	        "firstName": "Elliotte", 
    	        "lastName": "Harold", 
    	        "email": "cccc"
    	    },
			{
				"firstName": 1002.3
			}
    	]
	}
}`

func TestBasic(t *testing.T) {
	var mtok Result
	mtok = Get(basicJSON, "loggy")
	if mtok.Type != JSON {
		t.Fatalf("expected %v, got %v", JSON, mtok.Type)
	}
	if len(mtok.Map()) != 1 {
		t.Fatalf("expected %v, got %v", 1, len(mtok.Map()))
	}
	programmers := mtok.Map()["programmers"]
	if programmers.Array()[1].Map()["firstName"].Str != "Jason" {
		t.Fatalf("expected %v, got %v", "Jason", mtok.Map()["programmers"].Array()[1].Map()["firstName"].Str)
	}

	if Parse(basicJSON).Get("loggy.programmers").Get("1").Get("firstName").Str != "Jason" {
		t.Fatalf("expected %v, got %v", "Jason", Parse(basicJSON).Get("loggy.programmers").Get("1").Get("firstName").Str)
	}
	var token Result
	if token = Parse("-102"); token.Num != -102 {
		t.Fatal("expected %v, got %v", -102, token.Num)
	}
	if token = Parse("102"); token.Num != 102 {
		t.Fatal("expected %v, got %v", 102, token.Num)
	}
	if token = Parse("102.2"); token.Num != 102.2 {
		t.Fatal("expected %v, got %v", 102.2, token.Num)
	}
	if token = Parse(`"hello"`); token.Str != "hello" {
		t.Fatal("expected %v, got %v", "hello", token.Str)
	}
	if token = Parse(`"\"he\nllo\""`); token.Str != "\"he\nllo\"" {
		t.Fatal("expected %v, got %v", "\"he\nllo\"", token.Str)
	}
	mtok = Get(basicJSON, "loggy.programmers.#.firstName")
	if len(mtok.Array()) != 4 {
		t.Fatalf("expected 4, got %v", len(mtok.Array()))
	}
	for i, ex := range []string{"Brett", "Jason", "Elliotte", "1002.3"} {
		if mtok.Array()[i].String() != ex {
			t.Fatalf("expected '%v', got '%v'", ex, mtok.Array()[i].String())
		}
	}
	mtok = Get(basicJSON, "loggy.programmers.#.asd")
	if mtok.Type != JSON {
		t.Fatal("expected %v, got %v", JSON, mtok.Type)
	}
	if len(mtok.Array()) != 0 {
		t.Fatalf("expected 0, got %v", len(mtok.Array()))
	}

	if Get(basicJSON, "items.3.tags.#").Num != 3 {
		t.Fatalf("expected 3, got %v", Get(basicJSON, "items.3.tags.#").Num)
	}
	if Get(basicJSON, "items.3.points.1.#").Num != 2 {
		t.Fatalf("expected 2, got %v", Get(basicJSON, "items.3.points.1.#").Num)
	}
	if Get(basicJSON, "items.#").Num != 8 {
		t.Fatalf("expected 6, got %v", Get(basicJSON, "items.#").Num)
	}
	if Get(basicJSON, "vals.#").Num != 4 {
		t.Fatalf("expected 4, got %v", Get(basicJSON, "vals.#").Num)
	}
	if !Get(basicJSON, "name.last").Exists() {
		t.Fatal("expected true, got false")
	}
	token = Get(basicJSON, "name.here")
	if token.String() != "B\\\"R" {
		t.Fatal("expecting 'B\\\"R'", "got", token.String())
	}
	token = Get(basicJSON, "arr.#")
	if token.String() != "6" {
		t.Fatal("expecting '6'", "got", token.String())
	}
	token = Get(basicJSON, "arr.3.hello")
	if token.String() != "world" {
		t.Fatal("expecting 'world'", "got", token.String())
	}
	_ = token.Value().(string)
	token = Get(basicJSON, "name.first")
	if token.String() != "tom" {
		t.Fatal("expecting 'tom'", "got", token.String())
	}
	_ = token.Value().(string)
	token = Get(basicJSON, "name.last")
	if token.String() != "null" {
		t.Fatal("expecting 'null'", "got", token.String())
	}
	if token.Value() != nil {
		t.Fatal("should be nil")
	}
	token = Get(basicJSON, "age")
	if token.String() != "100" {
		t.Fatal("expecting '100'", "got", token.String())
	}
	_ = token.Value().(float64)
	token = Get(basicJSON, "happy")
	if token.String() != "true" {
		t.Fatal("expecting 'true'", "got", token.String())
	}
	_ = token.Value().(bool)
	token = Get(basicJSON, "immortal")
	if token.String() != "false" {
		t.Fatal("expecting 'false'", "got", token.String())
	}
	_ = token.Value().(bool)
	token = Get(basicJSON, "noop")
	if token.String() != `{"what is a wren?":"a bird"}` {
		t.Fatal("expecting '"+`{"what is a wren?":"a bird"}`+"'", "got", token.String())
	}
	_ = token.Value().(map[string]interface{})

	if Get(basicJSON, "").Value() != nil {
		t.Fatal("should be nil")
	}

	Get(basicJSON, "vals.hello")

	mm := Parse(basicJSON).Value().(map[string]interface{})
	fn := mm["loggy"].(map[string]interface{})["programmers"].([]interface{})[1].(map[string]interface{})["firstName"].(string)
	if fn != "Jason" {
		t.Fatalf("expecting %v, got %v", "Jason", fn)
	}
}
func TestMatch(t *testing.T) {
	if !wildcardMatch("hello world", "hello world", false) {
		t.Fatal("fail")
	}
	if wildcardMatch("hello world", "jello world", false) {
		t.Fatal("fail")
	}
	if !wildcardMatch("hello world", "hello*", false) {
		t.Fatal("fail")
	}
	if wildcardMatch("hello world", "jello*", false) {
		t.Fatal("fail")
	}
	if !wildcardMatch("hello world", "hello?world", false) {
		t.Fatal("fail")
	}
	if wildcardMatch("hello world", "jello?world", false) {
		t.Fatal("fail")
	}
	if !wildcardMatch("hello world", "he*o?world", false) {
		t.Fatal("fail")
	}
	if !wildcardMatch("hello world", "he*o?wor*", false) {
		t.Fatal("fail")
	}
	if !wildcardMatch("hello world", "he*o?*r*", false) {
		t.Fatal("fail")
	}
	if !wildcardMatch("的情况下解析一个", "*", true) {
		t.Fatal("fail")
	}
	if !wildcardMatch("的情况下解析一个", "*况下*", true) {
		t.Fatal("fail")
	}
	if !wildcardMatch("的情况下解析一个", "*况?*", true) {
		t.Fatal("fail")
	}
	if !wildcardMatch("的情况下解析一个", "的情况?解析一个", true) {
		t.Fatal("fail")
	}
}
func TestUnicode(t *testing.T) {
	var json = `{"key":0,"的情况下解":{"key":1,"的情况":2}}`
	if Get(json, "的情况下解.key").Num != 1 {
		t.Fatal("fail")
	}
	if Get(json, "的情况下解.的情况").Num != 2 {
		t.Fatal("fail")
	}
	if Get(json, "的情况下解.的?况").Num != 2 {
		t.Fatal("fail")
	}
	if Get(json, "的情况下解.的?*").Num != 2 {
		t.Fatal("fail")
	}
	if Get(json, "的情况下解.*?况").Num != 2 {
		t.Fatal("fail")
	}
	if Get(json, "的情?下解.*?况").Num != 2 {
		t.Fatal("fail")
	}
	if Get(json, "的情下解.*?况").Num != 0 {
		t.Fatal("fail")
	}
}

// TestWildcardMatch - Tests validate the logic of wild card matching.
// `WildcardMatch` supports '*' and '?' wildcards.
// Sample usage: In resource matching for folder policy validation.
func TestWildcardMatch(t *testing.T) {
	testCases := []struct {
		pattern string
		text    string
		matched bool
	}{
		// Test case - 1.
		// Test case with pattern containing key name with a prefix. Should accept the same text without a "*".
		{
			pattern: "my-folder/oo*",
			text:    "my-folder/oo",
			matched: true,
		},
		// Test case - 2.
		// Test case with "*" at the end of the pattern.
		{
			pattern: "my-folder/In*",
			text:    "my-folder/India/Karnataka/",
			matched: true,
		},
		// Test case - 3.
		// Test case with prefixes shuffled.
		// This should fail.
		{
			pattern: "my-folder/In*",
			text:    "my-folder/Karnataka/India/",
			matched: false,
		},
		// Test case - 4.
		// Test case with text expanded to the wildcards in the pattern.
		{
			pattern: "my-folder/In*/Ka*/Ban",
			text:    "my-folder/India/Karnataka/Ban",
			matched: true,
		},
		// Test case - 5.
		// Test case with the  keyname part is repeated as prefix several times.
		// This is valid.
		{
			pattern: "my-folder/In*/Ka*/Ban",
			text:    "my-folder/India/Karnataka/Ban/Ban/Ban/Ban/Ban",
			matched: true,
		},
		// Test case - 6.
		// Test case to validate that `*` can be expanded into multiple prefixes.
		{
			pattern: "my-folder/In*/Ka*/Ban",
			text:    "my-folder/India/Karnataka/Area1/Area2/Area3/Ban",
			matched: true,
		},
		// Test case - 7.
		// Test case to validate that `*` can be expanded into multiple prefixes.
		{
			pattern: "my-folder/In*/Ka*/Ban",
			text:    "my-folder/India/State1/State2/Karnataka/Area1/Area2/Area3/Ban",
			matched: true,
		},
		// Test case - 8.
		// Test case where the keyname part of the pattern is expanded in the text.
		{
			pattern: "my-folder/In*/Ka*/Ban",
			text:    "my-folder/India/Karnataka/Bangalore",
			matched: false,
		},
		// Test case - 9.
		// Test case with prefixes and wildcard expanded for all "*".
		{
			pattern: "my-folder/In*/Ka*/Ban*",
			text:    "my-folder/India/Karnataka/Bangalore",
			matched: true,
		},
		// Test case - 10.
		// Test case with keyname part being a wildcard in the pattern.
		{pattern: "my-folder/*",
			text:    "my-folder/India",
			matched: true,
		},
		// Test case - 11.
		{
			pattern: "my-folder/oo*",
			text:    "my-folder/odo",
			matched: false,
		},

		// Test case with pattern containing wildcard '?'.
		// Test case - 12.
		// "my-folder?/" matches "my-folder1/", "my-folder2/", "my-folder3" etc...
		// doesn't match "myfolder/".
		{
			pattern: "my-folder?/abc*",
			text:    "myfolder/abc",
			matched: false,
		},
		// Test case - 13.
		{
			pattern: "my-folder?/abc*",
			text:    "my-folder1/abc",
			matched: true,
		},
		// Test case - 14.
		{
			pattern: "my-?-folder/abc*",
			text:    "my--folder/abc",
			matched: false,
		},
		// Test case - 15.
		{
			pattern: "my-?-folder/abc*",
			text:    "my-1-folder/abc",
			matched: true,
		},
		// Test case - 16.
		{
			pattern: "my-?-folder/abc*",
			text:    "my-k-folder/abc",
			matched: true,
		},
		// Test case - 17.
		{
			pattern: "my??folder/abc*",
			text:    "myfolder/abc",
			matched: false,
		},
		// Test case - 18.
		{
			pattern: "my??folder/abc*",
			text:    "my4afolder/abc",
			matched: true,
		},
		// Test case - 19.
		{
			pattern: "my-folder?abc*",
			text:    "my-folder/abc",
			matched: true,
		},
		// Test case 20-21.
		// '?' matches '/' too. (works with s3).
		// This is because the namespace is considered flat.
		// "abc?efg" matches both "abcdefg" and "abc/efg".
		{
			pattern: "my-folder/abc?efg",
			text:    "my-folder/abcdefg",
			matched: true,
		},
		{
			pattern: "my-folder/abc?efg",
			text:    "my-folder/abc/efg",
			matched: true,
		},
		// Test case - 22.
		{
			pattern: "my-folder/abc????",
			text:    "my-folder/abc",
			matched: false,
		},
		// Test case - 23.
		{
			pattern: "my-folder/abc????",
			text:    "my-folder/abcde",
			matched: false,
		},
		// Test case - 24.
		{
			pattern: "my-folder/abc????",
			text:    "my-folder/abcdefg",
			matched: true,
		},
		// Test case 25-26.
		// test case with no '*'.
		{
			pattern: "my-folder/abc?",
			text:    "my-folder/abc",
			matched: false,
		},
		{
			pattern: "my-folder/abc?",
			text:    "my-folder/abcd",
			matched: true,
		},
		{
			pattern: "my-folder/abc?",
			text:    "my-folder/abcde",
			matched: false,
		},
		// Test case 27.
		{
			pattern: "my-folder/mnop*?",
			text:    "my-folder/mnop",
			matched: false,
		},
		// Test case 28.
		{
			pattern: "my-folder/mnop*?",
			text:    "my-folder/mnopqrst/mnopqr",
			matched: true,
		},
		// Test case 29.
		{
			pattern: "my-folder/mnop*?",
			text:    "my-folder/mnopqrst/mnopqrs",
			matched: true,
		},
		// Test case 30.
		{
			pattern: "my-folder/mnop*?",
			text:    "my-folder/mnop",
			matched: false,
		},
		// Test case 31.
		{
			pattern: "my-folder/mnop*?",
			text:    "my-folder/mnopq",
			matched: true,
		},
		// Test case 32.
		{
			pattern: "my-folder/mnop*?",
			text:    "my-folder/mnopqr",
			matched: true,
		},
		// Test case 33.
		{
			pattern: "my-folder/mnop*?and",
			text:    "my-folder/mnopqand",
			matched: true,
		},
		// Test case 34.
		{
			pattern: "my-folder/mnop*?and",
			text:    "my-folder/mnopand",
			matched: false,
		},
		// Test case 35.
		{
			pattern: "my-folder/mnop*?and",
			text:    "my-folder/mnopqand",
			matched: true,
		},
		// Test case 36.
		{
			pattern: "my-folder/mnop*?",
			text:    "my-folder/mn",
			matched: false,
		},
		// Test case 37.
		{
			pattern: "my-folder/mnop*?",
			text:    "my-folder/mnopqrst/mnopqrs",
			matched: true,
		},
		// Test case 38.
		{
			pattern: "my-folder/mnop*??",
			text:    "my-folder/mnopqrst",
			matched: true,
		},
		// Test case 39.
		{
			pattern: "my-folder/mnop*qrst",
			text:    "my-folder/mnopabcdegqrst",
			matched: true,
		},
		// Test case 40.
		{
			pattern: "my-folder/mnop*?and",
			text:    "my-folder/mnopqand",
			matched: true,
		},
		// Test case 41.
		{
			pattern: "my-folder/mnop*?and",
			text:    "my-folder/mnopand",
			matched: false,
		},
		// Test case 42.
		{
			pattern: "my-folder/mnop*?and?",
			text:    "my-folder/mnopqanda",
			matched: true,
		},
		// Test case 43.
		{
			pattern: "my-folder/mnop*?and",
			text:    "my-folder/mnopqanda",
			matched: false,
		},
		// Test case 44.

		{
			pattern: "my-?-folder/abc*",
			text:    "my-folder/mnopqanda",
			matched: false,
		},
	}
	// Iterating over the test cases, call the function under test and asert the output.
	for i, testCase := range testCases {
		actualResult := wildcardMatch(testCase.text, testCase.pattern, false)
		if testCase.matched != actualResult {
			t.Errorf("Test %d: Expected the result to be `%v`, but instead found it to be `%v`", i+1, testCase.matched, actualResult)
		}
	}
}
func TestUnescape(t *testing.T) {
	unescape(string([]byte{'\\', '\\', 0}))
	unescape(string([]byte{'\\', '/', '\\', 'b', '\\', 'f'}))
}
func assert(t testing.TB, cond bool) {
	if !cond {
		t.Fatal("assert failed")
	}
}
func TestLess(t *testing.T) {
	assert(t, !Result{Type: Null}.Less(Result{Type: Null}, true))
	assert(t, Result{Type: Null}.Less(Result{Type: False}, true))
	assert(t, Result{Type: Null}.Less(Result{Type: True}, true))
	assert(t, Result{Type: Null}.Less(Result{Type: JSON}, true))
	assert(t, Result{Type: Null}.Less(Result{Type: Number}, true))
	assert(t, Result{Type: Null}.Less(Result{Type: String}, true))
	assert(t, !Result{Type: False}.Less(Result{Type: Null}, true))
	assert(t, Result{Type: False}.Less(Result{Type: True}, true))
	assert(t, Result{Type: String, Str: "abc"}.Less(Result{Type: String, Str: "bcd"}, true))
	assert(t, Result{Type: String, Str: "ABC"}.Less(Result{Type: String, Str: "abc"}, true))
	assert(t, !Result{Type: String, Str: "ABC"}.Less(Result{Type: String, Str: "abc"}, false))
	assert(t, Result{Type: Number, Num: 123}.Less(Result{Type: Number, Num: 456}, true))
	assert(t, !Result{Type: Number, Num: 456}.Less(Result{Type: Number, Num: 123}, true))
	assert(t, !Result{Type: Number, Num: 456}.Less(Result{Type: Number, Num: 456}, true))
	assert(t, stringLessInsensitive("abcde", "BBCDE"))
	assert(t, stringLessInsensitive("abcde", "bBCDE"))
	assert(t, stringLessInsensitive("Abcde", "BBCDE"))
	assert(t, stringLessInsensitive("Abcde", "bBCDE"))
	assert(t, !stringLessInsensitive("bbcde", "aBCDE"))
	assert(t, !stringLessInsensitive("bbcde", "ABCDE"))
	assert(t, !stringLessInsensitive("Bbcde", "aBCDE"))
	assert(t, !stringLessInsensitive("Bbcde", "ABCDE"))
	assert(t, !stringLessInsensitive("abcde", "ABCDE"))
	assert(t, !stringLessInsensitive("Abcde", "ABCDE"))
	assert(t, !stringLessInsensitive("abcde", "ABCDE"))
	assert(t, !stringLessInsensitive("ABCDE", "ABCDE"))
	assert(t, !stringLessInsensitive("abcde", "abcde"))
	assert(t, !stringLessInsensitive("123abcde", "123Abcde"))
	assert(t, !stringLessInsensitive("123Abcde", "123Abcde"))
	assert(t, !stringLessInsensitive("123Abcde", "123abcde"))
	assert(t, !stringLessInsensitive("123abcde", "123abcde"))
	assert(t, !stringLessInsensitive("124abcde", "123abcde"))
	assert(t, !stringLessInsensitive("124Abcde", "123Abcde"))
	assert(t, !stringLessInsensitive("124Abcde", "123abcde"))
	assert(t, !stringLessInsensitive("124abcde", "123abcde"))
	assert(t, stringLessInsensitive("124abcde", "125abcde"))
	assert(t, stringLessInsensitive("124Abcde", "125Abcde"))
	assert(t, stringLessInsensitive("124Abcde", "125abcde"))
	assert(t, stringLessInsensitive("124abcde", "125abcde"))
}

func TestIssue6(t *testing.T) {
	data := `{
      "code": 0,
      "msg": "",
      "data": {
        "sz002024": {
          "qfqday": [
            [
              "2014-01-02",
              "8.93",
              "9.03",
              "9.17",
              "8.88",
              "621143.00"
            ],
            [
              "2014-01-03",
              "9.03",
              "9.30",
              "9.47",
              "8.98",
              "1624438.00"
            ]
          ]
        }
      }
    }`

	var num []string
	for _, v := range Get(data, "data.sz002024.qfqday.0").Array() {
		num = append(num, v.String())
	}
	if fmt.Sprintf("%v", num) != "[2014-01-02 8.93 9.03 9.17 8.88 621143.00]" {
		t.Fatalf("invalid result")
	}
}

var exampleJSON = `{
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
}`

type BenchStruct struct {
	Widget struct {
		Window struct {
			Name string `json:"name"`
		} `json:"window"`
		Image struct {
			HOffset int `json:"hOffset"`
		} `json:"image"`
		Text struct {
			OnMouseUp string `json:"onMouseUp"`
		} `json:"text"`
	} `json:"widget"`
}

var benchPaths = []string{
	"widget.window.name",
	"widget.image.hOffset",
	"widget.text.onMouseUp",
}

func BenchmarkGJSONGet(t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			if Get(exampleJSON, benchPaths[j]).Type == Null {
				t.Fatal("did not find the value")
			}
		}
	}
	t.N *= len(benchPaths) // because we are running against 3 paths
}

func BenchmarkGJSONUnmarshalMap(t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			parts := strings.Split(benchPaths[j], ".")
			m := Parse(exampleJSON).Value().(map[string]interface{})
			var v interface{}
			for len(parts) > 0 {
				part := parts[0]
				if len(parts) > 1 {
					m = m[part].(map[string]interface{})
					if m == nil {
						t.Fatal("did not find the value")
					}
				} else {
					v = m[part]
					if v == nil {
						t.Fatal("did not find the value")
					}
				}
				parts = parts[1:]
			}
		}
	}
	t.N *= len(benchPaths) // because we are running against 3 paths
}

func BenchmarkJSONUnmarshalMap(t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			parts := strings.Split(benchPaths[j], ".")
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(exampleJSON), &m); err != nil {
				t.Fatal(err)
			}
			var v interface{}
			for len(parts) > 0 {
				part := parts[0]
				if len(parts) > 1 {
					m = m[part].(map[string]interface{})
					if m == nil {
						t.Fatal("did not find the value")
					}
				} else {
					v = m[part]
					if v == nil {
						t.Fatal("did not find the value")
					}
				}
				parts = parts[1:]
			}
		}
	}
	t.N *= len(benchPaths) // because we are running against 3 paths
}

func BenchmarkJSONUnmarshalStruct(t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			var s BenchStruct
			if err := json.Unmarshal([]byte(exampleJSON), &s); err != nil {
				t.Fatal(err)
			}
			switch benchPaths[j] {
			case "widget.window.name":
				if s.Widget.Window.Name == "" {
					t.Fatal("did not find the value")
				}
			case "widget.image.hOffset":
				if s.Widget.Image.HOffset == 0 {
					t.Fatal("did not find the value")
				}
			case "widget.text.onMouseUp":
				if s.Widget.Text.OnMouseUp == "" {
					t.Fatal("did not find the value")
				}
			}
		}
	}
	t.N *= len(benchPaths) // because we are running against 3 paths
}

func BenchmarkJSONDecoder(t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			dec := json.NewDecoder(bytes.NewBuffer([]byte(exampleJSON)))
			var found bool
		outer:
			for {
				tok, err := dec.Token()
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Fatal(err)
				}
				switch v := tok.(type) {
				case string:
					if found {
						// break out once we find the value.
						break outer
					}
					switch benchPaths[j] {
					case "widget.window.name":
						if v == "name" {
							found = true
						}
					case "widget.image.hOffset":
						if v == "hOffset" {
							found = true
						}
					case "widget.text.onMouseUp":
						if v == "onMouseUp" {
							found = true
						}
					}
				}
			}
			if !found {
				t.Fatal("field not found")
			}
		}
	}
	t.N *= len(benchPaths) // because we are running against 3 paths
}

func BenchmarkFFJSONLexer(t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			l := fflib.NewFFLexer([]byte(exampleJSON))
			var found bool
		outer:
			for {
				t := l.Scan()
				if t == fflib.FFTok_eof {
					break
				}
				if t == fflib.FFTok_string {
					b, _ := l.CaptureField(t)
					v := string(b)
					if found {
						// break out once we find the value.
						break outer
					}
					switch benchPaths[j] {
					case "widget.window.name":
						if v == "\"name\"" {
							found = true
						}
					case "widget.image.hOffset":
						if v == "\"hOffset\"" {
							found = true
						}
					case "widget.text.onMouseUp":
						if v == "\"onMouseUp\"" {
							found = true
						}
					}
				}
			}
			if !found {
				t.Fatal("field not found")
			}
		}
	}
	t.N *= len(benchPaths) // because we are running against 3 paths
}

func BenchmarkEasyJSONLexer(t *testing.B) {
	t.ReportAllocs()
	skipCC := func(l *jlexer.Lexer, n int) {
		for i := 0; i < n; i++ {
			l.Skip()
			l.WantColon()
			l.Skip()
			l.WantComma()
		}
	}
	skipGroup := func(l *jlexer.Lexer, n int) {
		l.WantColon()
		l.Delim('{')
		skipCC(l, n)
		l.Delim('}')
		l.WantComma()
	}
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			l := &jlexer.Lexer{Data: []byte(exampleJSON)}
			l.Delim('{')
			if l.String() == "widget" {
				l.WantColon()
				l.Delim('{')
				switch benchPaths[j] {
				case "widget.window.name":
					skipCC(l, 1)
					if l.String() == "window" {
						l.WantColon()
						l.Delim('{')
						skipCC(l, 1)
						if l.String() == "name" {
							l.WantColon()
							if l.String() == "" {
								t.Fatal("did not find the value")
							}
						}
					}
				case "widget.image.hOffset":
					skipCC(l, 1)
					if l.String() == "window" {
						skipGroup(l, 4)
					}
					if l.String() == "image" {
						l.WantColon()
						l.Delim('{')
						skipCC(l, 1)
						if l.String() == "hOffset" {
							l.WantColon()
							if l.Int() == 0 {
								t.Fatal("did not find the value")
							}
						}
					}
				case "widget.text.onMouseUp":
					skipCC(l, 1)
					if l.String() == "window" {
						skipGroup(l, 4)
					}
					if l.String() == "image" {
						skipGroup(l, 4)
					}
					if l.String() == "text" {
						l.WantColon()
						l.Delim('{')
						skipCC(l, 5)
						if l.String() == "onMouseUp" {
							l.WantColon()
							if l.String() == "" {
								t.Fatal("did not find the value")
							}
						}
					}
				}
			}
		}
	}
	t.N *= len(benchPaths) // because we are running against 3 paths
}

func BenchmarkJSONParserGet(t *testing.B) {
	data := []byte(exampleJSON)
	keys := make([][]string, 0, len(benchPaths))
	for i := 0; i < len(benchPaths); i++ {
		keys = append(keys, strings.Split(benchPaths[i], "."))
	}
	t.ResetTimer()
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		for j, k := range keys {
			if j == 1 {
				// "widget.image.hOffset" is a number
				v, _ := jsonparser.GetInt(data, k...)
				if v == 0 {
					t.Fatal("did not find the value")
				}
			} else {
				// "widget.window.name",
				// "widget.text.onMouseUp",
				v, _ := jsonparser.GetString(data, k...)
				if v == "" {
					t.Fatal("did not find the value")
				}
			}
		}
	}
	t.N *= len(benchPaths) // because we are running against 3 paths
}
