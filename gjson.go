// Package gjson provides searching for json strings.
package gjson

import "strconv"

// Type is Result type
type Type int

const (
	// Null is a null json value
	Null Type = iota
	// False is a json false boolean
	False
	// Number is json number
	Number
	// String is a json string
	String
	// True is a json true boolean
	True
	// JSON is a raw block of JSON
	JSON
)

// Result represents a json value that is returned from Get().
type Result struct {
	// Type is the json type
	Type Type
	// Raw is the raw json
	Raw string
	// Str is the json string
	Str string
	// Num is the json number
	Num float64
}

// String returns a string representation of the value.
func (t Result) String() string {
	switch t.Type {
	default:
		return "null"
	case False:
		return "false"
	case Number:
		return strconv.FormatFloat(t.Num, 'f', -1, 64)
	case String:
		return t.Str
	case JSON:
		return t.Raw
	case True:
		return "true"
	}
}

// Bool returns an boolean representation.
func (t Result) Bool() bool {
	switch t.Type {
	default:
		return false
	case True:
		return true
	case String:
		return t.Str != "" && t.Str != "0"
	case Number:
		return t.Num != 0
	}
}

// Int returns an integer representation.
func (t Result) Int() int64 {
	switch t.Type {
	default:
		return 0
	case True:
		return 1
	case String:
		n, _ := strconv.ParseInt(t.Str, 10, 64)
		return n
	case Number:
		return int64(t.Num)
	}
}

// Float returns an float64 representation.
func (t Result) Float() float64 {
	switch t.Type {
	default:
		return 0
	case True:
		return 1
	case String:
		n, _ := strconv.ParseFloat(t.Str, 64)
		return n
	case Number:
		return t.Num
	}
}

// Array returns back an array of children. The result must be a JSON array.
func (t Result) Array() []Result {
	if t.Type != JSON {
		return nil
	}
	a, _, _, _, _ := t.arrayOrMap('[', false)
	return a
}

//  Map returns back an map of children. The result should be a JSON array.
func (t Result) Map() map[string]Result {
	if t.Type != JSON {
		return map[string]Result{}
	}
	_, _, o, _, _ := t.arrayOrMap('{', false)
	return o
}

// Get searches result for the specified path.
// The result should be a JSON array or object.
func (t Result) Get(path string) Result {
	return Get(t.Raw, path)
}

func (t Result) arrayOrMap(vc byte, valueize bool) (
	[]Result,
	[]interface{},
	map[string]Result,
	map[string]interface{},
	byte,
) {
	var a []Result
	var ai []interface{}
	var o map[string]Result
	var oi map[string]interface{}
	var json = t.Raw
	var i int
	var value Result
	var count int
	var key Result
	if vc == 0 {
		for ; i < len(json); i++ {
			if json[i] == '{' || json[i] == '[' {
				vc = json[i]
				i++
				break
			}
			if json[i] > ' ' {
				goto end
			}
		}
	} else {
		for ; i < len(json); i++ {
			if json[i] == vc {
				i++
				break
			}
			if json[i] > ' ' {
				goto end
			}
		}
	}
	if vc == '{' {
		if valueize {
			oi = make(map[string]interface{})
		} else {
			o = make(map[string]Result)
		}
	} else {
		if valueize {
			ai = make([]interface{}, 0)
		} else {
			a = make([]Result, 0)
		}
	}
	for ; i < len(json); i++ {
		if json[i] <= ' ' {
			continue
		}
		// get next value
		if json[i] == ']' || json[i] == '}' {
			break
		}
		switch json[i] {
		default:
			if (json[i] >= '0' && json[i] <= '9') || json[i] == '-' {
				value.Type = Number
				value.Raw, value.Num = tonum(json[i:])
			} else {
				continue
			}
		case '{', '[':
			value.Type = JSON
			value.Raw = squash(json[i:])
		case 'n':
			value.Type = Null
			value.Raw = tolit(json[i:])
		case 't':
			value.Type = True
			value.Raw = tolit(json[i:])
		case 'f':
			value.Type = False
			value.Raw = tolit(json[i:])
		case '"':
			value.Type = String
			value.Raw, value.Str = tostr(json[i:])
		}
		i += len(value.Raw) - 1

		if vc == '{' {
			if count%2 == 0 {
				key = value
			} else {
				if valueize {
					oi[key.Str] = value.Value()
				} else {
					o[key.Str] = value
				}
			}
			count++
		} else {
			if valueize {
				ai = append(ai, value.Value())
			} else {
				a = append(a, value)
			}
		}
	}
end:
	return a, ai, o, oi, vc
}

