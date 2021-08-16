package jsonschema

import "strings"

// Validate will return on the first encounter of something invalid
func (s *Schema) Validate(jsonDoc []byte) (bool, error) {
	// It's valid to have a text string with quotes as document, but the Validate func
	// expects non-quoted strings and the rest of the validators handles this automatically.
	// So we'll clean up any docs starting and ending with quotes.
	var err error
	typ := DetectJSONType(jsonDoc)

	if typ == String {
		jsonDoc = jsonDoc[1 : len(jsonDoc)-1]
	}

	// In Draft 4 the value 1.0 can NOT be an integer all other drafts allows this
	if s.IsDraft4() && typ == Integer && strings.Contains(string(jsonDoc), ".") {
		typ = Number
	}

	err = Validate(jsonDoc, typ, s)
	if err != nil {
		return false, err
	}
	return true, nil
}
