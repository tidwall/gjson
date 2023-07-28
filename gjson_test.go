package gjson

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tidwall/pretty"
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
		GetBytes([]byte(lstr), "zzzz")
		Parse(lstr)
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
			println("["+token.Raw+"]", "["+token.Str+"]", "["+su+"]",
				"["+string(sm)+"]")
			t.Fatal("string mismatch")
		}
	}
}

func TestEmoji(t *testing.T) {
	const input = `{"utf8":"Example emoji, KO: \ud83d\udd13, \ud83c\udfc3 ` +
		`OK: \u2764\ufe0f "}`
	value := Get(input, "utf8")
	var s string
	json.Unmarshal([]byte(value.Raw), &s)
	if value.String() != s {
		t.Fatalf("expected '%v', got '%v'", s, value.String())
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
var basicJSON = `  {"age":100, "name":{"here":"B\\\"R"},
	"noop":{"what is a wren?":"a bird"},
	"happy":true,"immortal":false,
	"items":[1,2,3,{"tags":[1,2,3],"points":[[1,2],[3,4]]},4,5,6,7],
	"arr":["1",2,"3",{"hello":"world"},"4",5],
	"vals":[1,2,3,{"sadf":sdf"asdf"}],"name":{"first":"tom","last":null},
	"created":"2014-05-16T08:28:06.989Z",
	"loggy":{
		"programmers": [
    	    {
    	        "firstName": "Brett",
    	        "lastName": "McLaughlin",
    	        "email": "aaaa",
				"tag": "good"
    	    },
    	    {
    	        "firstName": "Jason",
    	        "lastName": "Hunter",
    	        "email": "bbbb",
				"tag": "bad"
    	    },
    	    {
    	        "firstName": "Elliotte",
    	        "lastName": "Harold",
    	        "email": "cccc",
				"tag":, "good"
    	    },
			{
				"firstName": 1002.3,
				"age": 101
			}
    	]
	},
	"lastly":{"end...ing":"soon","yay":"final"}
}`

func TestPath(t *testing.T) {
	json := basicJSON
	r := Get(json, "@this")
	path := r.Path(json)
	if path != "@this" {
		t.FailNow()
	}

	r = Parse(json)
	path = r.Path(json)
	if path != "@this" {
		t.FailNow()
	}

	obj := Parse(json)
	obj.ForEach(func(key, val Result) bool {
		kp := key.Path(json)
		assert(t, kp == "")
		vp := val.Path(json)
		if vp == "name" {
			// there are two "name" keys
			return true
		}
		val2 := obj.Get(vp)
		assert(t, val2.Raw == val.Raw)
		return true
	})
	arr := obj.Get("loggy.programmers")
	arr.ForEach(func(_, val Result) bool {
		vp := val.Path(json)
		val2 := Get(json, vp)
		assert(t, val2.Raw == val.Raw)
		return true
	})
	get := func(path string) {
		r1 := Get(json, path)
		path2 := r1.Path(json)
		r2 := Get(json, path2)
		assert(t, r1.Raw == r2.Raw)
	}
	get("age")
	get("name")
	get("name.here")
	get("noop")
	get("noop.what is a wren?")
	get("arr.0")
	get("arr.1")
	get("arr.2")
	get("arr.3")
	get("arr.3.hello")
	get("arr.4")
	get("arr.5")
	get("loggy.programmers.2.email")
	get("lastly.end\\.\\.\\.ing")
	get("lastly.yay")

}

func TestTimeResult(t *testing.T) {
	assert(t, Get(basicJSON, "created").String() ==
		Get(basicJSON, "created").Time().Format(time.RFC3339Nano))
}

func TestParseAny(t *testing.T) {
	assert(t, Parse("100").Float() == 100)
	assert(t, Parse("true").Bool())
	assert(t, Parse("false").Bool() == false)
	assert(t, Parse("yikes").Exists() == false)
}

func TestManyVariousPathCounts(t *testing.T) {
	json := `{"a":"a","b":"b","c":"c"}`
	counts := []int{3, 4, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127,
		128, 129, 255, 256, 257, 511, 512, 513}
	paths := []string{"a", "b", "c"}
	expects := []string{"a", "b", "c"}
	for _, count := range counts {
		var gpaths []string
		for i := 0; i < count; i++ {
			if i < len(paths) {
				gpaths = append(gpaths, paths[i])
			} else {
				gpaths = append(gpaths, fmt.Sprintf("not%d", i))
			}
		}
		results := GetMany(json, gpaths...)
		for i := 0; i < len(paths); i++ {
			if results[i].String() != expects[i] {
				t.Fatalf("expected '%v', got '%v'", expects[i],
					results[i].String())
			}
		}
	}
}
func TestManyRecursion(t *testing.T) {
	var json string
	var path string
	for i := 0; i < 100; i++ {
		json += `{"a":`
		path += ".a"
	}
	json += `"b"`
	for i := 0; i < 100; i++ {
		json += `}`
	}
	path = path[1:]
	assert(t, GetMany(json, path)[0].String() == "b")
}
func TestByteSafety(t *testing.T) {
	jsonb := []byte(`{"name":"Janet","age":38}`)
	mtok := GetBytes(jsonb, "name")
	if mtok.String() != "Janet" {
		t.Fatalf("expected %v, got %v", "Jason", mtok.String())
	}
	mtok2 := GetBytes(jsonb, "age")
	if mtok2.Raw != "38" {
		t.Fatalf("expected %v, got %v", "Jason", mtok2.Raw)
	}
	jsonb[9] = 'T'
	jsonb[12] = 'd'
	jsonb[13] = 'y'
	if mtok.String() != "Janet" {
		t.Fatalf("expected %v, got %v", "Jason", mtok.String())
	}
}

func get(json, path string) Result {
	return GetBytes([]byte(json), path)
}

func TestBasic(t *testing.T) {
	var mtok Result
	mtok = get(basicJSON, `loggy.programmers.#[tag="good"].firstName`)
	if mtok.String() != "Brett" {
		t.Fatalf("expected %v, got %v", "Brett", mtok.String())
	}
	mtok = get(basicJSON, `loggy.programmers.#[tag="good"]#.firstName`)
	if mtok.String() != `["Brett","Elliotte"]` {
		t.Fatalf("expected %v, got %v", `["Brett","Elliotte"]`, mtok.String())
	}
}

func TestIsArrayIsObject(t *testing.T) {
	mtok := get(basicJSON, "loggy")
	assert(t, mtok.IsObject())
	assert(t, !mtok.IsArray())

	mtok = get(basicJSON, "loggy.programmers")
	assert(t, !mtok.IsObject())
	assert(t, mtok.IsArray())

	mtok = get(basicJSON, `loggy.programmers.#[tag="good"]#.firstName`)
	assert(t, mtok.IsArray())

	mtok = get(basicJSON, `loggy.programmers.0.firstName`)
	assert(t, !mtok.IsObject())
	assert(t, !mtok.IsArray())
}

func TestPlus53BitInts(t *testing.T) {
	json := `{"IdentityData":{"GameInstanceId":634866135153775564}}`
	value := Get(json, "IdentityData.GameInstanceId")
	assert(t, value.Uint() == 634866135153775564)
	assert(t, value.Int() == 634866135153775564)
	assert(t, value.Float() == 634866135153775616)

	json = `{"IdentityData":{"GameInstanceId":634866135153775564.88172}}`
	value = Get(json, "IdentityData.GameInstanceId")
	assert(t, value.Uint() == 634866135153775616)
	assert(t, value.Int() == 634866135153775616)
	assert(t, value.Float() == 634866135153775616.88172)

	json = `{
		"min_uint64": 0,
		"max_uint64": 18446744073709551615,
		"overflow_uint64": 18446744073709551616,
		"min_int64": -9223372036854775808,
		"max_int64": 9223372036854775807,
		"overflow_int64": 9223372036854775808,
		"min_uint53":  0,
		"max_uint53":  4503599627370495,
		"overflow_uint53": 4503599627370496,
		"min_int53": -2251799813685248,
		"max_int53": 2251799813685247,
		"overflow_int53": 2251799813685248
	}`

	assert(t, Get(json, "min_uint53").Uint() == 0)
	assert(t, Get(json, "max_uint53").Uint() == 4503599627370495)
	assert(t, Get(json, "overflow_uint53").Int() == 4503599627370496)
	assert(t, Get(json, "min_int53").Int() == -2251799813685248)
	assert(t, Get(json, "max_int53").Int() == 2251799813685247)
	assert(t, Get(json, "overflow_int53").Int() == 2251799813685248)
	assert(t, Get(json, "min_uint64").Uint() == 0)
	assert(t, Get(json, "max_uint64").Uint() == 18446744073709551615)
	// this next value overflows the max uint64 by one which will just
	// flip the number to zero
	assert(t, Get(json, "overflow_uint64").Int() == 0)
	assert(t, Get(json, "min_int64").Int() == -9223372036854775808)
	assert(t, Get(json, "max_int64").Int() == 9223372036854775807)
	// this next value overflows the max int64 by one which will just
	// flip the number to the negative sign.
	assert(t, Get(json, "overflow_int64").Int() == -9223372036854775808)
}
func TestIssue38(t *testing.T) {
	// These should not fail, even though the unicode is invalid.
	Get(`["S3O PEDRO DO BUTI\udf93"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93asdf"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u1"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u13"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u134"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u1345"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u1345asd"]`, "0")
}
func TestTypes(t *testing.T) {
	assert(t, (Result{Type: String}).Type.String() == "String")
	assert(t, (Result{Type: Number}).Type.String() == "Number")
	assert(t, (Result{Type: Null}).Type.String() == "Null")
	assert(t, (Result{Type: False}).Type.String() == "False")
	assert(t, (Result{Type: True}).Type.String() == "True")
	assert(t, (Result{Type: JSON}).Type.String() == "JSON")
	assert(t, (Result{Type: 100}).Type.String() == "")
	// bool
	assert(t, (Result{Type: True}).Bool() == true)
	assert(t, (Result{Type: False}).Bool() == false)
	assert(t, (Result{Type: Number, Num: 1}).Bool() == true)
	assert(t, (Result{Type: Number, Num: 0}).Bool() == false)
	assert(t, (Result{Type: String, Str: "1"}).Bool() == true)
	assert(t, (Result{Type: String, Str: "T"}).Bool() == true)
	assert(t, (Result{Type: String, Str: "t"}).Bool() == true)
	assert(t, (Result{Type: String, Str: "true"}).Bool() == true)
	assert(t, (Result{Type: String, Str: "True"}).Bool() == true)
	assert(t, (Result{Type: String, Str: "TRUE"}).Bool() == true)
	assert(t, (Result{Type: String, Str: "tRuE"}).Bool() == true)
	assert(t, (Result{Type: String, Str: "0"}).Bool() == false)
	assert(t, (Result{Type: String, Str: "f"}).Bool() == false)
	assert(t, (Result{Type: String, Str: "F"}).Bool() == false)
	assert(t, (Result{Type: String, Str: "false"}).Bool() == false)
	assert(t, (Result{Type: String, Str: "False"}).Bool() == false)
	assert(t, (Result{Type: String, Str: "FALSE"}).Bool() == false)
	assert(t, (Result{Type: String, Str: "fAlSe"}).Bool() == false)
	assert(t, (Result{Type: String, Str: "random"}).Bool() == false)

	// int
	assert(t, (Result{Type: String, Str: "1"}).Int() == 1)
	assert(t, (Result{Type: True}).Int() == 1)
	assert(t, (Result{Type: False}).Int() == 0)
	assert(t, (Result{Type: Number, Num: 1}).Int() == 1)
	// uint
	assert(t, (Result{Type: String, Str: "1"}).Uint() == 1)
	assert(t, (Result{Type: True}).Uint() == 1)
	assert(t, (Result{Type: False}).Uint() == 0)
	assert(t, (Result{Type: Number, Num: 1}).Uint() == 1)
	// float
	assert(t, (Result{Type: String, Str: "1"}).Float() == 1)
	assert(t, (Result{Type: True}).Float() == 1)
	assert(t, (Result{Type: False}).Float() == 0)
	assert(t, (Result{Type: Number, Num: 1}).Float() == 1)
}
func TestForEach(t *testing.T) {
	Result{}.ForEach(nil)
	Result{Type: String, Str: "Hello"}.ForEach(func(_, value Result) bool {
		assert(t, value.String() == "Hello")
		return false
	})
	Result{Type: JSON, Raw: "*invalid*"}.ForEach(nil)

	json := ` {"name": {"first": "Janet","last": "Prichard"},
	"asd\nf":"\ud83d\udd13","age": 47}`
	var count int
	ParseBytes([]byte(json)).ForEach(func(key, value Result) bool {
		count++
		return true
	})
	assert(t, count == 3)
	ParseBytes([]byte(`{"bad`)).ForEach(nil)
	ParseBytes([]byte(`{"ok":"bad`)).ForEach(nil)
}
func TestMap(t *testing.T) {
	assert(t, len(ParseBytes([]byte(`"asdf"`)).Map()) == 0)
	assert(t, ParseBytes([]byte(`{"asdf":"ghjk"`)).Map()["asdf"].String() ==
		"ghjk")
	assert(t, len(Result{Type: JSON, Raw: "**invalid**"}.Map()) == 0)
	assert(t, Result{Type: JSON, Raw: "**invalid**"}.Value() == nil)
	assert(t, Result{Type: JSON, Raw: "{"}.Map() != nil)
}
func TestBasic1(t *testing.T) {
	mtok := get(basicJSON, `loggy.programmers`)
	var count int
	mtok.ForEach(func(key, value Result) bool {
		assert(t, key.Exists())
		assert(t, key.String() == fmt.Sprint(count))
		assert(t, key.Int() == int64(count))
		count++
		if count == 3 {
			return false
		}
		if count == 1 {
			i := 0
			value.ForEach(func(key, value Result) bool {
				switch i {
				case 0:
					if key.String() != "firstName" ||
						value.String() != "Brett" {
						t.Fatalf("expected %v/%v got %v/%v", "firstName",
							"Brett", key.String(), value.String())
					}
				case 1:
					if key.String() != "lastName" ||
						value.String() != "McLaughlin" {
						t.Fatalf("expected %v/%v got %v/%v", "lastName",
							"McLaughlin", key.String(), value.String())
					}
				case 2:
					if key.String() != "email" || value.String() != "aaaa" {
						t.Fatalf("expected %v/%v got %v/%v", "email", "aaaa",
							key.String(), value.String())
					}
				}
				i++
				return true
			})
		}
		return true
	})
	if count != 3 {
		t.Fatalf("expected %v, got %v", 3, count)
	}
}
func TestBasic2(t *testing.T) {
	mtok := get(basicJSON, `loggy.programmers.#[age=101].firstName`)
	if mtok.String() != "1002.3" {
		t.Fatalf("expected %v, got %v", "1002.3", mtok.String())
	}
	mtok = get(basicJSON,
		`loggy.programmers.#[firstName != "Brett"].firstName`)
	if mtok.String() != "Jason" {
		t.Fatalf("expected %v, got %v", "Jason", mtok.String())
	}
	mtok = get(basicJSON, `loggy.programmers.#[firstName % "Bre*"].email`)
	if mtok.String() != "aaaa" {
		t.Fatalf("expected %v, got %v", "aaaa", mtok.String())
	}
	mtok = get(basicJSON, `loggy.programmers.#[firstName !% "Bre*"].email`)
	if mtok.String() != "bbbb" {
		t.Fatalf("expected %v, got %v", "bbbb", mtok.String())
	}
	mtok = get(basicJSON, `loggy.programmers.#[firstName == "Brett"].email`)
	if mtok.String() != "aaaa" {
		t.Fatalf("expected %v, got %v", "aaaa", mtok.String())
	}
	mtok = get(basicJSON, "loggy")
	if mtok.Type != JSON {
		t.Fatalf("expected %v, got %v", JSON, mtok.Type)
	}
	if len(mtok.Map()) != 1 {
		t.Fatalf("expected %v, got %v", 1, len(mtok.Map()))
	}
	programmers := mtok.Map()["programmers"]
	if programmers.Array()[1].Map()["firstName"].Str != "Jason" {
		t.Fatalf("expected %v, got %v", "Jason",
			mtok.Map()["programmers"].Array()[1].Map()["firstName"].Str)
	}
}
func TestBasic3(t *testing.T) {
	var mtok Result
	if Parse(basicJSON).Get("loggy.programmers").Get("1").
		Get("firstName").Str != "Jason" {
		t.Fatalf("expected %v, got %v", "Jason", Parse(basicJSON).
			Get("loggy.programmers").Get("1").Get("firstName").Str)
	}
	var token Result
	if token = Parse("-102"); token.Num != -102 {
		t.Fatalf("expected %v, got %v", -102, token.Num)
	}
	if token = Parse("102"); token.Num != 102 {
		t.Fatalf("expected %v, got %v", 102, token.Num)
	}
	if token = Parse("102.2"); token.Num != 102.2 {
		t.Fatalf("expected %v, got %v", 102.2, token.Num)
	}
	if token = Parse(`"hello"`); token.Str != "hello" {
		t.Fatalf("expected %v, got %v", "hello", token.Str)
	}
	if token = Parse(`"\"he\nllo\""`); token.Str != "\"he\nllo\"" {
		t.Fatalf("expected %v, got %v", "\"he\nllo\"", token.Str)
	}
	mtok = get(basicJSON, "loggy.programmers.#.firstName")
	if len(mtok.Array()) != 4 {
		t.Fatalf("expected 4, got %v", len(mtok.Array()))
	}
	for i, ex := range []string{"Brett", "Jason", "Elliotte", "1002.3"} {
		if mtok.Array()[i].String() != ex {
			t.Fatalf("expected '%v', got '%v'", ex, mtok.Array()[i].String())
		}
	}
	mtok = get(basicJSON, "loggy.programmers.#.asd")
	if mtok.Type != JSON {
		t.Fatalf("expected %v, got %v", JSON, mtok.Type)
	}
	if len(mtok.Array()) != 0 {
		t.Fatalf("expected 0, got %v", len(mtok.Array()))
	}
}
func TestBasic4(t *testing.T) {
	if get(basicJSON, "items.3.tags.#").Num != 3 {
		t.Fatalf("expected 3, got %v", get(basicJSON, "items.3.tags.#").Num)
	}
	if get(basicJSON, "items.3.points.1.#").Num != 2 {
		t.Fatalf("expected 2, got %v",
			get(basicJSON, "items.3.points.1.#").Num)
	}
	if get(basicJSON, "items.#").Num != 8 {
		t.Fatalf("expected 6, got %v", get(basicJSON, "items.#").Num)
	}
	if get(basicJSON, "vals.#").Num != 4 {
		t.Fatalf("expected 4, got %v", get(basicJSON, "vals.#").Num)
	}
	if !get(basicJSON, "name.last").Exists() {
		t.Fatal("expected true, got false")
	}
	token := get(basicJSON, "name.here")
	if token.String() != "B\\\"R" {
		t.Fatal("expecting 'B\\\"R'", "got", token.String())
	}
	token = get(basicJSON, "arr.#")
	if token.String() != "6" {
		fmt.Printf("%#v\n", token)
		t.Fatal("expecting 6", "got", token.String())
	}
	token = get(basicJSON, "arr.3.hello")
	if token.String() != "world" {
		t.Fatal("expecting 'world'", "got", token.String())
	}
	_ = token.Value().(string)
	token = get(basicJSON, "name.first")
	if token.String() != "tom" {
		t.Fatal("expecting 'tom'", "got", token.String())
	}
	_ = token.Value().(string)
	token = get(basicJSON, "name.last")
	if token.String() != "" {
		t.Fatal("expecting ''", "got", token.String())
	}
	if token.Value() != nil {
		t.Fatal("should be nil")
	}
}
func TestBasic5(t *testing.T) {
	token := get(basicJSON, "age")
	if token.String() != "100" {
		t.Fatal("expecting '100'", "got", token.String())
	}
	_ = token.Value().(float64)
	token = get(basicJSON, "happy")
	if token.String() != "true" {
		t.Fatal("expecting 'true'", "got", token.String())
	}
	_ = token.Value().(bool)
	token = get(basicJSON, "immortal")
	if token.String() != "false" {
		t.Fatal("expecting 'false'", "got", token.String())
	}
	_ = token.Value().(bool)
	token = get(basicJSON, "noop")
	if token.String() != `{"what is a wren?":"a bird"}` {
		t.Fatal("expecting '"+`{"what is a wren?":"a bird"}`+"'", "got",
			token.String())
	}
	_ = token.Value().(map[string]interface{})

	if get(basicJSON, "").Value() != nil {
		t.Fatal("should be nil")
	}

	get(basicJSON, "vals.hello")

	type msi = map[string]interface{}
	type fi = []interface{}
	mm := Parse(basicJSON).Value().(msi)
	fn := mm["loggy"].(msi)["programmers"].(fi)[1].(msi)["firstName"].(string)
	if fn != "Jason" {
		t.Fatalf("expecting %v, got %v", "Jason", fn)
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

func TestUnescape(t *testing.T) {
	unescape(string([]byte{'\\', '\\', 0}))
	unescape(string([]byte{'\\', '/', '\\', 'b', '\\', 'f'}))
}
func assert(t testing.TB, cond bool) {
	if !cond {
		panic("assert failed")
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
	assert(t, Result{Type: String, Str: "abc"}.Less(Result{Type: String,
		Str: "bcd"}, true))
	assert(t, Result{Type: String, Str: "ABC"}.Less(Result{Type: String,
		Str: "abc"}, true))
	assert(t, !Result{Type: String, Str: "ABC"}.Less(Result{Type: String,
		Str: "abc"}, false))
	assert(t, Result{Type: Number, Num: 123}.Less(Result{Type: Number,
		Num: 456}, true))
	assert(t, !Result{Type: Number, Num: 456}.Less(Result{Type: Number,
		Num: 123}, true))
	assert(t, !Result{Type: Number, Num: 456}.Less(Result{Type: Number,
		Num: 456}, true))
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

func TestUnmarshalMap(t *testing.T) {
	var m1 = Parse(exampleJSON).Value().(map[string]interface{})
	var m2 map[string]interface{}
	if err := json.Unmarshal([]byte(exampleJSON), &m2); err != nil {
		t.Fatal(err)
	}
	b1, err := json.Marshal(m1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := json.Marshal(m2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("b1 != b2")
	}
}

func TestSingleArrayValue(t *testing.T) {
	var json = `{"key": "value","key2":[1,2,3,4,"A"]}`
	var result = Get(json, "key")
	var array = result.Array()
	if len(array) != 1 {
		t.Fatal("array is empty")
	}
	if array[0].String() != "value" {
		t.Fatalf("got %s, should be %s", array[0].String(), "value")
	}

	array = Get(json, "key2.#").Array()
	if len(array) != 1 {
		t.Fatalf("got '%v', expected '%v'", len(array), 1)
	}

	array = Get(json, "key3").Array()
	if len(array) != 0 {
		t.Fatalf("got '%v', expected '%v'", len(array), 0)
	}

}

var manyJSON = `  {
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"hello":"world"
	}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}
	"position":{"type":"Point","coordinates":[-115.24,33.09]},
	"loves":["world peace"],
	"name":{"last":"Anderson","first":"Nancy"},
	"age":31
	"":{"a":"emptya","b":"emptyb"},
	"name.last":"Yellow",
	"name.first":"Cat",
}`

var testWatchForFallback bool

func TestManyBasic(t *testing.T) {
	testWatchForFallback = true
	defer func() {
		testWatchForFallback = false
	}()
	testMany := func(shouldFallback bool, expect string, paths ...string) {
		results := GetManyBytes(
			[]byte(manyJSON),
			paths...,
		)
		if len(results) != len(paths) {
			t.Fatalf("expected %v, got %v", len(paths), len(results))
		}
		if fmt.Sprintf("%v", results) != expect {
			fmt.Printf("%v\n", paths)
			t.Fatalf("expected %v, got %v", expect, results)
		}
	}
	testMany(false, "[Point]", "position.type")
	testMany(false, `[emptya ["world peace"] 31]`, ".a", "loves", "age")
	testMany(false, `[["world peace"]]`, "loves")
	testMany(false, `[{"last":"Anderson","first":"Nancy"} Nancy]`, "name",
		"name.first")
	testMany(true, `[]`, strings.Repeat("a.", 40)+"hello")
	res := Get(manyJSON, strings.Repeat("a.", 48)+"a")
	testMany(true, `[`+res.String()+`]`, strings.Repeat("a.", 48)+"a")
	// these should fallback
	testMany(true, `[Cat Nancy]`, "name\\.first", "name.first")
	testMany(true, `[world]`, strings.Repeat("a.", 70)+"hello")
}
func testMany(t *testing.T, json string, paths, expected []string) {
	testManyAny(t, json, paths, expected, true)
	testManyAny(t, json, paths, expected, false)
}
func testManyAny(t *testing.T, json string, paths, expected []string,
	bytes bool) {
	var result []Result
	for i := 0; i < 2; i++ {
		var which string
		if i == 0 {
			which = "Get"
			result = nil
			for j := 0; j < len(expected); j++ {
				if bytes {
					result = append(result, GetBytes([]byte(json), paths[j]))
				} else {
					result = append(result, Get(json, paths[j]))
				}
			}
		} else if i == 1 {
			which = "GetMany"
			if bytes {
				result = GetManyBytes([]byte(json), paths...)
			} else {
				result = GetMany(json, paths...)
			}
		}
		for j := 0; j < len(expected); j++ {
			if result[j].String() != expected[j] {
				t.Fatalf("Using key '%s' for '%s'\nexpected '%v', got '%v'",
					paths[j], which, expected[j], result[j].String())
			}
		}
	}
}
func TestIssue20(t *testing.T) {
	json := `{ "name": "FirstName", "name1": "FirstName1", ` +
		`"address": "address1", "addressDetails": "address2", }`
	paths := []string{"name", "name1", "address", "addressDetails"}
	expected := []string{"FirstName", "FirstName1", "address1", "address2"}
	t.Run("SingleMany", func(t *testing.T) {
		testMany(t, json, paths,
			expected)
	})
}

func TestIssue21(t *testing.T) {
	json := `{ "Level1Field1":3,
	           "Level1Field4":4,
			   "Level1Field2":{ "Level2Field1":[ "value1", "value2" ],
			   "Level2Field2":{ "Level3Field1":[ { "key1":"value1" } ] } } }`
	paths := []string{"Level1Field1", "Level1Field2.Level2Field1",
		"Level1Field2.Level2Field2.Level3Field1", "Level1Field4"}
	expected := []string{"3", `[ "value1", "value2" ]`,
		`[ { "key1":"value1" } ]`, "4"}
	t.Run("SingleMany", func(t *testing.T) {
		testMany(t, json, paths,
			expected)
	})
}

func TestRandomMany(t *testing.T) {
	var lstr string
	defer func() {
		if v := recover(); v != nil {
			println("'" + hex.EncodeToString([]byte(lstr)) + "'")
			println("'" + lstr + "'")
			panic(v)
		}
	}()
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 512)
	for i := 0; i < 50000; i++ {
		n, err := rand.Read(b[:rand.Int()%len(b)])
		if err != nil {
			t.Fatal(err)
		}
		lstr = string(b[:n])
		paths := make([]string, rand.Int()%64)
		for i := range paths {
			var b []byte
			n := rand.Int() % 5
			for j := 0; j < n; j++ {
				if j > 0 {
					b = append(b, '.')
				}
				nn := rand.Int() % 10
				for k := 0; k < nn; k++ {
					b = append(b, 'a'+byte(rand.Int()%26))
				}
			}
			paths[i] = string(b)
		}
		GetMany(lstr, paths...)
	}
}

var complicatedJSON = `
{
	"tagged": "OK",
	"Tagged": "KO",
	"NotTagged": true,
	"unsettable": 101,
	"Nested": {
		"Yellow": "Green",
		"yellow": "yellow"
	},
	"nestedTagged": {
		"Green": "Green",
		"Map": {
			"this": "that",
			"and": "the other thing"
		},
		"Ints": {
			"Uint": 99,
			"Uint16": 16,
			"Uint32": 32,
			"Uint64": 65
		},
		"Uints": {
			"int": -99,
			"Int": -98,
			"Int16": -16,
			"Int32": -32,
			"int64": -64,
			"Int64": -65
		},
		"Uints": {
			"Float32": 32.32,
			"Float64": 64.64
		},
		"Byte": 254,
		"Bool": true
	},
	"LeftOut": "you shouldn't be here",
	"SelfPtr": {"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}},
	"SelfSlice": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"SelfSlicePtr": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"SelfPtrSlice": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"interface": "Tile38 Rocks!",
	"Interface": "Please Download",
	"Array": [0,2,3,4,5],
	"time": "2017-05-07T13:24:43-07:00",
	"Binary": "R0lGODlhPQBEAPeo",
	"NonBinary": [9,3,100,115]
}
`

func testvalid(t *testing.T, json string, expect bool) {
	t.Helper()
	_, ok := validpayload([]byte(json), 0)
	if ok != expect {
		t.Fatal("mismatch")
	}
}

func TestValidBasic(t *testing.T) {
	testvalid(t, "0", true)
	testvalid(t, "00", false)
	testvalid(t, "-00", false)
	testvalid(t, "-.", false)
	testvalid(t, "-.123", false)
	testvalid(t, "0.0", true)
	testvalid(t, "10.0", true)
	testvalid(t, "10e1", true)
	testvalid(t, "10EE", false)
	testvalid(t, "10E-", false)
	testvalid(t, "10E+", false)
	testvalid(t, "10E123", true)
	testvalid(t, "10E-123", true)
	testvalid(t, "10E-0123", true)
	testvalid(t, "", false)
	testvalid(t, " ", false)
	testvalid(t, "{}", true)
	testvalid(t, "{", false)
	testvalid(t, "-", false)
	testvalid(t, "-1", true)
	testvalid(t, "-1.", false)
	testvalid(t, "-1.0", true)
	testvalid(t, " -1.0", true)
	testvalid(t, " -1.0 ", true)
	testvalid(t, "-1.0 ", true)
	testvalid(t, "-1.0 i", false)
	testvalid(t, "-1.0 i", false)
	testvalid(t, "true", true)
	testvalid(t, " true", true)
	testvalid(t, " true ", true)
	testvalid(t, " True ", false)
	testvalid(t, " tru", false)
	testvalid(t, "false", true)
	testvalid(t, " false", true)
	testvalid(t, " false ", true)
	testvalid(t, " False ", false)
	testvalid(t, " fals", false)
	testvalid(t, "null", true)
	testvalid(t, " null", true)
	testvalid(t, " null ", true)
	testvalid(t, " Null ", false)
	testvalid(t, " nul", false)
	testvalid(t, " []", true)
	testvalid(t, " [true]", true)
	testvalid(t, " [ true, null ]", true)
	testvalid(t, " [ true,]", false)
	testvalid(t, `{"hello":"world"}`, true)
	testvalid(t, `{ "hello": "world" }`, true)
	testvalid(t, `{ "hello": "world", }`, false)
	testvalid(t, `{"a":"b",}`, false)
	testvalid(t, `{"a":"b","a"}`, false)
	testvalid(t, `{"a":"b","a":}`, false)
	testvalid(t, `{"a":"b","a":1}`, true)
	testvalid(t, `{"a":"b",2"1":2}`, false)
	testvalid(t, `{"a":"b","a": 1, "c":{"hi":"there"} }`, true)
	testvalid(t, `{"a":"b","a": 1, "c":{"hi":"there", "easy":["going",`+
		`{"mixed":"bag"}]} }`, true)
	testvalid(t, `""`, true)
	testvalid(t, `"`, false)
	testvalid(t, `"\n"`, true)
	testvalid(t, `"\"`, false)
	testvalid(t, `"\\"`, true)
	testvalid(t, `"a\\b"`, true)
	testvalid(t, `"a\\b\\\"a"`, true)
	testvalid(t, `"a\\b\\\uFFAAa"`, true)
	testvalid(t, `"a\\b\\\uFFAZa"`, false)
	testvalid(t, `"a\\b\\\uFFA"`, false)
	testvalid(t, string(complicatedJSON), true)
	testvalid(t, string(exampleJSON), true)
	testvalid(t, "[-]", false)
	testvalid(t, "[-.123]", false)
}

var jsonchars = []string{"{", "[", ",", ":", "}", "]", "1", "0", "true",
	"false", "null", `""`, `"\""`, `"a"`}

func makeRandomJSONChars(b []byte) {
	var bb []byte
	for len(bb) < len(b) {
		bb = append(bb, jsonchars[rand.Int()%len(jsonchars)]...)
	}
	copy(b, bb[:len(b)])
}

func TestValidRandom(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 100000)
	start := time.Now()
	for time.Since(start) < time.Second*3 {
		n := rand.Int() % len(b)
		rand.Read(b[:n])
		validpayload(b[:n], 0)
	}

	start = time.Now()
	for time.Since(start) < time.Second*3 {
		n := rand.Int() % len(b)
		makeRandomJSONChars(b[:n])
		validpayload(b[:n], 0)
	}
}

func TestGetMany47(t *testing.T) {
	json := `{"bar": {"id": 99, "mybar": "my mybar" }, "foo": ` +
		`{"myfoo": [605]}}`
	paths := []string{"foo.myfoo", "bar.id", "bar.mybar", "bar.mybarx"}
	expected := []string{"[605]", "99", "my mybar", ""}
	results := GetMany(json, paths...)
	if len(expected) != len(results) {
		t.Fatalf("expected %v, got %v", len(expected), len(results))
	}
	for i, path := range paths {
		if results[i].String() != expected[i] {
			t.Fatalf("expected '%v', got '%v' for path '%v'", expected[i],
				results[i].String(), path)
		}
	}
}

func TestGetMany48(t *testing.T) {
	json := `{"bar": {"id": 99, "xyz": "my xyz"}, "foo": {"myfoo": [605]}}`
	paths := []string{"foo.myfoo", "bar.id", "bar.xyz", "bar.abc"}
	expected := []string{"[605]", "99", "my xyz", ""}
	results := GetMany(json, paths...)
	if len(expected) != len(results) {
		t.Fatalf("expected %v, got %v", len(expected), len(results))
	}
	for i, path := range paths {
		if results[i].String() != expected[i] {
			t.Fatalf("expected '%v', got '%v' for path '%v'", expected[i],
				results[i].String(), path)
		}
	}
}

func TestResultRawForLiteral(t *testing.T) {
	for _, lit := range []string{"null", "true", "false"} {
		result := Parse(lit)
		if result.Raw != lit {
			t.Fatalf("expected '%v', got '%v'", lit, result.Raw)
		}
	}
}

func TestNullArray(t *testing.T) {
	n := len(Get(`{"data":null}`, "data").Array())
	if n != 0 {
		t.Fatalf("expected '%v', got '%v'", 0, n)
	}
	n = len(Get(`{}`, "data").Array())
	if n != 0 {
		t.Fatalf("expected '%v', got '%v'", 0, n)
	}
	n = len(Get(`{"data":[]}`, "data").Array())
	if n != 0 {
		t.Fatalf("expected '%v', got '%v'", 0, n)
	}
	n = len(Get(`{"data":[null]}`, "data").Array())
	if n != 1 {
		t.Fatalf("expected '%v', got '%v'", 1, n)
	}
}

func TestIssue54(t *testing.T) {
	var r []Result
	json := `{"MarketName":null,"Nounce":6115}`
	r = GetMany(json, "Nounce", "Buys", "Sells", "Fills")
	if strings.Replace(fmt.Sprintf("%v", r), " ", "", -1) != "[6115]" {
		t.Fatalf("expected '%v', got '%v'", "[6115]",
			strings.Replace(fmt.Sprintf("%v", r), " ", "", -1))
	}
	r = GetMany(json, "Nounce", "Buys", "Sells")
	if strings.Replace(fmt.Sprintf("%v", r), " ", "", -1) != "[6115]" {
		t.Fatalf("expected '%v', got '%v'", "[6115]",
			strings.Replace(fmt.Sprintf("%v", r), " ", "", -1))
	}
	r = GetMany(json, "Nounce")
	if strings.Replace(fmt.Sprintf("%v", r), " ", "", -1) != "[6115]" {
		t.Fatalf("expected '%v', got '%v'", "[6115]",
			strings.Replace(fmt.Sprintf("%v", r), " ", "", -1))
	}
}

func TestIssue55(t *testing.T) {
	json := `{"one": {"two": 2, "three": 3}, "four": 4, "five": 5}`
	results := GetMany(json, "four", "five", "one.two", "one.six")
	expected := []string{"4", "5", "2", ""}
	for i, r := range results {
		if r.String() != expected[i] {
			t.Fatalf("expected %v, got %v", expected[i], r.String())
		}
	}
}
func TestIssue58(t *testing.T) {
	json := `{"data":[{"uid": 1},{"uid": 2}]}`
	res := Get(json, `data.#[uid!=1]`).Raw
	if res != `{"uid": 2}` {
		t.Fatalf("expected '%v', got '%v'", `{"uid": 1}`, res)
	}
}

func TestObjectGrouping(t *testing.T) {
	json := `
[
	true,
	{"name":"tom"},
	false,
	{"name":"janet"},
	null
]
`
	res := Get(json, "#.name")
	if res.String() != `["tom","janet"]` {
		t.Fatalf("expected '%v', got '%v'", `["tom","janet"]`, res.String())
	}
}

func TestJSONLines(t *testing.T) {
	json := `
true
false
{"name":"tom"}
[1,2,3,4,5]
{"name":"janet"}
null
12930.1203
	`
	paths := []string{"..#", "..0", "..2.name", "..#.name", "..6", "..7"}
	ress := []string{"7", "true", "tom", `["tom","janet"]`, "12930.1203", ""}
	for i, path := range paths {
		res := Get(json, path)
		if res.String() != ress[i] {
			t.Fatalf("expected '%v', got '%v'", ress[i], res.String())
		}
	}

	json = `
{"name": "Gilbert", "wins": [["straight", "7♣"], ["one pair", "10♥"]]}
{"name": "Alexa", "wins": [["two pair", "4♠"], ["two pair", "9♠"]]}
{"name": "May", "wins": []}
{"name": "Deloise", "wins": [["three of a kind", "5♣"]]}
`

	var i int
	lines := strings.Split(strings.TrimSpace(json), "\n")
	ForEachLine(json, func(line Result) bool {
		if line.Raw != lines[i] {
			t.Fatalf("expected '%v', got '%v'", lines[i], line.Raw)
		}
		i++
		return true
	})
	if i != 4 {
		t.Fatalf("expected '%v', got '%v'", 4, i)
	}

}

func TestNumUint64String(t *testing.T) {
	var i int64 = 9007199254740993 //2^53 + 1
	j := fmt.Sprintf(`{"data":  [  %d, "hello" ] }`, i)
	res := Get(j, "data.0")
	if res.String() != "9007199254740993" {
		t.Fatalf("expected '%v', got '%v'", "9007199254740993", res.String())
	}
}

func TestNumInt64String(t *testing.T) {
	var i int64 = -9007199254740993
	j := fmt.Sprintf(`{"data":[ "hello", %d ]}`, i)
	res := Get(j, "data.1")
	if res.String() != "-9007199254740993" {
		t.Fatalf("expected '%v', got '%v'", "-9007199254740993", res.String())
	}
}

func TestNumBigString(t *testing.T) {
	i := "900719925474099301239109123101" // very big
	j := fmt.Sprintf(`{"data":[ "hello", "%s" ]}`, i)
	res := Get(j, "data.1")
	if res.String() != "900719925474099301239109123101" {
		t.Fatalf("expected '%v', got '%v'", "900719925474099301239109123101",
			res.String())
	}
}

func TestNumFloatString(t *testing.T) {
	var i int64 = -9007199254740993
	j := fmt.Sprintf(`{"data":[ "hello", %d ]}`, i) //No quotes around value!!
	res := Get(j, "data.1")
	if res.String() != "-9007199254740993" {
		t.Fatalf("expected '%v', got '%v'", "-9007199254740993", res.String())
	}
}

func TestDuplicateKeys(t *testing.T) {
	// this is valid json according to the JSON spec
	var json = `{"name": "Alex","name": "Peter"}`
	if Parse(json).Get("name").String() !=
		Parse(json).Map()["name"].String() {
		t.Fatalf("expected '%v', got '%v'",
			Parse(json).Get("name").String(),
			Parse(json).Map()["name"].String(),
		)
	}
	if !Valid(json) {
		t.Fatal("should be valid")
	}
}

func TestArrayValues(t *testing.T) {
	var json = `{"array": ["PERSON1","PERSON2",0],}`
	values := Get(json, "array").Array()
	var output string
	for i, val := range values {
		if i > 0 {
			output += "\n"
		}
		output += fmt.Sprintf("%#v", val)
	}
	expect := strings.Join([]string{
		`gjson.Result{Type:3, Raw:"\"PERSON1\"", Str:"PERSON1", Num:0, ` +
			`Index:11, Indexes:[]int(nil)}`,
		`gjson.Result{Type:3, Raw:"\"PERSON2\"", Str:"PERSON2", Num:0, ` +
			`Index:21, Indexes:[]int(nil)}`,
		`gjson.Result{Type:2, Raw:"0", Str:"", Num:0, Index:31, Indexes:[]int(nil)}`,
	}, "\n")
	if output != expect {
		t.Fatalf("expected '%v', got '%v'", expect, output)
	}

}

func BenchmarkValid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Valid(complicatedJSON)
	}
}

func BenchmarkValidBytes(b *testing.B) {
	complicatedJSON := []byte(complicatedJSON)
	for i := 0; i < b.N; i++ {
		ValidBytes(complicatedJSON)
	}
}

func BenchmarkGoStdlibValidBytes(b *testing.B) {
	complicatedJSON := []byte(complicatedJSON)
	for i := 0; i < b.N; i++ {
		json.Valid(complicatedJSON)
	}
}

func TestModifier(t *testing.T) {
	json := `{"other":{"hello":"world"},"arr":[1,2,3,4,5,6]}`
	opts := *pretty.DefaultOptions
	opts.SortKeys = true
	exp := string(pretty.PrettyOptions([]byte(json), &opts))
	res := Get(json, `@pretty:{"sortKeys":true}`).String()
	if res != exp {
		t.Fatalf("expected '%v', got '%v'", exp, res)
	}
	res = Get(res, "@pretty|@reverse|@ugly").String()
	if res != json {
		t.Fatalf("expected '%v', got '%v'", json, res)
	}
	if res := Get(res, "@this").String(); res != json {
		t.Fatalf("expected '%v', got '%v'", json, res)
	}
	if res := Get(res, "other.@this").String(); res != `{"hello":"world"}` {
		t.Fatalf("expected '%v', got '%v'", json, res)
	}
	res = Get(res, "@pretty|@reverse|arr|@reverse|2").String()
	if res != "4" {
		t.Fatalf("expected '%v', got '%v'", "4", res)
	}
	AddModifier("case", func(json, arg string) string {
		if arg == "upper" {
			return strings.ToUpper(json)
		}
		if arg == "lower" {
			return strings.ToLower(json)
		}
		return json
	})
	res = Get(json, "other|@case:upper").String()
	if res != `{"HELLO":"WORLD"}` {
		t.Fatalf("expected '%v', got '%v'", `{"HELLO":"WORLD"}`, res)
	}
}

func TestChaining(t *testing.T) {
	json := `{
		"info": {
			"friends": [
				{"first": "Dale", "last": "Murphy", "age": 44},
				{"first": "Roger", "last": "Craig", "age": 68},
				{"first": "Jane", "last": "Murphy", "age": 47}
			]
		}
	  }`
	res := Get(json, "info.friends|0|first").String()
	if res != "Dale" {
		t.Fatalf("expected '%v', got '%v'", "Dale", res)
	}
	res = Get(json, "info.friends|@reverse|0|age").String()
	if res != "47" {
		t.Fatalf("expected '%v', got '%v'", "47", res)
	}
	res = Get(json, "@ugly|i\\nfo|friends.0.first").String()
	if res != "Dale" {
		t.Fatalf("expected '%v', got '%v'", "Dale", res)
	}
}

func TestSplitPipe(t *testing.T) {
	split := func(t *testing.T, path, el, er string, eo bool) {
		t.Helper()
		left, right, ok := splitPossiblePipe(path)
		// fmt.Printf("%-40s [%v] [%v] [%v]\n", path, left, right, ok)
		if left != el || right != er || ok != eo {
			t.Fatalf("expected '%v/%v/%v', got '%v/%v/%v",
				el, er, eo, left, right, ok)
		}
	}

	split(t, "hello", "", "", false)
	split(t, "hello.world", "", "", false)
	split(t, "hello|world", "hello", "world", true)
	split(t, "hello\\|world", "", "", false)
	split(t, "hello.#", "", "", false)
	split(t, `hello.#[a|1="asdf\"|1324"]#\|that`, "", "", false)
	split(t, `hello.#[a|1="asdf\"|1324"]#|that.more|yikes`,
		`hello.#[a|1="asdf\"|1324"]#`, "that.more|yikes", true)
	split(t, `a.#[]#\|b`, "", "", false)

}

func TestArrayEx(t *testing.T) {
	json := `
	[
		{
			"c":[
				{"a":10.11}
			]
		}, {
			"c":[
				{"a":11.11}
			]
		}
	]`
	res := Get(json, "@ugly|#.c.#[a=10.11]").String()
	if res != `[{"a":10.11}]` {
		t.Fatalf("expected '%v', got '%v'", `[{"a":10.11}]`, res)
	}
	res = Get(json, "@ugly|#.c.#").String()
	if res != `[1,1]` {
		t.Fatalf("expected '%v', got '%v'", `[1,1]`, res)
	}
	res = Get(json, "@reverse|0|c|0|a").String()
	if res != "11.11" {
		t.Fatalf("expected '%v', got '%v'", "11.11", res)
	}
	res = Get(json, "#.c|#").String()
	if res != "2" {
		t.Fatalf("expected '%v', got '%v'", "2", res)
	}
}

func TestPipeDotMixing(t *testing.T) {
	json := `{
		"info": {
			"friends": [
				{"first": "Dale", "last": "Murphy", "age": 44},
				{"first": "Roger", "last": "Craig", "age": 68},
				{"first": "Jane", "last": "Murphy", "age": 47}
			]
		}
	  }`
	var res string
	res = Get(json, `info.friends.#[first="Dale"].last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(json, `info|friends.#[first="Dale"].last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(json, `info|friends.#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(json, `info|friends|#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(json, `@ugly|info|friends|#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(json, `@ugly|info.@ugly|friends|#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(json, `@ugly.info|@ugly.friends|#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
}

func TestDeepSelectors(t *testing.T) {
	json := `{
		"info": {
			"friends": [
				{
					"first": "Dale", "last": "Murphy",
					"extra": [10,20,30],
					"details": {
						"city": "Tempe",
						"state": "Arizona"
					}
				},
				{
					"first": "Roger", "last": "Craig",
					"extra": [40,50,60],
					"details": {
						"city": "Phoenix",
						"state": "Arizona"
					}
				}
			]
		}
	  }`
	var res string
	res = Get(json, `info.friends.#[first="Dale"].extra.0`).String()
	if res != "10" {
		t.Fatalf("expected '%v', got '%v'", "10", res)
	}
	res = Get(json, `info.friends.#[first="Dale"].extra|0`).String()
	if res != "10" {
		t.Fatalf("expected '%v', got '%v'", "10", res)
	}
	res = Get(json, `info.friends.#[first="Dale"]|extra|0`).String()
	if res != "10" {
		t.Fatalf("expected '%v', got '%v'", "10", res)
	}
	res = Get(json, `info.friends.#[details.city="Tempe"].last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(json, `info.friends.#[details.city="Phoenix"].last`).String()
	if res != "Craig" {
		t.Fatalf("expected '%v', got '%v'", "Craig", res)
	}
	res = Get(json, `info.friends.#[details.state="Arizona"].last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
}

func TestMultiArrayEx(t *testing.T) {
	json := `{
		"info": {
			"friends": [
				{
					"first": "Dale", "last": "Murphy", "kind": "Person",
					"cust1": true,
					"extra": [10,20,30],
					"details": {
						"city": "Tempe",
						"state": "Arizona"
					}
				},
				{
					"first": "Roger", "last": "Craig", "kind": "Person",
					"cust2": false,
					"extra": [40,50,60],
					"details": {
						"city": "Phoenix",
						"state": "Arizona"
					}
				}
			]
		}
	  }`

	var res string

	res = Get(json, `info.friends.#[kind="Person"]#.kind|0`).String()
	if res != "Person" {
		t.Fatalf("expected '%v', got '%v'", "Person", res)
	}
	res = Get(json, `info.friends.#.kind|0`).String()
	if res != "Person" {
		t.Fatalf("expected '%v', got '%v'", "Person", res)
	}

	res = Get(json, `info.friends.#[kind="Person"]#.kind`).String()
	if res != `["Person","Person"]` {
		t.Fatalf("expected '%v', got '%v'", `["Person","Person"]`, res)
	}
	res = Get(json, `info.friends.#.kind`).String()
	if res != `["Person","Person"]` {
		t.Fatalf("expected '%v', got '%v'", `["Person","Person"]`, res)
	}

	res = Get(json, `info.friends.#[kind="Person"]#|kind`).String()
	if res != `` {
		t.Fatalf("expected '%v', got '%v'", ``, res)
	}
	res = Get(json, `info.friends.#|kind`).String()
	if res != `` {
		t.Fatalf("expected '%v', got '%v'", ``, res)
	}

	res = Get(json, `i*.f*.#[kind="Other"]#`).String()
	if res != `[]` {
		t.Fatalf("expected '%v', got '%v'", `[]`, res)
	}
}

func TestQueries(t *testing.T) {
	json := `{
		"info": {
			"friends": [
				{
					"first": "Dale", "last": "Murphy", "kind": "Person",
					"cust1": true,
					"extra": [10,20,30],
					"details": {
						"city": "Tempe",
						"state": "Arizona"
					}
				},
				{
					"first": "Roger", "last": "Craig", "kind": "Person",
					"cust2": false,
					"extra": [40,50,60],
					"details": {
						"city": "Phoenix",
						"state": "Arizona"
					}
				}
			]
		}
	  }`

	// numbers
	assert(t, Get(json, "i*.f*.#[extra.0<11].first").Exists())
	assert(t, Get(json, "i*.f*.#[extra.0<=11].first").Exists())
	assert(t, !Get(json, "i*.f*.#[extra.0<10].first").Exists())
	assert(t, Get(json, "i*.f*.#[extra.0<=10].first").Exists())
	assert(t, Get(json, "i*.f*.#[extra.0=10].first").Exists())
	assert(t, !Get(json, "i*.f*.#[extra.0=11].first").Exists())
	assert(t, Get(json, "i*.f*.#[extra.0!=10].first").String() == "Roger")
	assert(t, Get(json, "i*.f*.#[extra.0>10].first").String() == "Roger")
	assert(t, Get(json, "i*.f*.#[extra.0>=10].first").String() == "Dale")

	// strings
	assert(t, Get(json, `i*.f*.#[extra.0<"11"].first`).Exists())
	assert(t, Get(json, `i*.f*.#[first>"Dale"].last`).String() == "Craig")
	assert(t, Get(json, `i*.f*.#[first>="Dale"].last`).String() == "Murphy")
	assert(t, Get(json, `i*.f*.#[first="Dale"].last`).String() == "Murphy")
	assert(t, Get(json, `i*.f*.#[first!="Dale"].last`).String() == "Craig")
	assert(t, !Get(json, `i*.f*.#[first<"Dale"].last`).Exists())
	assert(t, Get(json, `i*.f*.#[first<="Dale"].last`).Exists())
	assert(t, Get(json, `i*.f*.#[first%"Da*"].last`).Exists())
	assert(t, Get(json, `i*.f*.#[first%"Dale"].last`).Exists())
	assert(t, Get(json, `i*.f*.#[first%"*a*"]#|#`).String() == "1")
	assert(t, Get(json, `i*.f*.#[first%"*e*"]#|#`).String() == "2")
	assert(t, Get(json, `i*.f*.#[first!%"*e*"]#|#`).String() == "0")

	// trues
	assert(t, Get(json, `i*.f*.#[cust1=true].first`).String() == "Dale")
	assert(t, Get(json, `i*.f*.#[cust2=false].first`).String() == "Roger")
	assert(t, Get(json, `i*.f*.#[cust1!=false].first`).String() == "Dale")
	assert(t, Get(json, `i*.f*.#[cust2!=true].first`).String() == "Roger")
	assert(t, !Get(json, `i*.f*.#[cust1>true].first`).Exists())
	assert(t, Get(json, `i*.f*.#[cust1>=true].first`).Exists())
	assert(t, !Get(json, `i*.f*.#[cust2<false].first`).Exists())
	assert(t, Get(json, `i*.f*.#[cust2<=false].first`).Exists())

}

func TestQueryArrayValues(t *testing.T) {
	json := `{
		"artists": [
			["Bob Dylan"],
			"John Lennon",
			"Mick Jagger",
			"Elton John",
			"Michael Jackson",
			"John Smith",
			true,
			123,
			456,
			false,
			null
		]
	}`
	assert(t, Get(json, `a*.#[0="Bob Dylan"]#|#`).String() == "1")
	assert(t, Get(json, `a*.#[0="Bob Dylan 2"]#|#`).String() == "0")
	assert(t, Get(json, `a*.#[%"John*"]#|#`).String() == "2")
	assert(t, Get(json, `a*.#[_%"John*"]#|#`).String() == "0")
	assert(t, Get(json, `a*.#[="123"]#|#`).String() == "1")
}

func TestParenQueries(t *testing.T) {
	json := `{
		"friends": [{"a":10},{"a":20},{"a":30},{"a":40}]
	}`
	assert(t, Get(json, "friends.#(a>9)#|#").Int() == 4)
	assert(t, Get(json, "friends.#(a>10)#|#").Int() == 3)
	assert(t, Get(json, "friends.#(a>40)#|#").Int() == 0)
}

func TestSubSelectors(t *testing.T) {
	json := `{
		"info": {
			"friends": [
				{
					"first": "Dale", "last": "Murphy", "kind": "Person",
					"cust1": true,
					"extra": [10,20,30],
					"details": {
						"city": "Tempe",
						"state": "Arizona"
					}
				},
				{
					"first": "Roger", "last": "Craig", "kind": "Person",
					"cust2": false,
					"extra": [40,50,60],
					"details": {
						"city": "Phoenix",
						"state": "Arizona"
					}
				}
			]
		}
	  }`
	assert(t, Get(json, "[]").String() == "[]")
	assert(t, Get(json, "{}").String() == "{}")
	res := Get(json, `{`+
		`abc:info.friends.0.first,`+
		`info.friends.1.last,`+
		`"a`+"\r"+`a":info.friends.0.kind,`+
		`"abc":info.friends.1.kind,`+
		`{123:info.friends.1.cust2},`+
		`[info.friends.#[details.city="Phoenix"]#|#]`+
		`}.@pretty.@ugly`).String()
	// println(res)
	// {"abc":"Dale","last":"Craig","\"a\ra\"":"Person","_":{"123":false},"_":[1]}
	assert(t, Get(res, "abc").String() == "Dale")
	assert(t, Get(res, "last").String() == "Craig")
	assert(t, Get(res, "\"a\ra\"").String() == "Person")
	assert(t, Get(res, "@reverse.abc").String() == "Person")
	assert(t, Get(res, "_.123").String() == "false")
	assert(t, Get(res, "@reverse._.0").String() == "1")
	assert(t, Get(json, "info.friends.[0.first,1.extra.0]").String() ==
		`["Dale",40]`)
	assert(t, Get(json, "info.friends.#.[first,extra.0]").String() ==
		`[["Dale",10],["Roger",40]]`)
}

func TestArrayCountRawOutput(t *testing.T) {
	assert(t, Get(`[1,2,3,4]`, "#").Raw == "4")
}

func TestParseQuery(t *testing.T) {
	var path, op, value, remain string
	var ok bool

	path, op, value, remain, _, _, ok =
		parseQuery(`#(service_roles.#(=="one").()==asdf).cap`)
	assert(t, ok &&
		path == `service_roles.#(=="one").()` &&
		op == "=" &&
		value == `asdf` &&
		remain == `.cap`)

	path, op, value, remain, _, _, ok = parseQuery(`#(first_name%"Murphy").last`)
	assert(t, ok &&
		path == `first_name` &&
		op == `%` &&
		value == `"Murphy"` &&
		remain == `.last`)

	path, op, value, remain, _, _, ok = parseQuery(`#( first_name !% "Murphy" ).last`)
	assert(t, ok &&
		path == `first_name` &&
		op == `!%` &&
		value == `"Murphy"` &&
		remain == `.last`)

	path, op, value, remain, _, _, ok = parseQuery(`#(service_roles.#(=="one"))`)
	assert(t, ok &&
		path == `service_roles.#(=="one")` &&
		op == `` &&
		value == `` &&
		remain == ``)

	path, op, value, remain, _, _, ok =
		parseQuery(`#(a\("\"(".#(=="o\"(ne")%"ab\")").remain`)
	assert(t, ok &&
		path == `a\("\"(".#(=="o\"(ne")` &&
		op == "%" &&
		value == `"ab\")"` &&
		remain == `.remain`)
}

func TestParentSubQuery(t *testing.T) {
	var json = `{
		"topology": {
		  "instances": [
			{
			  "service_version": "1.2.3",
			  "service_locale": {"lang": "en"},
			  "service_roles": ["one", "two"]
			},
			{
			  "service_version": "1.2.4",
			  "service_locale": {"lang": "th"},
			  "service_roles": ["three", "four"]
			},
			{
			  "service_version": "1.2.2",
			  "service_locale": {"lang": "en"},
			  "service_roles": ["one"]
			}
		  ]
		}
	  }`
	res := Get(json, `topology.instances.#( service_roles.#(=="one"))#.service_version`)
	// should return two instances
	assert(t, res.String() == `["1.2.3","1.2.2"]`)
}

func TestSingleModifier(t *testing.T) {
	var data = `{"@key": "value"}`
	assert(t, Get(data, "@key").String() == "value")
	assert(t, Get(data, "\\@key").String() == "value")
}

func TestModifiersInMultipaths(t *testing.T) {
	AddModifier("case", func(json, arg string) string {
		if arg == "upper" {
			return strings.ToUpper(json)
		}
		if arg == "lower" {
			return strings.ToLower(json)
		}
		return json
	})
	json := `{"friends": [
		{"age": 44, "first": "Dale", "last": "Murphy"},
		{"age": 68, "first": "Roger", "last": "Craig"},
		{"age": 47, "first": "Jane", "last": "Murphy"}
	]}`

	res := Get(json, `friends.#.{age,first|@case:upper}|@ugly`)
	exp := `[{"age":44,"@case:upper":"DALE"},{"age":68,"@case:upper":"ROGER"},{"age":47,"@case:upper":"JANE"}]`
	assert(t, res.Raw == exp)

	res = Get(json, `{friends.#.{age,first:first|@case:upper}|0.first}`)
	exp = `{"first":"DALE"}`
	assert(t, res.Raw == exp)

	res = Get(readmeJSON, `{"children":children|@case:upper,"name":name.first,"age":age}`)
	exp = `{"children":["SARA","ALEX","JACK"],"name":"Tom","age":37}`
	assert(t, res.Raw == exp)
}

func TestIssue141(t *testing.T) {
	json := `{"data": [{"q": 11, "w": 12}, {"q": 21, "w": 22}, {"q": 31, "w": 32} ], "sql": "some stuff here"}`
	assert(t, Get(json, "data.#").Int() == 3)
	assert(t, Get(json, "data.#.{q}|@ugly").Raw == `[{"q":11},{"q":21},{"q":31}]`)
	assert(t, Get(json, "data.#.q|@ugly").Raw == `[11,21,31]`)
}

func TestChainedModifierStringArgs(t *testing.T) {
	// issue #143
	AddModifier("push", func(json, arg string) string {
		json = strings.TrimSpace(json)
		if len(json) < 2 || !Parse(json).IsArray() {
			return json
		}
		json = strings.TrimSpace(json[1 : len(json)-1])
		if len(json) == 0 {
			return "[" + arg + "]"
		}
		return "[" + json + "," + arg + "]"
	})
	res := Get("[]", `@push:"2"|@push:"3"|@push:{"a":"b","c":["e","f"]}|@push:true|@push:10.23`)
	assert(t, res.String() == `["2","3",{"a":"b","c":["e","f"]},true,10.23]`)
}

func TestFlatten(t *testing.T) {
	json := `[1,[2],[3,4],[5,[6,[7]]],{"hi":"there"},8,[9]]`
	assert(t, Get(json, "@flatten").String() == `[1,2,3,4,5,[6,[7]],{"hi":"there"},8,9]`)
	assert(t, Get(json, `@flatten:{"deep":true}`).String() == `[1,2,3,4,5,6,7,{"hi":"there"},8,9]`)
	assert(t, Get(`{"9999":1234}`, "@flatten").String() == `{"9999":1234}`)
}

func TestJoin(t *testing.T) {
	assert(t, Get(`[{},{}]`, "@join").String() == `{}`)
	assert(t, Get(`[{"a":1},{"b":2}]`, "@join").String() == `{"a":1,"b":2}`)
	assert(t, Get(`[{"a":1,"b":1},{"b":2}]`, "@join").String() == `{"a":1,"b":2}`)
	assert(t, Get(`[{"a":1,"b":1},{"b":2},5,{"c":3}]`, "@join").String() == `{"a":1,"b":2,"c":3}`)
	assert(t, Get(`[{"a":1,"b":1},{"b":2},5,{"c":3}]`, `@join:{"preserve":true}`).String() == `{"a":1,"b":1,"b":2,"c":3}`)
	assert(t, Get(`[{"a":1,"b":1},{"b":2},5,{"c":3}]`, `@join:{"preserve":true}.b`).String() == `1`)
	assert(t, Get(`{"9999":1234}`, "@join").String() == `{"9999":1234}`)
}

func TestValid(t *testing.T) {
	assert(t, Get("[{}", "@valid").Exists() == false)
	assert(t, Get("[{}]", "@valid").Exists() == true)
}

// https://github.com/tidwall/gjson/issues/152
func TestJoin152(t *testing.T) {
	var json = `{
		"distance": 1374.0,
		"validFrom": "2005-11-14",
		"historical": {
		  "type": "Day",
		  "name": "last25Hours",
		  "summary": {
			"units": {
			  "temperature": "C",
			  "wind": "m/s",
			  "snow": "cm",
			  "precipitation": "mm"
			},
			"days": [
			  {
				"time": "2020-02-08",
				"hours": [
				  {
					"temperature": {
					  "min": -2.0,
					  "max": -1.6,
					  "value": -1.6
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 92.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-08T16:00:00+01:00"
				  },
				  {
					"temperature": {
					  "min": -1.7,
					  "max": -1.3,
					  "value": -1.3
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 92.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-08T17:00:00+01:00"
				  },
				  {
					"temperature": {
					  "min": -1.3,
					  "max": -0.9,
					  "value": -1.2
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 91.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-08T18:00:00+01:00"
				  }
				]
			  },
			  {
				"time": "2020-02-09",
				"hours": [
				  {
					"temperature": {
					  "min": -1.7,
					  "max": -0.9,
					  "value": -1.5
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 91.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-09T00:00:00+01:00"
				  },
				  {
					"temperature": {
					  "min": -1.5,
					  "max": 0.9,
					  "value": 0.2
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 67.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-09T01:00:00+01:00"
				  }
				]
			  }
			]
		  }
		}
	  }`

	res := Get(json, "historical.summary.days.#.hours|@flatten|#.humidity.value")
	assert(t, res.Raw == `[92.0,92.0,91.0,91.0,67.0]`)
}

func TestVariousFuzz(t *testing.T) {
	// Issue #192	assert(t, squash(`"000"hello`) == `"000"`)
	assert(t, squash(`"000"`) == `"000"`)
	assert(t, squash(`"000`) == `"000`)
	assert(t, squash(`"`) == `"`)

	assert(t, squash(`[000]hello`) == `[000]`)
	assert(t, squash(`[000]`) == `[000]`)
	assert(t, squash(`[000`) == `[000`)
	assert(t, squash(`[`) == `[`)
	assert(t, squash(`]`) == `]`)

	testJSON := `0.#[[{}]].@valid:"000`
	Get(testJSON, testJSON)

	// Issue #195
	testJSON = `\************************************` +
		`**********{**",**,,**,**,**,**,"",**,**,**,**,**,**,**,**,**,**]`
	Get(testJSON, testJSON)

	// Issue #196
	testJSON = `[#.@pretty.@join:{""[]""preserve"3,"][{]]]`
	Get(testJSON, testJSON)

	// Issue #237
	testJSON1 := `["*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,,,,,,"]`
	testJSON2 := `#[%"*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,*,,,,,,""*,*"]`
	Get(testJSON1, testJSON2)

}

func TestSubpathsWithMultipaths(t *testing.T) {
	const json = `
[
  {"a": 1},
  {"a": 2, "values": ["a", "b", "c", "d", "e"]},
  true,
  ["a", "b", "c", "d", "e"],
  4
]
`
	assert(t, Get(json, `1.values.@ugly`).Raw == `["a","b","c","d","e"]`)
	assert(t, Get(json, `1.values.[0,3]`).Raw == `["a","d"]`)
	assert(t, Get(json, `3.@ugly`).Raw == `["a","b","c","d","e"]`)
	assert(t, Get(json, `3.[0,3]`).Raw == `["a","d"]`)
	assert(t, Get(json, `#.@ugly`).Raw == `[{"a":1},{"a":2,"values":["a","b","c","d","e"]},true,["a","b","c","d","e"],4]`)
	assert(t, Get(json, `#.[0,3]`).Raw == `[[],[],[],["a","d"],[]]`)
}

func TestFlattenRemoveNonExist(t *testing.T) {
	raw := Get("[[1],[2,[[],[3]],[4,[5],[],[[[6]]]]]]", `@flatten:{"deep":true}`).Raw
	assert(t, raw == "[1,2,3,4,5,6]")
}

func TestPipeEmptyArray(t *testing.T) {
	raw := Get("[]", `#(hello)#`).Raw
	assert(t, raw == "[]")
}

func TestEncodedQueryString(t *testing.T) {
	json := `{
		"friends": [
			{"first": "Dale", "last": "Mur\nphy", "age": 44},
			{"first": "Roger", "last": "Craig", "age": 68},
			{"first": "Jane", "last": "Murphy", "age": 47}
		]
	}`
	assert(t, Get(json, `friends.#(last=="Mur\nphy").age`).Int() == 44)
	assert(t, Get(json, `friends.#(last=="Murphy").age`).Int() == 47)
}

func TestTildeQueries(t *testing.T) {
	json := `{
		"vals": [
			{ "a": 1, "b": "data" },
			{ "a": 2, "b": true },
			{ "a": 3, "b": false },
			{ "a": 4, "b": "0" },
			{ "a": 5, "b": 0 },
			{ "a": 6, "b": "1" },
			{ "a": 7, "b": 1 },
			{ "a": 8, "b": "true" },
			{ "a": 9, "b": false },
			{ "a": 10, "b": null },
			{ "a": 11 }
		]
	}`
	trues := Get(json, `vals.#(b==~true)#.a`).Raw
	truesNOT := Get(json, `vals.#(b!=~true)#.a`).Raw
	falses := Get(json, `vals.#(b==~false)#.a`).Raw
	falsesNOT := Get(json, `vals.#(b!=~false)#.a`).Raw
	nulls := Get(json, `vals.#(b==~null)#.a`).Raw
	nullsNOT := Get(json, `vals.#(b!=~null)#.a`).Raw
	exists := Get(json, `vals.#(b==~*)#.a`).Raw
	existsNOT := Get(json, `vals.#(b!=~*)#.a`).Raw

	assert(t, trues == "[2,6,7,8]")
	assert(t, truesNOT == "[1,3,4,5,9,10,11]")
	assert(t, falses == "[3,4,5,9,10,11]")
	assert(t, falsesNOT == "[1,2,6,7,8]")
	assert(t, nulls == "[10,11]")
	assert(t, nullsNOT == "[1,2,3,4,5,6,7,8,9]")
	assert(t, exists == "[1,2,3,4,5,6,7,8,9,10]")
	assert(t, existsNOT == "[11]")
	json = `{
		"vals": [
		  { "a": 1, "b": "something" },
		  { "a": 2, "b": "else" },
		  { "a": 3, "b": false },
		  { "a": 4, "b": "0" },
		  { "a": 5, "b": 0 },
		  { "a": 6, "b": "1" },
		  { "a": 7, "b": 1 },
		  { "a": 8, "b": "true" },
		  { "a": 9, "b": false },
		  { "a": 10, "b": null },
		  { "a": 11 }
		],
		"anything": "else"
	}`
	trues = Get(json, `vals.#(b==~true)#.a`).Raw
	truesNOT = Get(json, `vals.#(b!=~true)#.a`).Raw
	falses = Get(json, `vals.#(b==~false)#.a`).Raw
	falsesNOT = Get(json, `vals.#(b!=~false)#.a`).Raw
	nulls = Get(json, `vals.#(b==~null)#.a`).Raw
	nullsNOT = Get(json, `vals.#(b!=~null)#.a`).Raw
	exists = Get(json, `vals.#(b==~*)#.a`).Raw
	existsNOT = Get(json, `vals.#(b!=~*)#.a`).Raw
	assert(t, trues == "[6,7,8]")
	assert(t, truesNOT == "[1,2,3,4,5,9,10,11]")
	assert(t, falses == "[3,4,5,9,10,11]")
	assert(t, falsesNOT == "[1,2,6,7,8]")
	assert(t, nulls == "[10,11]")
	assert(t, nullsNOT == "[1,2,3,4,5,6,7,8,9]")
	assert(t, exists == "[1,2,3,4,5,6,7,8,9,10]")
	assert(t, existsNOT == "[11]")
}

func TestModifierDoubleQuotes(t *testing.T) {
	josn := `{
		"data": [
		  {
			"name": "Product P4",
			"productId": "1bb3",
			"vendorId": "10de"
		  },
		  {
			"name": "Product P4",
			"productId": "1cc3",
			"vendorId": "20de"
		  },
		  {
			"name": "Product P4",
			"productId": "1dd3",
			"vendorId": "30de"
		  }
		]
	  }`
	AddModifier("string", func(josn, arg string) string {
		return strconv.Quote(josn)
	})

	res := Get(josn, "data.#.{name,value:{productId,vendorId}.@string.@ugly}")

	assert(t, res.Raw == `[`+
		`{"name":"Product P4","value":"{\"productId\":\"1bb3\",\"vendorId\":\"10de\"}"},`+
		`{"name":"Product P4","value":"{\"productId\":\"1cc3\",\"vendorId\":\"20de\"}"},`+
		`{"name":"Product P4","value":"{\"productId\":\"1dd3\",\"vendorId\":\"30de\"}"}`+
		`]`)

}

func TestIndexes(t *testing.T) {
	var exampleJSON = `{
		"vals": [
			[1,66,{test: 3}],
			[4,5,[6]]
		],
		"objectArray":[
			{"first": "Dale", "age": 44},
			{"first": "Roger", "age": 68},
		]
	}`

	testCases := []struct {
		path     string
		expected []string
	}{
		{
			`vals.#.1`,
			[]string{`6`, "5"},
		},
		{
			`vals.#.2`,
			[]string{"{", "["},
		},
		{
			`objectArray.#(age>43)#.first`,
			[]string{`"`, `"`},
		},
		{
			`objectArray.@reverse.#.first`,
			nil,
		},
	}

	for _, tc := range testCases {
		r := Get(exampleJSON, tc.path)

		assert(t, len(r.Indexes) == len(tc.expected))

		for i, a := range r.Indexes {
			assert(t, string(exampleJSON[a]) == tc.expected[i])
		}
	}
}

func TestIndexesMatchesRaw(t *testing.T) {
	var exampleJSON = `{
		"objectArray":[
			{"first": "Jason", "age": 41},
			{"first": "Dale", "age": 44},
			{"first": "Roger", "age": 68},
			{"first": "Mandy", "age": 32}
		]
	}`
	r := Get(exampleJSON, `objectArray.#(age>43)#.first`)
	assert(t, len(r.Indexes) == 2)
	assert(t, Parse(exampleJSON[r.Indexes[0]:]).String() == "Dale")
	assert(t, Parse(exampleJSON[r.Indexes[1]:]).String() == "Roger")
	r = Get(exampleJSON, `objectArray.#(age>43)#`)
	assert(t, Parse(exampleJSON[r.Indexes[0]:]).Get("first").String() == "Dale")
	assert(t, Parse(exampleJSON[r.Indexes[1]:]).Get("first").String() == "Roger")
}

func TestIssue240(t *testing.T) {
	nonArrayData := `{"jsonrpc":"2.0","method":"subscription","params":
		{"channel":"funny_channel","data":
			{"name":"Jason","company":"good_company","number":12345}
		}
	}`
	parsed := Parse(nonArrayData)
	assert(t, len(parsed.Get("params.data").Array()) == 1)

	arrayData := `{"jsonrpc":"2.0","method":"subscription","params":
		{"channel":"funny_channel","data":[
			{"name":"Jason","company":"good_company","number":12345}
		]}
	}`
	parsed = Parse(arrayData)
	assert(t, len(parsed.Get("params.data").Array()) == 1)
}

func TestKeysValuesModifier(t *testing.T) {
	var json = `{
		"1300014": {
		  "code": "1300014",
		  "price": 59.18,
		  "symbol": "300014",
		  "update": "2020/04/15 15:59:54",
		},
		"1300015": {
		  "code": "1300015",
		  "price": 43.31,
		  "symbol": "300015",
		  "update": "2020/04/15 15:59:54",
		}
	  }`
	assert(t, Get(json, `@keys`).String() == `["1300014","1300015"]`)
	assert(t, Get(``, `@keys`).String() == `[]`)
	assert(t, Get(`"hello"`, `@keys`).String() == `[null]`)
	assert(t, Get(`[]`, `@keys`).String() == `[]`)
	assert(t, Get(`[1,2,3]`, `@keys`).String() == `[null,null,null]`)

	assert(t, Get(json, `@values.#.code`).String() == `["1300014","1300015"]`)
	assert(t, Get(``, `@values`).String() == `[]`)
	assert(t, Get(`"hello"`, `@values`).String() == `["hello"]`)
	assert(t, Get(`[]`, `@values`).String() == `[]`)
	assert(t, Get(`[1,2,3]`, `@values`).String() == `[1,2,3]`)
}

func TestNaNInf(t *testing.T) {
	json := `[+Inf,-Inf,Inf,iNF,-iNF,+iNF,NaN,nan,nAn,-0,+0]`
	raws := []string{"+Inf", "-Inf", "Inf", "iNF", "-iNF", "+iNF", "NaN", "nan",
		"nAn", "-0", "+0"}
	nums := []float64{math.Inf(+1), math.Inf(-1), math.Inf(0), math.Inf(0),
		math.Inf(-1), math.Inf(+1), math.NaN(), math.NaN(), math.NaN(),
		math.Copysign(0, -1), 0}

	assert(t, int(Get(json, `#`).Int()) == len(raws))
	for i := 0; i < len(raws); i++ {
		r := Get(json, fmt.Sprintf("%d", i))
		assert(t, r.Raw == raws[i])
		assert(t, r.Num == nums[i] || (math.IsNaN(r.Num) && math.IsNaN(nums[i])))
		assert(t, r.Type == Number)
	}

	var i int
	Parse(json).ForEach(func(_, r Result) bool {
		assert(t, r.Raw == raws[i])
		assert(t, r.Num == nums[i] || (math.IsNaN(r.Num) && math.IsNaN(nums[i])))
		assert(t, r.Type == Number)
		i++
		return true
	})

	// Parse should also return valid numbers
	assert(t, math.IsNaN(Parse("nan").Float()))
	assert(t, math.IsNaN(Parse("NaN").Float()))
	assert(t, math.IsNaN(Parse(" NaN").Float()))
	assert(t, math.IsInf(Parse("+inf").Float(), +1))
	assert(t, math.IsInf(Parse("-inf").Float(), -1))
	assert(t, math.IsInf(Parse("+INF").Float(), +1))
	assert(t, math.IsInf(Parse("-INF").Float(), -1))
	assert(t, math.IsInf(Parse(" +INF").Float(), +1))
	assert(t, math.IsInf(Parse(" -INF").Float(), -1))
}

func TestEmptyValueQuery(t *testing.T) {
	// issue: https://github.com/tidwall/gjson/issues/246
	assert(t, Get(
		`["ig","","tw","fb","tw","ig","tw"]`,
		`#(!="")#`).Raw ==
		`["ig","tw","fb","tw","ig","tw"]`)
	assert(t, Get(
		`["ig","","tw","fb","tw","ig","tw"]`,
		`#(!=)#`).Raw ==
		`["ig","tw","fb","tw","ig","tw"]`)
}

func TestParseIndex(t *testing.T) {
	assert(t, Parse(`{}`).Index == 0)
	assert(t, Parse(` {}`).Index == 1)
	assert(t, Parse(` []`).Index == 1)
	assert(t, Parse(` true`).Index == 1)
	assert(t, Parse(` false`).Index == 1)
	assert(t, Parse(` null`).Index == 1)
	assert(t, Parse(` +inf`).Index == 1)
	assert(t, Parse(` -inf`).Index == 1)
}

func TestRevSquash(t *testing.T) {
	assert(t, revSquash(` {}`) == `{}`)
	assert(t, revSquash(` }`) == ` }`)
	assert(t, revSquash(` [123]`) == `[123]`)
	assert(t, revSquash(` ,123,123]`) == ` ,123,123]`)
	assert(t, revSquash(` hello,[[true,false],[0,1,2,3,5],[123]]`) == `[[true,false],[0,1,2,3,5],[123]]`)
	assert(t, revSquash(` "hello"`) == `"hello"`)
	assert(t, revSquash(` "hel\\lo"`) == `"hel\\lo"`)
	assert(t, revSquash(` "hel\\"lo"`) == `"lo"`)
	assert(t, revSquash(` "hel\\\"lo"`) == `"hel\\\"lo"`)
	assert(t, revSquash(`hel\\\"lo"`) == `hel\\\"lo"`)
	assert(t, revSquash(`\"hel\\\"lo"`) == `\"hel\\\"lo"`)
	assert(t, revSquash(`\\\"hel\\\"lo"`) == `\\\"hel\\\"lo"`)
	assert(t, revSquash(`\\\\"hel\\\"lo"`) == `"hel\\\"lo"`)
	assert(t, revSquash(`hello"`) == `hello"`)
	json := `true,[0,1,"sadf\"asdf",{"hi":["hello","t\"\"u",{"a":"b"}]},9]`
	assert(t, revSquash(json) == json[5:])
	assert(t, revSquash(json[:len(json)-3]) == `{"hi":["hello","t\"\"u",{"a":"b"}]}`)
	assert(t, revSquash(json[:len(json)-4]) == `["hello","t\"\"u",{"a":"b"}]`)
	assert(t, revSquash(json[:len(json)-5]) == `{"a":"b"}`)
	assert(t, revSquash(json[:len(json)-6]) == `"b"`)
	assert(t, revSquash(json[:len(json)-10]) == `"a"`)
	assert(t, revSquash(json[:len(json)-15]) == `"t\"\"u"`)
	assert(t, revSquash(json[:len(json)-24]) == `"hello"`)
	assert(t, revSquash(json[:len(json)-33]) == `"hi"`)
	assert(t, revSquash(json[:len(json)-39]) == `"sadf\"asdf"`)
}

const readmeJSON = `
{
  "name": {"first": "Tom", "last": "Anderson"},
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
    {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
    {"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
    {"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
  ]
}
`

func TestQueryGetPath(t *testing.T) {
	assert(t, strings.Join(
		Get(readmeJSON, "friends.#.first").Paths(readmeJSON), " ") ==
		"friends.0.first friends.1.first friends.2.first")
	assert(t, strings.Join(
		Get(readmeJSON, "friends.#(last=Murphy)").Paths(readmeJSON), " ") ==
		"")
	assert(t, Get(readmeJSON, "friends.#(last=Murphy)").Path(readmeJSON) ==
		"friends.0")
	assert(t, strings.Join(
		Get(readmeJSON, "friends.#(last=Murphy)#").Paths(readmeJSON), " ") ==
		"friends.0 friends.2")
	arr := Get(readmeJSON, "friends.#.first").Array()
	for i := 0; i < len(arr); i++ {
		assert(t, arr[i].Path(readmeJSON) == fmt.Sprintf("friends.%d.first", i))
	}
}

func TestStaticJSON(t *testing.T) {
	json := `{
		"name": {"first": "Tom", "last": "Anderson"}
	}`
	assert(t, Get(json,
		`"bar"`).Raw ==
		``)
	assert(t, Get(json,
		`!"bar"`).Raw ==
		`"bar"`)
	assert(t, Get(json,
		`!{"name":{"first":"Tom"}}.{name.first}.first`).Raw ==
		`"Tom"`)
	assert(t, Get(json,
		`{name.last,"foo":!"bar"}`).Raw ==
		`{"last":"Anderson","foo":"bar"}`)
	assert(t, Get(json,
		`{name.last,"foo":!{"a":"b"},"that"}`).Raw ==
		`{"last":"Anderson","foo":{"a":"b"}}`)
	assert(t, Get(json,
		`{name.last,"foo":!{"c":"d"},!"that"}`).Raw ==
		`{"last":"Anderson","foo":{"c":"d"},"_":"that"}`)
	assert(t, Get(json,
		`[!true,!false,!null,!inf,!nan,!hello,{"name":!"andy",name.last},+inf,!["any","thing"]]`).Raw ==
		`[true,false,null,inf,nan,{"name":"andy","last":"Anderson"},["any","thing"]]`,
	)
}

func TestArrayKeys(t *testing.T) {
	N := 100
	json := "["
	for i := 0; i < N; i++ {
		if i > 0 {
			json += ","
		}
		json += fmt.Sprint(i)
	}
	json += "]"
	var i int
	Parse(json).ForEach(func(key, value Result) bool {
		assert(t, key.String() == fmt.Sprint(i))
		assert(t, key.Int() == int64(i))
		i++
		return true
	})
	assert(t, i == N)
}

func TestToFromStr(t *testing.T) {
	json := `{"Message":"{\"Records\":[{\"eventVersion\":\"2.1\"}]"}`
	res := Get(json, "Message.@fromstr.Records.#.eventVersion.@tostr").Raw
	assert(t, res == `["\"2.1\""]`)
}

func TestGroup(t *testing.T) {
	json := `{"id":["123","456","789"],"val":[2,1]}`
	res := Get(json, "@group").Raw
	assert(t, res == `[{"id":"123","val":2},{"id":"456","val":1},{"id":"789"}]`)

	json = `
{
	"issues": [
	  {
		"fields": {
		  "labels": [
			"milestone_1",
			"group:foo",
			"plan:a",
			"plan:b"
		  ]
		},
		"id": "123"
	  },{
		"fields": {
		  "labels": [
			"milestone_1",
			"group:foo",
			"plan:a",
			"plan"
		  ]
		},
		"id": "456"
	  }
	]
  }
  `
	res = Get(json, `{"id":issues.#.id,"plans":issues.#.fields.labels.#(%"plan:*")#|#.#}|@group|#(plans>=2)#.id`).Raw
	assert(t, res == `["123"]`)
}

func testJSONString(t *testing.T, str string) {
	gjsonString := string(AppendJSONString(nil, str))
	data, err := json.Marshal(str)
	if err != nil {
		panic(123)
	}
	goString := string(data)
	if gjsonString != goString {
		t.Fatal(strconv.Quote(str) + "\n\t" +
			gjsonString + "\n\t" +
			goString + "\n\t<<< MISMATCH >>>")
	}
}

func TestJSONString(t *testing.T) {
	testJSONString(t, "hello")
	testJSONString(t, "he\"llo")
	testJSONString(t, "he\"l\\lo")
	const input = `{"utf8":"Example emoji, KO: \ud83d\udd13, \ud83c\udfc3 ` +
		`OK: \u2764\ufe0f "}`
	value := Get(input, "utf8")
	var s string
	json.Unmarshal([]byte(value.Raw), &s)
	if value.String() != s {
		t.Fatalf("expected '%v', got '%v'", s, value.String())
	}
	testJSONString(t, s)
	testJSONString(t, "R\xfd\xfc\a!\x82eO\x16?_\x0f\x9ab\x1dr")
	testJSONString(t, "_\xb9\v\xad\xb3|X!\xb6\xd9U&\xa4\x1a\x95\x04")
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	start := time.Now()
	var buf [16]byte
	for time.Since(start) < time.Second*2 {
		if _, err := rng.Read(buf[:]); err != nil {
			t.Fatal(err)
		}
		testJSONString(t, string(buf[:]))
	}
}

func TestIndexAtSymbol(t *testing.T) {
	json := `{
		"@context": {
		  "rdfs": "http://www.w3.org/2000/01/rdf-schema#",
		  "@vocab": "http://schema.org/",
		  "sh": "http://www.w3.org/ns/shacl#"
		}
	}`
	assert(t, Get(json, "@context.@vocab").Index == 85)
}

func TestDeepModifierWithOptions(t *testing.T) {
	rawJson := `{"x":[{"y":[{"z":{"b":1, "c": 2, "a": 3}}]}]}`
	jsonPathExpr := `x.#.y.#.z.@pretty:{"sortKeys":true}`
	results := GetManyBytes([]byte(rawJson), jsonPathExpr)
	assert(t, len(results) == 1)
	actual := results[0].Raw
	expected := `[[{
  "a": 3,
  "b": 1,
  "c": 2
}
]]`
	if expected != actual {
		t.Fatal(strconv.Quote(rawJson) + "\n\t" +
			expected + "\n\t" +
			actual + "\n\t<<< MISMATCH >>>")
	}
}

func TestIssue301(t *testing.T) {
	json := `{
		"children": ["Sara","Alex","Jack"],
		"fav.movie": ["Deer Hunter"]
	}`

	assert(t, Get(json, `children.0`).String() == "Sara")
	assert(t, Get(json, `children.[0]`).String() == `["Sara"]`)
	assert(t, Get(json, `children.1`).String() == "Alex")
	assert(t, Get(json, `children.[1]`).String() == `["Alex"]`)
	assert(t, Get(json, `children.[10]`).String() == `[]`)
	assert(t, Get(json, `fav\.movie.0`).String() == "Deer Hunter")
	assert(t, Get(json, `fav\.movie.[0]`).String() == `["Deer Hunter"]`)
	assert(t, Get(json, `fav\.movie.1`).String() == "")
	assert(t, Get(json, `fav\.movie.[1]`).String() == "[]")

}

func TestTruthy(t *testing.T) {
	json := `{
		"aye1": true,
		"aye2": 1,
		"aye3": -1,
		"aye4": 0.001,
		"aye5": "aye",
		"aye6": {},
		"aye7": [],
		"aye8": +Inf,
		"aye9": -Inf,

		"nay1": false,
		"nay2": null,
		"nay3": 0,
		"nay4": -0,
		"nay5": "",
		"nay6": NaN,
	}`

	assert(t, Get(json, "aye1").Truthy() == true)
	assert(t, Get(json, "aye2").Truthy() == true)
	assert(t, Get(json, "aye3").Truthy() == true)
	assert(t, Get(json, "aye4").Truthy() == true)
	assert(t, Get(json, "aye5").Truthy() == true)
	assert(t, Get(json, "aye6").Truthy() == true)
	assert(t, Get(json, "aye7").Truthy() == true)
	assert(t, Get(json, "aye8").Truthy() == true)
	assert(t, Get(json, "aye9").Truthy() == true)

	assert(t, Get(json, "nay1").Truthy() == false)
	assert(t, Get(json, "nay2").Truthy() == false)
	assert(t, Get(json, "nay3").Truthy() == false)
	assert(t, Get(json, "nay4").Truthy() == false)
	assert(t, Get(json, "nay5").Truthy() == false)
	assert(t, Get(json, "nay6").Truthy() == false)
}