// Parse parses the json and returns a result
func Parse(json string) Result {
	var value Result
	for i := 0; i < len(json); i++ {
		if json[i] == '{' || json[i] == '[' {
			value.Type = JSON
			value.Raw = json[i:] // just take the entire raw
			break
		}
		if json[i] <= ' ' {
			continue
		}
		switch json[i] {
		default:
			if (json[i] >= '0' && json[i] <= '9') || json[i] == '-' {
				value.Type = Number
				value.Raw, value.Num = tonum(json[i:])
			} else {
				return Result{}
			}
		case 'n':
			value.Type = Null
			value.Raw = tolit(json[i:])
		case 't':
			value.Type = True
			value.Raw = tolit(json[i:])
		case 'f':
			value.Type = False
			value.Raw = tolit(json[i:])
		case '"':
			value.Type = String
			value.Raw, value.Str = tostr(json[i:])
		}
		break
	}
	return value
}

func squash(json string) string {
	// expects that the lead character is a '[' or '{'
	// squash the value, ignoring all nested arrays and objects.
	// the first '[' or '{' has already been read
	depth := 1
	for i := 1; i < len(json); i++ {
		if json[i] >= '"' && json[i] <= '}' {
			switch json[i] {
			case '"':
				i++
				s2 := i
				for ; i < len(json); i++ {
					if json[i] > '\\' {
						continue
					}
					if json[i] == '"' {
						// look for an escaped slash
						if json[i-1] == '\\' {
							n := 0
							for j := i - 2; j > s2-1; j-- {
								if json[j] != '\\' {
									break
								}
								n++
							}
							if n%2 == 0 {
								continue
							}
						}
						break
					}
				}
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return json[:i+1]
				}
			}
		}
	}
	return json
}

func tonum(json string) (raw string, num float64) {
	for i := 1; i < len(json); i++ {
		// less than dash might have valid characters
		if json[i] <= '-' {
			if json[i] <= ' ' || json[i] == ',' {
				// break on whitespace and comma
				raw = json[:i]
				num, _ = strconv.ParseFloat(raw, 64)
				return
			}
			// could be a '+' or '-'. let's assume so.
			continue
		}
		if json[i] < ']' {
			// probably a valid number
			continue
		}
		if json[i] == 'e' || json[i] == 'E' {
			// allow for exponential numbers
			continue
		}
		// likely a ']' or '}'
		raw = json[:i]
		num, _ = strconv.ParseFloat(raw, 64)
		return
	}
	raw = json
	num, _ = strconv.ParseFloat(raw, 64)
	return
}

func tolit(json string) (raw string) {
	for i := 1; i < len(json); i++ {
		if json[i] <= 'a' || json[i] >= 'z' {
			return json[:i]
		}
	}
	return json
}

func tostr(json string) (raw string, str string) {
	// expects that the lead character is a '"'
	for i := 1; i < len(json); i++ {
		if json[i] > '\\' {
			continue
		}
		if json[i] == '"' {
			return json[:i+1], json[1:i]
		}
		if json[i] == '\\' {
			i++
			for ; i < len(json); i++ {
				if json[i] > '\\' {
					continue
				}
				if json[i] == '"' {
					// look for an escaped slash
					if json[i-1] == '\\' {
						n := 0
						for j := i - 2; j > 0; j-- {
							if json[j] != '\\' {
								break
							}
							n++
						}
						if n%2 == 0 {
							continue
						}
					}
					break
				}
			}
			return json[:i+1], unescape(json[1:i])
		}
	}
	return json, json[1:]
}

