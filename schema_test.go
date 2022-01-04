package jsonschema

import (
	"encoding/json"
	"errors"
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

func TestParserKeepSorting(t *testing.T) {
	var testSchema = `{"$id":"bla","const":null,"properties":{"yadda":{"type":["string","null"]},"bla":{"enum":["abc",123,1.23,null,false]}}}`

	schema, err := NewFromString(testSchema)
	if err != nil {
		t.Fatal(err)
	}

	parsedSchema := schema.String()

	if testSchema != parsedSchema {
		t.Fatalf("expected schemas to be equal, but got:\nexpected:\n%s\nactual:\n%s \n", testSchema, parsedSchema)
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

func TestUnknowns(t *testing.T) {
	var testSchema = `{"someField":"someName","stringField":{"type":"string"}}`

	schema, err := NewFromString(testSchema)
	if err != nil {
		t.Fatal(err)
	}

	someField, err := schema.GetUnknown("someField")
	if err != nil {
		t.Fatal(err)
	}
	if someField == nil {
		t.Fatal(errors.New("unknown property someField is nil"))
	}
	if someField.String == nil {
		t.Fatal(errors.New("unknown property someField is not a string"))
	}
	if *someField.String != "someName" {
		t.Fatal(errors.New(`unknown property someField is not "someName"`))
	}

	stringField, err := schema.GetUnknown("stringField")
	if err != nil {
		t.Fatal(err)
	}
	if stringField == nil {
		t.Fatal(errors.New("unknown property stringField is nil"))
	}
	if stringField.Object == nil {
		t.Fatal(errors.New("unknown property stringField is not an object"))
	}
	if typ, ok := (*stringField.Object)["type"]; !ok || *typ.String != "string" {
		t.Fatal(errors.New(`unknown property stringField type is not "string"`))
	}

	newSchema, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}

	if testSchema != string(newSchema) {
		t.Fatalf("expected schemas to be equal, but got:\nexpected:\n%s\nactual:\n%s \n", testSchema, string(newSchema))
	}
}
