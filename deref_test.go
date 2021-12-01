package jsonschema

import (
	_ "embed"
	"testing"
)

var deRefTests = []struct {
	schema         string
	expectedSchema string
}{
	{
		schema:         `{"$id":"http://example.com/schema-refs-absolute-uris-defs1.json","properties":{"foo":{"$id":"http://example.com/schema-refs-absolute-uris-defs2.json","definitions":{"inner":{"properties":{"bar":{"type":"string"}}}},"allOf":[{"$ref":"#/definitions/inner"}]}},"allOf":[{"$ref":"schema-refs-absolute-uris-defs2.json"}]}`,
		expectedSchema: `{"$id":"http://example.com/schema-refs-absolute-uris-defs1.json","allOf":[{"$id":"http://example.com/schema-refs-absolute-uris-defs2.json","definitions":{"inner":{"properties":{"bar":{"type":"string"}}}},"allOf":[{"properties":{"bar":{"type":"string"}}}]}],"properties":{"foo":{"$id":"http://example.com/schema-refs-absolute-uris-defs2.json","definitions":{"inner":{"properties":{"bar":{"type":"string"}}}},"allOf":[{"properties":{"bar":{"type":"string"}}}]}}}`,
	},
	{
		schema:         `{"$schema":"http://json-schema.org/draft-04/schema#","properties":{"foo":{"$ref":"#"}},"additionalProperties":false}`,
		expectedSchema: `{"$schema":"http://json-schema.org/draft-04/schema#","properties":{"foo":{"$schema":"http://json-schema.org/draft-04/schema#","properties":{"foo":{"$schema":"http://json-schema.org/draft-04/schema#","properties":{"foo":{"$schema":"http://json-schema.org/draft-04/schema#","properties":{"foo":{"$ref":"#"}},"additionalProperties":false}},"additionalProperties":false}},"additionalProperties":false}},"additionalProperties":false}`,
	},
}

func TestDeRef(t *testing.T) {
	for _, tt := range deRefTests {
		s, err := New([]byte(tt.schema))
		if err != nil {
			t.Fatal(err)
		}

		err = s.DeRef()
		if err != nil {
			t.Fatal(err)
		}

		if s.String() != tt.expectedSchema {
			t.Fatalf(
				"expected de-ref'ed schema to match:\n%s\ngot:\n%s\n",
				tt.expectedSchema,
				s.String(),
			)
		}
	}
}