// Exists returns true if value exists.
//
//  if gjson.Get(json, "name.last").Exists(){
//		println("value exists")
//  }
func (t Result) Exists() bool {
	return t.Type != Null || len(t.Raw) != 0
}

// Value returns one of these types:
//
//	bool, for JSON booleans
//	float64, for JSON numbers
//	Number, for JSON numbers
//	string, for JSON string literals
//	nil, for JSON null
//
func (t Result) Value() interface{} {
	if t.Type == String {
		return t.Str
	}
	switch t.Type {
	default:
		return nil
	case False:
		return false
	case Number:
		return t.Num
	case JSON:
		_, ai, _, oi, vc := t.arrayOrMap(0, true)
		if vc == '{' {
			return oi
		} else if vc == '[' {
			return ai
		}
		return nil
	case True:
		return true
	}

}

func parseString(json string, i int, raw bool) (int, string, bool, bool) {
	var s = i
	for ; i < len(json); i++ {
		if json[i] > '\\' {
			continue
		}
		if json[i] == '"' {
			if raw {
				return i + 1, json[s-1 : i+1], false, true
			} else {
				return i + 1, json[s:i], false, true
			}
		}
		if json[i] == '\\' {
			i++
			for ; i < len(json); i++ {
				if json[i] > '\\' {
					continue
				}
				if json[i] == '"' {
					// look for an escaped slash
					if json[i-1] == '\\' {
						n := 0
						for j := i - 2; j > 0; j-- {
							if json[j] != '\\' {
								break
							}
							n++
						}
						if n%2 == 0 {
							continue
						}
					}
					if raw {
						return i + 1, json[s-1 : i+1], true, true
					} else {
						return i + 1, json[s:i], true, true
					}
				}
			}
			break
		}
	}
	if raw {
		return i, json[s-1:], false, false
	} else {
		return i, json[s:], false, false
	}
}

func parseNumber(json string, i int) (int, string) {
	var s = i
	i++
	for ; i < len(json); i++ {
		if json[i] <= ' ' || json[i] == ',' || json[i] == ']' || json[i] == '}' {
			return i, json[s:i]
		}
	}
	return i, json[s:]
}

func parseLiteral(json string, i int) (int, string) {
	var s = i
	i++
	for ; i < len(json); i++ {
		if json[i] < 'a' || json[i] > 'z' {
			return i, json[s:i]
		}
	}
	return i, json[s:]
}

func parseArrayPath(path string) (
	part string, npath string, more bool, alogok bool, arrch bool, alogkey string,
) {
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			return path[:i], path[i+1:], true, alogok, arrch, alogkey
		}
		if path[i] == '#' {
			arrch = true
			if i == 0 && len(path) > 1 && path[1] == '.' {
				alogok = true
				alogkey = path[2:]
				path = path[:1]
			}
			continue
		}
	}
	return path, "", false, alogok, arrch, alogkey
}

func parseObjectPath(path string) (
	part string, npath string, wild bool, uc bool, more bool,
) {
	for i := 0; i < len(path); i++ {
		if path[i]&0x60 == 0x60 {
			// alpha lowercase
			continue
		}
		if path[i] == '.' {
			return path[:i], path[i+1:], wild, uc, true
		}
		if path[i] == '*' || path[i] == '?' {
			wild = true
			continue
		}
		if path[i] > 0x7f {
			uc = true
			continue
		}
		if path[i] == '\\' {
			// go into escape mode. this is a slower path that
			// strips off the escape character from the part.
			epart := []byte(path[:i])
			i++
			if i < len(path) {
				epart = append(epart, path[i])
				i++
				for ; i < len(path); i++ {
					if path[i] > 0x7f {
						uc = true
						continue
					}
					if path[i] == '\\' {
						i++
						if i < len(path) {
							epart = append(epart, path[i])
						}
						continue
					} else if path[i] == '.' {
						return string(epart), path[i+1:], wild, uc, true
					} else if path[i] == '*' || path[i] == '?' {
						wild = true
					}
					epart = append(epart, path[i])
				}
			}
			// append the last part
			return string(epart), "", wild, uc, false
		}
	}
	return path, "", wild, uc, false
}

