package jsonschema

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/buger/jsonparser"
)

// uniqueValidator is a bit awkward, but is built this way,
// to make the validation fast and relatively simple.
type uniqueValidator struct {
	// For arrays the key is: [index]:[value]:type
	// For objects the key is: [key]:[value]:jtype
	// All others have: [value]:[type]
	cache map[string]struct{}
}

// This will output object and arrays as a sorted json value
func sortObject(data []byte) []byte {
	vals := map[string][]byte{}
	err := jsonparser.ObjectEach(data, func(key, value []byte, dataType jsonparser.ValueType, offset int) error {
		if dataType == jsonparser.String {
			vals[string(key)] = []byte(fmt.Sprintf(`"%s"`, string(value)))
		} else {
			vals[string(key)] = value
		}
		return nil
	})

	if err != nil {
		return data
	}

	keys := []string{}
	for k := range vals {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var sortedKVs []string
	for _, k := range keys {
		sortedKVs = append(sortedKVs, fmt.Sprintf(`"%s":%s`, k, vals[k]))
	}
	sortedData := fmt.Sprintf("{%s}", strings.Join(sortedKVs, ","))

	return []byte(sortedData)
}

func newUniqueValidator() *uniqueValidator {
	return &uniqueValidator{cache: map[string]struct{}{}}
}

func (u *uniqueValidator) Exists(data []byte, vt jsonparser.ValueType) bool {
	var key string

	switch vt {
	case jsonparser.Number:
		key = strings.TrimRight(strings.TrimRight(string(data), "0"), ".")
	case jsonparser.Array:
		// Objects must have sorted keys in order for comparison to work (objects can be in arrays)
		arrVal := newUniqueValidator()
		var errs error
		_, err := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if arrVal.Exists(value, dataType) {
				errs = addError(errors.New("already exists"), errs)
				return
			}
		})
		if err != nil || errs != nil {
			// errs = addError(err, errs)
			return true
		}
		key = string(data)

	case jsonparser.Object:
		// Objects must have sorted keys in order for comparison to work (objects can be in arrays)
		objVal := newUniqueValidator()
		err := jsonparser.ObjectEach(data, func(key, value []byte, dataType jsonparser.ValueType, offset int) error {
			if objVal.Exists(value, dataType) {
				return errors.New("already exists")
			}
			return nil
		})
		if err != nil {
			return true
		}
		key = string(sortObject(data))

	default:
		key = string(data)
	}

	if _, ok := u.cache[key+vt.String()]; ok {
		return true
	}

	u.cache[key+vt.String()] = struct{}{}

	return false
}
