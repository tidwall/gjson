// +build gofuzz

package gjson

// FuzzParse tests JSON parsing
func FuzzParse(fuzz []byte) int {
	valid := ValidBytes(fuzz)
	if !valid {
		return 0
	}
	ParseBytes(fuzz)
	return 1
}

// FuzzPath tests search for GJSON path
func FuzzPath(fuzz []byte) int {
	if len(fuzz) < 3 {
		return -1
	}
	length := uint8(fuzz[0])
	if length == 0 {
		return -1
	}
	if len(fuzz) < int(length)+2 {
		return -1
	}
	path := string(fuzz[1 : length+1])
	json := fuzz[length+1:]
	valid := ValidBytes(json)
	if !valid {
		return 0
	}
	GetBytes(json, path)
	return 1
}