func squashObjectOrArray(json string, i int) (int, string) {
	// expects that the lead character is a '[' or '{'
	// squash the value, ignoring all nested arrays and objects.
	// the first '[' or '{' has already been read
	s := i
	i++
	depth := 1
	for ; i < len(json); i++ {
		if json[i] >= '"' && json[i] <= '}' {
			switch json[i] {
			case '"':
				i++
				s2 := i
				for ; i < len(json); i++ {
					if json[i] > '\\' {
						continue
					}
					if json[i] == '"' {
						// look for an escaped slash
						if json[i-1] == '\\' {
							n := 0
							for j := i - 2; j > s2-1; j-- {
								if json[j] != '\\' {
									break
								}
								n++
							}
							if n%2 == 0 {
								continue
							}
						}
						break
					}
				}
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					i++
					return i, json[s:i]
				}
			}
		}
	}
	return i, json[s:]
}

func parseObject(json string, i int, path string, value *Result) (int, bool) {
	var match, kesc, vesc, ok, hit bool
	var key, val string
	part, npath, wild, uc, more := parseObjectPath(path)
	for i < len(json) {
		for ; i < len(json); i++ {
			if json[i] == '"' {
				i, key, kesc, ok = parseString(json, i+1, false)
				break
			}
			if json[i] == '}' {
				return i + 1, false
			}
		}
		if !ok {
			return i, false
		}
		if wild {
			if kesc {
				match = wildcardMatch(unescape(key), part, uc)
			} else {
				match = wildcardMatch(key, part, uc)
			}
		} else {
			if kesc {
				match = part == unescape(key)
			} else {
				match = part == key
			}
		}
		hit = match && !more
		for ; i < len(json); i++ {
			switch json[i] {
			default:
				continue
			case '"':
				i++
				i, val, vesc, ok = parseString(json, i, true)
				if !ok {
					return i, false
				}
				if hit {
					if vesc {
						value.Str = unescape(val[1 : len(val)-1])
					} else {
						value.Str = val[1 : len(val)-1]
					}
					value.Raw = val
					value.Type = String
					return i, true
				}
			case '{':
				if match && !hit {
					i, hit = parseObject(json, i+1, npath, value)
					if hit {
						return i, true
					}
				} else {
					i, val = squashObjectOrArray(json, i)
					if hit {
						value.Raw = val
						value.Type = JSON
						return i, true
					}
				}
			case '[':
				if match && !hit {
					i, hit = parseArray(json, i+1, npath, value)
					if hit {
						return i, true
					}
				} else {
					i, val = squashObjectOrArray(json, i)
					if hit {
						value.Raw = val
						value.Type = JSON
						return i, true
					}
				}
			case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				i, val = parseNumber(json, i)
				if hit {
					value.Raw = val
					value.Type = Number
					value.Num, _ = strconv.ParseFloat(val, 64)
					return i, true
				}
			case 't', 'f', 'n':
				vc := json[i]
				i, val = parseLiteral(json, i)
				if hit {
					value.Raw = val
					switch vc {
					case 't':
						value.Type = True
					case 'f':
						value.Type = False
					}
					return i, true
				}
			}
			break
		}
	}
	return i, false
}

