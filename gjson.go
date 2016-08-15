// Package gjson provides searching for json strings.
package gjson

import "strconv"

// Type is Result type
type Type byte

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
	switch t.Type {
	default:
		return nil
	case False:
		return false
	case Number:
		return t.Num
	case String:
		return t.Str
	case JSON:
		return t.Raw
	case True:
		return true
	}
}

type part struct {
	wild bool
	key  string
}

type frame struct {
	key   string
	count int
	stype byte
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
// To get the number of elements in an array use the '#' character.
// The dot and wildcard character can be escaped with '\'.
//
//  {
//    "name": {"first": "Tom", "last": "Anderson"},
//    "age":37,
//    "children": ["Sara","Alex","Jack"]
//  }
//  "name.last"          >> "Anderson"
//  "age"                >> 37
//  "children.#"         >> 3
//  "children.1"         >> "Alex"
//  "child*.2"           >> "Jack"
//  "c?ildren.0"         >> "Sara"
//
func Get(json string, path string) Result {
	var s int
	var wild bool
	var parts = make([]part, 0, 4)

	// do nothing when no path specified
	if len(path) == 0 {
		return Result{} // nothing
	}

	// parse the path. just split on the dot
	for i := 0; i < len(path); i++ {
	next_part:
		if path[i] == '\\' {
			// go into escape mode
			epart := []byte(path[s:i])
			i++
			if i < len(path) {
				epart = append(epart, path[i])
				i++
				for ; i < len(path); i++ {
					if path[i] == '\\' {
						i++
						if i < len(path) {
							epart = append(epart, path[i])
						}
						continue
					} else if path[i] == '.' {
						parts = append(parts, part{wild: wild, key: string(epart)})
						if wild {
							wild = false
						}
						s = i + 1
						i++
						goto next_part
					} else if path[i] == '*' || path[i] == '?' {
						wild = true
					}
					epart = append(epart, path[i])
				}
			}
			parts = append(parts, part{wild: wild, key: string(epart)})
			goto end_parts
		} else if path[i] == '.' {
			parts = append(parts, part{wild: wild, key: path[s:i]})
			if wild {
				wild = false
			}
			s = i + 1
		} else if path[i] == '*' || path[i] == '?' {
			wild = true
		}
	}
	parts = append(parts, part{wild: wild, key: path[s:]})
end_parts:

	var i, depth int
	var squashed string
	var f frame
	var matched bool
	var stack = make([]frame, 0, 4)

	depth = 1

	// look for first delimiter
	for ; i < len(json); i++ {
		if json[i] > ' ' {
			if json[i] == '{' {
				f.stype = '{'
			} else if json[i] == '[' {
				f.stype = '['
			} else {
				// not a valid type
				return Result{}
			}
			i++
			break
		}
	}

	stack = append(stack, f)

	// search for key
read_key:
	if f.stype == '[' {
		f.key = strconv.FormatInt(int64(f.count), 10)
		f.count++
	} else {
		for ; i < len(json); i++ {
			if json[i] == '"' {
				//read to end of key
				i++
				// readstr
				// the first double-quote has already been read
				s = i
				for ; i < len(json); i++ {
					if json[i] == '"' {
						f.key = json[s:i]
						i++
						break
					}
					if json[i] == '\\' {
						i++
						for ; i < len(json); i++ {
							if json[i] == '"' {
								// look for an escaped slash
								if json[i-1] == '\\' {
									n := 0
									for j := i - 2; j > s-1; j-- {
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
						f.key = unescape(json[s:i])
						i++
						break
					}
				}
				break
			}
		}
	}
	// end readstr

	// we have a brand new key.
	// is it the key that we are looking for?
	if parts[depth-1].wild {
		// it's a wildcard path element
		matched = wildcardMatch(f.key, parts[depth-1].key)
	} else {
		matched = parts[depth-1].key == f.key
	}

	// read to the value token
	// there's likely a colon here, but who cares. just burn past it.
	var val string
	var vc byte
	for ; i < len(json); i++ {
		if json[i] < '"' { // control character
			continue
		}
		if json[i] < '-' { // string
			i++
			// we read the val below
			vc = '"'
			goto proc_val
		}
		if json[i] < '[' { // number
			if json[i] == ':' {
				continue
			}
			vc = '0'
			s = i
			i++
			// look for characters that cannot be in a number
			for ; i < len(json); i++ {
				switch json[i] {
				default:
					continue
				case ' ', '\t', '\r', '\n', ',', ']', '}':
				}
				break
			}
			val = json[s:i]
			goto proc_val
		}
		if json[i] < ']' { // '['
			i++
			vc = '['
			goto proc_delim
		}
		if json[i] < 'u' { // true, false, null
			vc = json[i]
			s = i
			i++
			for ; i < len(json); i++ {
				// let's pick up any character. it doesn't matter.
				if json[i] < 'a' || json[i] > 'z' {
					break
				}
			}
			val = json[s:i]
			goto proc_val
		}
		// must be an open objet
		i++
		vc = '{'
		goto proc_delim
	}

	// sanity check before we move on
	if i >= len(json) {
		return Result{}
	}

proc_delim:
	if (matched && depth == len(parts)) || !matched {
		// -- BEGIN SQUASH -- //
		// squash the value, ignoring all nested arrays and objects.
		s = i - 1
		// the first '[' or '{' has already been read
		depth := 1
		for ; i < len(json); i++ {
			if json[i] >= '"' && json[i] <= '}' {
				if json[i] == '{' || json[i] == '[' {
					depth++
				} else if json[i] == '}' || json[i] == ']' {
					depth--
					if depth == 0 {
						i++
						break
					}
				} else if json[i] == '"' {
					i++
					s2 := i
					for ; i < len(json); i++ {
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
					if i == len(json) {
						break
					}
				}
			}
		}
		squashed = json[s:i]
		// -- END SQUASH -- //
	}

	// process the value
proc_val:
	if matched {
		// hit, that's good!
		if depth == len(parts) {
			var value Result
			value.Raw = val
			switch vc {
			case '{', '[':
				value.Raw = squashed
				value.Type = JSON
			case 'n':
				value.Type = Null
			case 't':
				value.Type = True
			case 'f':
				value.Type = False
			case '"':
				value.Type = String
				// readstr
				// the val has not been read yet
				// the first double-quote has already been read
				s = i
				for ; i < len(json); i++ {
					if json[i] == '"' {
						value.Str = json[s:i]
						i++
						break
					}
					if json[i] == '\\' {
						i++
						for ; i < len(json); i++ {
							if json[i] == '"' {
								// look for an escaped slash
								if json[i-1] == '\\' {
									n := 0
									for j := i - 2; j > s-1; j-- {
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
						value.Str = unescape(json[s:i])
						i++
						break
					}
				}
				// end readstr
			case '0':
				value.Type = Number
				value.Num, _ = strconv.ParseFloat(val, 64)
			}
			return value
			//} else if vc != '{' {
			//  can only deep search objects
			//	return Result{}
		} else {
			f.stype = vc
			f.count = 0
			stack = append(stack, f)
			depth++
			goto read_key
		}
	}
	if vc == '"' {
		// readstr
		// the val has not been read yet. we can read and throw away.
		// the first double-quote has already been read
		s = i
		for ; i < len(json); i++ {
			if json[i] == '"' {
				// look for an escaped slash
				if json[i-1] == '\\' {
					n := 0
					for j := i - 2; j > s-1; j-- {
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
		i++
		// end readstr
	}

	// read to the comma or end of object
	for ; i < len(json); i++ {
		switch json[i] {
		case '}', ']':
			if parts[depth-1].key == "#" {
				return Result{Type: Number, Num: float64(f.count)}
			}
			// step the stack back
			depth--
			if depth == 0 {
				return Result{}
			}
			stack = stack[:len(stack)-1]
			f = stack[len(stack)-1]
		case ',':
			i++
			goto read_key
		}
	}
	return Result{}
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
	switch t.Type {
	default:
		return t.Raw < token.Raw
	case String:
		if caseSensitive {
			return t.Str < token.Str
		}
		return stringLessInsensitive(t.Str, token.Str)
	case Number:
		return t.Num < token.Num
	}
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
func wildcardMatch(str, pattern string) bool {
	if pattern == "*" {
		return true
	}
	return deepMatch(str, pattern)
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
			return wildcardMatch(str, pattern[1:]) ||
				(len(str) > 0 && wildcardMatch(str[1:], pattern))
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0 && len(pattern) == 0
}
