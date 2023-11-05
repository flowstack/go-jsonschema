package jsonschema

import (
	"testing"
)

func TestAddSchema(t *testing.T) {
	var testSchema = `{"$id":"http://example.com/schemas/thing","properties":{"id":{"type":"number"},"item":{"$ref":"http://example.com/schemas/item"}}}`
	var testRefSchemaItem = `{"$id":"http://example.com/schemas/item","properties":{"id":{"type":"number"},"label":{"type":"string"},"subitem1":{"$ref":"http://example.com/schemas/subitem"},"subitem2":{"$ref":"http://example.com/schemas/subitem"}}}`
	var testRefSchemaSubitem = `{"$id":"http://example.com/schemas/subitem","properties":{"id":{"type":"number"},"label":{"type":"string"}}}`
	var testDoc = `{"id":123,"item":{"id":321,"label":"item","subitem1":{"id":789,"label":"subitem1"},"subitem2":{"id":987,"label":"subitem2"}}}`
	var testDocInvalid = `{"id":123,"item":{"id":321,"label":"item","subitem1":{"id":789,"label":"subitem1"},"subitem2":{"id":"987","label":"subitem2"}}}`
	var expectedError = `value "987" is of type string, but should be of type: number at @.item.subitem2.id`

	schema, err := NewFromString(testSchema)
	if err != nil {
		t.Fatal(err)
	}

	// Add the last / deepest schema first to test the logic works correctly
	err = schema.AddSchemaString(testRefSchemaSubitem)
	if err != nil {
		t.Fatal(err)
	}

	err = schema.AddSchemaString(testRefSchemaItem)
	if err != nil {
		t.Fatal(err)
	}

	err = schema.DeRef()
	if err != nil {
		t.Fatal(err)
	}

	valid, err := schema.Validate([]byte(testDoc))
	if err != nil {
		t.Fatalf(`expected error to be empty, got: %s`, err.Error())
	} else if !valid {
		t.Fatal(`expected document to be valid`)
	}

	valid, err = schema.Validate([]byte(testDocInvalid))
	if err == nil || err.Error() != expectedError {
		t.Fatalf(`unexpected error %v, expected: %s`, err, expectedError)
	} else if valid {
		t.Fatal(`expected document to be invalid`)
	}
}