func parseArray(json string, i int, path string, value *Result) (int, bool) {
	var match, vesc, ok, hit bool
	var val string
	var h int
	var alog []int
	var partidx int
	part, npath, more, alogok, arrch, alogkey := parseArrayPath(path)
	if !arrch {
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			partidx = -1
		} else {
			partidx = int(n)
		}
	}
	for i < len(json) {
		if !arrch {
			match = partidx == h
			hit = match && !more
		}
		h++
		if alogok {
			alog = append(alog, i)
		}
		for ; i < len(json); i++ {
			switch json[i] {
			default:
				continue
			case '"':
				i++
				i, val, vesc, ok = parseString(json, i, true)
				if !ok {
					return i, false
				}
				if hit {
					if alogok {
						break
					}
					if vesc {
						value.Str = unescape(val[1 : len(val)-1])
					} else {
						value.Str = val[1 : len(val)-1]
					}
					value.Raw = val
					value.Type = String
					return i, true
				}
			case '{':
				if match && !hit {
					i, hit = parseObject(json, i+1, npath, value)
					if hit {
						if alogok {
							break
						}
						return i, true
					}
				} else {
					i, val = squashObjectOrArray(json, i)
					if hit {
						if alogok {
							break
						}
						value.Raw = val
						value.Type = JSON
						return i, true
					}
				}
			case '[':
				if match && !hit {
					i, hit = parseArray(json, i+1, npath, value)
					if hit {
						if alogok {
							break
						}
						return i, true
					}
				} else {
					i, val = squashObjectOrArray(json, i)
					if hit {
						if alogok {
							break
						}
						value.Raw = val
						value.Type = JSON
						return i, true
					}
				}
			case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				i, val = parseNumber(json, i)
				if hit {
					if alogok {
						break
					}
					value.Raw = val
					value.Type = Number
					value.Num, _ = strconv.ParseFloat(val, 64)
					return i, true
				}
			case 't', 'f', 'n':
				vc := json[i]
				i, val = parseLiteral(json, i)
				if hit {
					if alogok {
						break
					}
					value.Raw = val
					switch vc {
					case 't':
						value.Type = True
					case 'f':
						value.Type = False
					}
					return i, true
				}
			case ']':
				// TODO... '#' counter?
				if arrch && part == "#" {
					if alogok {
						var jsons = make([]byte, 0, 64)
						jsons = append(jsons, '[')
						for j := 0; j < len(alog); j++ {
							res := Get(json[alog[j]:], alogkey)
							if res.Exists() {
								if j > 0 {
									jsons = append(jsons, ',')
								}
								jsons = append(jsons, []byte(res.Raw)...)
							}
						}
						jsons = append(jsons, ']')
						value.Type = JSON
						value.Raw = string(jsons)
						return i + 1, true
					} else {
						if alogok {
							break
						}
						value.Raw = val
						value.Type = Number
						value.Num = float64(h - 1)
						return i + 1, true
					}
				}
				return i + 1, false
			}
			break
		}
	}
	return i, false
}

// Get searches json for the specified path.
// A path is in dot syntax, such as "name.last" or "age".
// This function expects that the json is well-formed, and does not validate.
// Invalid json will not panic, but it may return back unexpected results.
// When the value is found it's returned immediately.
//
// A path is a series of keys seperated by a dot.
// A key may contain special wildcard characters '*' and '?'.
// To access an array value use the index as the key.
// To get the number of elements in an array or to access a child path, use the '#' character.
// The dot and wildcard character can be escaped with '\'.
//
//  {
//    "name": {"first": "Tom", "last": "Anderson"},
//    "age":37,
//    "children": ["Sara","Alex","Jack"],
//    "friends": [
//      {"first": "James", "last": "Murphy"},
//      {"first": "Roger", "last": "Craig"}
//    ]
//  }
//  "name.last"          >> "Anderson"
//  "age"                >> 37
//  "children.#"         >> 3
//  "children.1"         >> "Alex"
//  "child*.2"           >> "Jack"
//  "c?ildren.0"         >> "Sara"
//  "friends.#.first"    >> [ "James", "Roger" ]
//
func Get(json, path string) Result {
	var i int
	var value Result
	for ; i < len(json); i++ {
		if json[i] == '{' {
			i++
			parseObject(json, i, path, &value)
			break
		}
		if json[i] == '[' {
			i++
			parseArray(json, i, path, &value)
			break
		}
	}
	return value
}

