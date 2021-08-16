package jsonschema

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"strings"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

var (
	trueLiteral  = []byte("true")
	falseLiteral = []byte("false")
	nullLiteral  = []byte("null")
)

var regexpExtraSpaceChars = string('\ufeff') + string('\u000b') + string('\u00a0') + string('\u2029') + string('\u2003')

var regexpControlChars = map[string]string{
	`\c@`: `\000`,
	`\cA`: `\001`,
	`\cB`: `\002`,
	`\cC`: `\003`,
	`\cD`: `\004`,
	`\cE`: `\005`,
	`\cF`: `\006`,
	`\cG`: `\007`,
	`\cH`: `\008`,
	`\cI`: `\009`,
	`\cJ`: `\00A`,
	`\cK`: `\00B`,
	`\cL`: `\00C`,
	`\cM`: `\00D`,
	`\cN`: `\00E`,
	`\cO`: `\00F`,
	`\cP`: `\010`,
	`\cQ`: `\011`,
	`\cR`: `\012`,
	`\cS`: `\013`,
	`\cT`: `\014`,
	`\cU`: `\015`,
	`\cV`: `\016`,
	`\cW`: `\017`,
	`\cX`: `\018`,
	`\cY`: `\019`,
	`\cZ`: `\01A`,
	`\c[`: `\01B`,
	`\c\`: `\01C`,
	`\c]`: `\01D`,
	`\c^`: `\01E`,
	`\c_`: `\01F`,
}

func addError(err, errs error) error {
	if err == nil {
		return errs
	}
	if errs == nil {
		return err
	}

	return fmt.Errorf("%w\n%s", errs, err.Error())
}

func ConvertRegexp(re string) string {
	re = strings.ReplaceAll(re, `\w`, `\pL`)
	// re = strings.ReplaceAll(re, `\w`, `[0-9A-Za-z_]`)
	// re = strings.ReplaceAll(re, `\W`, `[^0-9A-Za-z_]`)
	re = strings.ReplaceAll(re, `\d`, `\pN`)
	// re = strings.ReplaceAll(re, `\d`, `[0-9]`)
	// re = strings.ReplaceAll(re, `\D`, `[^0-9]`)
	re = strings.ReplaceAll(re, `\s`, `[\s`+regexpExtraSpaceChars+`]`)
	re = strings.ReplaceAll(re, `\S`, `[^\s`+regexpExtraSpaceChars+`]`)

	// Replace control escape characters with their unicod equivalent
	for cc, unicode := range regexpControlChars {
		re = strings.ReplaceAll(re, cc, unicode)
		re = strings.ReplaceAll(re, strings.ToLower(cc), unicode)
	}

	return re
}

func IsInteger(val []byte) bool {
	floatVal, ok := new(big.Float).SetString(string(val))
	if !ok {
		return false
	}

	return floatVal.IsInt()
}

// This should more or less match what jsonparser detects.
// Unfortunately jsonparser doesn't expose it's detect type function.
func DetectJSONType(data []byte) ValueType {
	// Empty
	if len(data) == 0 {
		return Unknown
	}

	// Object
	if data[0] == '{' && data[len(data)-1] == '}' {
		return Object
	}

	// Array
	if data[0] == '[' && data[len(data)-1] == ']' {
		return Array
	}

	// String
	if data[0] == '"' && data[len(data)-1] == '"' {
		return String
	}

	// Number
	isInt := true
	isNum := false
numberLoop:
	for idx, c := range data {
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			// Do nothing
		case '-', '+':
			if idx != 0 {
				isInt = false
				isNum = false
				break numberLoop
			}
			// Do nothing
		case 'e':
			if idx != 1 {
				isInt = false
				isNum = false
				break numberLoop
			}
			// Do nothing
		case '.':
			isInt = false
			isNum = true
		default:
			isInt = false
			isNum = false
			break numberLoop
		}
	}
	if isNum {
		if IsInteger(data) {
			isInt = true
			isNum = false
		} else {
			return Number
		}
	}
	if isInt {
		return Integer
	}

	// Null
	if bytes.Equal(data, nullLiteral) {
		return Null
	}

	// Boolean
	if bytes.Equal(data, trueLiteral) || bytes.Equal(data, falseLiteral) {
		return Boolean
	}

	// Assume it's a string
	// TODO: Ensure this is a valid assumption
	return String
}
