package jsonschema

import (
	"encoding/json"
	"testing"
)

func TestBooleanSchema(t *testing.T) {
	var testSchema = `true`

	schema, err := NewFromString(testSchema)
	if err != nil {
		t.Fatal(err)
	}

	newSchema, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}

	if testSchema != string(newSchema) {
		t.Fatalf("expected schemas to be equal, but got:\nexpected:\n%s\nactual:\n%s \n", testSchema, string(newSchema))
	}
}

func TestParserSimple(t *testing.T) {
	var testSchema = `{"$id":"bla","const":null,"properties":{"bla":{"type":["string","null"]},"yadda":{"enum":["abc",123,1.23,null,false]}}}`

	schema, err := NewFromString(testSchema)
	if err != nil {
		t.Fatal(err)
	}

	newSchema, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}

	if testSchema != string(newSchema) {
		t.Fatalf("expected schemas to be equal, but got:\nexpected:\n%s\nactual:\n%s \n", testSchema, string(newSchema))
	}
}

func TestItems(t *testing.T) {
	var testSchema = `{"properties":{"itemField":{"type":"array","items":{"type":"string"}},"itemsField":{"type":"array","items":[{"type":"string"}]}}}`

	schema, err := NewFromString(testSchema)
	if err != nil {
		t.Fatal(err)
	}

	newSchema, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}

	if testSchema != string(newSchema) {
		t.Fatalf("expected schemas to be equal, but got:\nexpected:\n%s\nactual:\n%s \n", testSchema, string(newSchema))
	}
}