// unescape unescapes a string
func unescape(json string) string { //, error) {
	var str = make([]byte, 0, len(json))
	for i := 0; i < len(json); i++ {
		switch {
		default:
			str = append(str, json[i])
		case json[i] < ' ':
			return "" //, errors.New("invalid character in string")
		case json[i] == '\\':
			i++
			if i >= len(json) {
				return "" //, errors.New("invalid escape sequence")
			}
			switch json[i] {
			default:
				return "" //, errors.New("invalid escape sequence")
			case '\\':
				str = append(str, '\\')
			case '/':
				str = append(str, '/')
			case 'b':
				str = append(str, '\b')
			case 'f':
				str = append(str, '\f')
			case 'n':
				str = append(str, '\n')
			case 'r':
				str = append(str, '\r')
			case 't':
				str = append(str, '\t')
			case '"':
				str = append(str, '"')
			case 'u':
				if i+5 > len(json) {
					return "" //, errors.New("invalid escape sequence")
				}
				i++
				// extract the codepoint
				var code int
				for j := i; j < i+4; j++ {
					switch {
					default:
						return "" //, errors.New("invalid escape sequence")
					case json[j] >= '0' && json[j] <= '9':
						code += (int(json[j]) - '0') << uint(12-(j-i)*4)
					case json[j] >= 'a' && json[j] <= 'f':
						code += (int(json[j]) - 'a' + 10) << uint(12-(j-i)*4)
					case json[j] >= 'a' && json[j] <= 'f':
						code += (int(json[j]) - 'a' + 10) << uint(12-(j-i)*4)
					}
				}
				str = append(str, []byte(string(code))...)
				i += 3 // only 3 because we will increment on the for-loop
			}
		}
	}
	return string(str) //, nil
}

// Less return true if a token is less than another token.
// The caseSensitive paramater is used when the tokens are Strings.
// The order when comparing two different type is:
//
//  Null < False < Number < String < True < JSON
//
func (t Result) Less(token Result, caseSensitive bool) bool {
	if t.Type < token.Type {
		return true
	}
	if t.Type > token.Type {
		return false
	}
	if t.Type == String {
		if caseSensitive {
			return t.Str < token.Str
		}
		return stringLessInsensitive(t.Str, token.Str)
	}
	if t.Type == Number {
		return t.Num < token.Num
	}
	return t.Raw < token.Raw
}

func stringLessInsensitive(a, b string) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] >= 'A' && a[i] <= 'Z' {
			if b[i] >= 'A' && b[i] <= 'Z' {
				// both are uppercase, do nothing
				if a[i] < b[i] {
					return true
				} else if a[i] > b[i] {
					return false
				}
			} else {
				// a is uppercase, convert a to lowercase
				if a[i]+32 < b[i] {
					return true
				} else if a[i]+32 > b[i] {
					return false
				}
			}
		} else if b[i] >= 'A' && b[i] <= 'Z' {
			// b is uppercase, convert b to lowercase
			if a[i] < b[i]+32 {
				return true
			} else if a[i] > b[i]+32 {
				return false
			}
		} else {
			// neither are uppercase
			if a[i] < b[i] {
				return true
			} else if a[i] > b[i] {
				return false
			}
		}
	}
	return len(a) < len(b)
}

// wilcardMatch returns true if str matches pattern. This is a very
// simple wildcard match where '*' matches on any number characters
// and '?' matches on any one character.
func wildcardMatch(str, pattern string, uc bool) bool {
	if pattern == "*" {
		return true
	}
	if !uc {
		return deepMatch(str, pattern)
	}
	rstr := make([]rune, 0, len(str))
	rpattern := make([]rune, 0, len(pattern))
	for _, r := range str {
		rstr = append(rstr, r)
	}
	for _, r := range pattern {
		rpattern = append(rpattern, r)
	}
	return deepMatchRune(rstr, rpattern)
}
func deepMatch(str, pattern string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		default:
			if len(str) == 0 || str[0] != pattern[0] {
				return false
			}
		case '?':
			if len(str) == 0 {
				return false
			}
		case '*':
			return deepMatch(str, pattern[1:]) ||
				(len(str) > 0 && deepMatch(str[1:], pattern))
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0 && len(pattern) == 0
}
func deepMatchRune(str, pattern []rune) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		default:
			if len(str) == 0 || str[0] != pattern[0] {
				return false
			}
		case '?':
			if len(str) == 0 {
				return false
			}
		case '*':
			return deepMatchRune(str, pattern[1:]) ||
				(len(str) > 0 && deepMatchRune(str[1:], pattern))
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0 && len(pattern) == 0
}
