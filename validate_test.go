package jsonschema

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/xeipuuv/gojsonschema"
	"gitlab.com/flowstack/jsonschema/testtools"
)

type schemaTest struct {
	Description string          `json:"description"` // valid definition schema
	Data        json.RawMessage `json:"data"`
	Valid       bool            `json:"valid"`
}

type schemaTests struct {
	Description string          `json:"description"` // validate definition against metaschema
	Schema      json.RawMessage `json:"schema"`      // {"$ref": "http://json-schema.org/draft-07/schema#"}
	Tests       []schemaTest    `json:"tests"`
}

var testSchemaVersions = []string{"draft4", "draft6", "draft7", "draft2019-09", "draft2020-12"}

// var testSchemaVersions = []string{"draft4", "draft6", "draft7", "draft2019-09"}

// This is basically to get an idea of how much work is left to support draft2019-09.
// Another consideration is how to de-ref $defs, if at all - they're to be treated as self-contained schemas.
// TODO: Make the tests pass
var ignoreDraft2019_09TestFiles = map[string]struct{}{
	"anchor.json":                  {}, // not implemented
	"content.json":                 {}, // not implemented - optional for all earlier draft standards
	"defs.json":                    {}, // not implemented
	"dependentRequired.json":       {}, // not implemented, but is basically the dependencies code
	"dependentSchemas.json":        {}, // not implemented, but is basically the dependencies code
	"format.json":                  {}, // more checks than for the earlier draft standards
	"id.json":                      {}, // seems to be things that should be checked anyway
	"infinite-loop-detection.json": {}, // $defs not implemented
	"items.json":                   {}, // $defs not implemented
	"maxContains.json":             {}, // not implemented
	"minContains.json":             {}, // not implemented
	"recursiveRef.json":            {}, // not implemented
	"ref.json":                     {}, // $defs not implemented
	"refRemote.json":               {}, // $defs not implemented
	"unevaluatedItems.json":        {}, // not implemented
	"unevaluatedProperties.json":   {}, // not implemented
	"unknownKeyword.json":          {}, // not implemented
	"refOfUnknownKeyword.json":     {}, // not implemented (optional)
}

// Same as for draft2019-09.
var ignoreDraft2020_12TestFiles = map[string]struct{}{
	"anchor.json":                  {}, // not implemented
	"content.json":                 {}, // not implemented - optional for all earlier draft standards
	"defs.json":                    {}, // not implemented
	"dependentRequired.json":       {}, // not implemented, but is basically the dependencies code
	"dependentSchemas.json":        {}, // not implemented, but is basically the dependencies code
	"dynamicRef.json":              {}, // not implemented
	"format.json":                  {}, // more checks than for the earlier draft standards
	"id.json":                      {}, // seems to be things that should be checked anyway
	"infinite-loop-detection.json": {}, // $defs not implemented
	"items.json":                   {}, // $defs not implemented
	"maxContains.json":             {}, // not implemented
	"minContains.json":             {}, // not implemented
	"prefixItems.json":             {}, // not implemented
	"ref.json":                     {}, // $defs not implemented
	"refRemote.json":               {}, // $defs not implemented
	"unevaluatedItems.json":        {}, // not implemented
	"unevaluatedProperties.json":   {}, // not implemented
	"uniqueItems.json":             {}, // prefixItems not implemented
	"unknownKeyword.json":          {}, // not implemented
	"refOfUnknownKeyword.json":     {}, // not implemented (optional)
}

var testDataPath = "testdata"

func TestMain(m *testing.M) {
	// Start a server for the remote test schema
	remoteSchemasPath := path.Join(testDataPath, "remotes")
	go func() {
		err := http.ListenAndServe(":1234", http.FileServer(http.Dir(remoteSchemasPath)))
		if err != nil {
			panic(err)
		}
	}()

	os.Exit(m.Run())
}

func TestValidateEmptyDocWithSchema(t *testing.T) {
	schema, err := NewFromString("{}")
	if err != nil {
		t.Fatal(err)
	}

	_, err = schema.Validate([]byte(""))
	if err == nil {
		t.Fatal(`expected empty err, expected: empty document does not validate against the schema`)
	} else if err.Error() != `empty document is not valid against any other schemas than "false"` {
		t.Fatalf(`expected error to be:\nempty document is not valid against any other schemas than "false"\n, got:\n%s`, err.Error())
	}
}

// TODO: verify that this is the wanted outcome
func TestValidateEmptyDocWithFalseSchema(t *testing.T) {
	schema, err := NewFromString("false")
	if err != nil {
		t.Fatal(err)
	}

	valid, err := schema.Validate([]byte(""))
	if err != nil {
		t.Fatalf("expected error to be empty, got:\n%s", err.Error())
	} else if !valid {
		t.Fatal(`expected document to be valid`)
	}
}

// TODO: verify that this is the wanted outcome
func TestValidateValueWithTrueSchema(t *testing.T) {
	schema, err := NewFromString("true")
	if err != nil {
		t.Fatal(err)
	}

	valid, err := schema.Validate([]byte("1"))
	if err != nil {
		t.Fatalf(`expected error to be empty, got: %s`, err.Error())
	} else if !valid {
		t.Fatal(`expected document to be valid`)
	}
}

func TestValidateValue(t *testing.T) {
	schema, err := NewFromString("{}")
	if err != nil {
		t.Fatal(err)
	}

	valid, err := schema.Validate([]byte("1"))
	if err != nil {
		t.Fatalf(`expected error to be empty, got: "%s"`, err.Error())
	} else if !valid {
		t.Fatal(`expected validation to be true, got false`)
	}
}

func TestValidateSchema(t *testing.T) {
	var testSchema = `{"$id":"bla","const":null,"properties":{"bla":{"type":["string","null"]},"yadda":{"enum":["abc",123,1.23,null,false]}}}`

	valid, err := Validate([]byte(testSchema))
	if err != nil {
		t.Fatalf(`expected error to be empty, got: %s`, err.Error())
	} else if !valid {
		t.Fatal(`expected document to be valid`)
	}
}

// TestParse runs through all of the test suite's tests (including optional)
func TestParseAndValidate(t *testing.T) {
	for _, testSchemaVersions := range testSchemaVersions {
		dirPath := path.Join("./", testDataPath, testSchemaVersions)

		parseAndValidateHelper(t, dirPath, testSchemaVersions)
	}
}

// These are long running benchmarks, so they won't be included by default
// func BenchmarkParse(b *testing.B) {
// 	for _, testSchemaVersions := range testSchemaVersions {
// 		dirPath := path.Join("./", testDataPath, testSchemaVersions)

// 		parseBenchmarkHelper(b, dirPath, testSchemaVersions)
// 	}
// }

// func BenchmarkValidate(b *testing.B) {
// 	for _, testSchemaVersions := range testSchemaVersions {
// 		dirPath := path.Join("./", testDataPath, testSchemaVersions)

// 		validateBenchmarkHelper(b, dirPath, testSchemaVersions)
// 	}
// }

// Helper for recursing the testdata dirs
func parseAndValidateHelper(t *testing.T, dirPath, schemaVersion string) {
	t.Helper()

	files, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			// Only go through files for now - optionals could / should be included though
			parseAndValidateHelper(t, path.Join(dirPath, file.Name()), schemaVersion)
			continue
		}

		// Temporarily disable some draft2019-09 tests.
		if schemaVersion == "draft2019-09" {
			if _, ok := ignoreDraft2019_09TestFiles[file.Name()]; ok {
				continue
			}
		}

		// Temporarily disable some draft2020-12 tests.
		if schemaVersion == "draft2020-12" {
			if _, ok := ignoreDraft2020_12TestFiles[file.Name()]; ok {
				continue
			}
		}

		// TODO: Make the failing cases pass...
		if path.Base(dirPath) == "format" {
			// The following formats ARE validated, but the validations fails in some rare edge cases,
			// which in many cases, can be mitigated by formatting the values.
			// E.g. 087.1.2.3 is invalid due to leading 0, but will be formatted correctly by net.ParseIP.
			// Formatting of URI/IRI(-reference) is done with Go's buildin url methods.
			if file.Name() == "idn-hostname.json" {
				continue
			}
			if file.Name() == "ipv4.json" || file.Name() == "ipv6.json" {
				continue
			}
			if file.Name() == "iri.json" || file.Name() == "iri-reference.json" {
				continue
			}
			if file.Name() == "uri.json" || file.Name() == "uri-reference.json" {
				continue
			}
			if file.Name() == "relative-json-pointer.json" {
				continue
			}
		}

		if path.Base(dirPath) == "optional" {
			// EcmaScript regex is a different (slower) beast than Go's regex2 engine.
			// A couple of manipulations are done to the regexes, before they're run,
			// in order to support some of EcmaScript regex, but not everything is working yet.
			if file.Name() == "ecmascript-regex.json" {
				continue
			}
			// Content validation (e.g. is this value, valid JSON, JPEG, etc.) could (should?) be done.
			if file.Name() == "content.json" {
				continue
			}
		}

		filePath := path.Join(dirPath, file.Name())

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}

		// Extract the tests into structs
		schemaTests := []schemaTests{}
		err = json.Unmarshal(data, &schemaTests)
		if err != nil {
			t.Fatalf("error while parsing: %s\nerror:%s", filePath, err.Error())
		}

		// log.Println("======================", filePath, "======================")
		for i, schemaTest := range schemaTests {
			// log.Printf("____________________________ #%d: %s ____________________________\n", i, schemaTest.Description)

			// Parse the schema
			schema, err := New(schemaTest.Schema)
			if err != nil {
				t.Fatalf("error while parsing: %s, test #%d\nerror: %s", filePath, i+1, err.Error())
			}

			// log.Println("CACHE")
			// for k, s := range *schema.cache {
			// 	log.Println(k, s.String())
			// }

			// Verify that we actually have all the information
			actualSchema, err := json.Marshal(schema)
			if err != nil {
				t.Fatalf("error while parsing: %s, test #%d\nerror: %s", filePath, i+1, err.Error())
			}

			expectedSchema, err := testtools.SortAndCompactJSON(schemaTest.Schema)
			if err != nil {
				t.Fatal(err)
			}
			actualSchema, err = testtools.SortAndCompactJSON(actualSchema)
			if err != nil {
				t.Fatal(err)
			}

			if string(expectedSchema) != string(actualSchema) {
				// Allow match failure for testdata/draft*/unknownKeyword.json
				// The schema has errors on purpose, so the parser SHOULD get something different
				if file.Name() != "unknownKeyword.json" {
					t.Fatalf(
						"%s, test #%d\nexpected schemas to be equal, got:\nexpected:\n%s\nactual:\n%s \n",
						filePath, i+1, string(expectedSchema), string(actualSchema))
				}
			}

			// Force set the $schema value, to ensure the parser / validators knows which version is expected
			var schemaStr string
			switch schemaVersion {
			case "draft4":
				schemaStr = "http://json-schema.org/draft-04/schema#"
			case "draft6":
				schemaStr = "http://json-schema.org/draft-06/schema#"
			case "draft7":
				schemaStr = "http://json-schema.org/draft-07/schema#"
			}
			schema.Schema = &schemaStr

			// Go through the tests and check that the validations matches
			for n, test := range schemaTest.Tests {
				// log.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~ #%d.%d: %s ~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n", i, n, test.Description)
				actual, err := schema.Validate(test.Data)

				if actual != test.Valid {
					errStr := fmt.Sprintf("expected validation to be %t, got: %t\n", test.Valid, actual)
					if err != nil {
						errStr += err.Error() + "\n\n"
					}

					if errStr != "" {
						errStr = "errors encountered:\n" + errStr

						t.Fatalf(`%s,
Test #%d.%d: "%s"

%sSchema:
%s

Test document:
%s`,
							filePath, i+1, n+1, test.Description, errStr, string(schemaTest.Schema), string(test.Data))
					}
				}
			}

			// Go through the tests again, but this time with de-ref'ed $refs
			err = schema.DeRef()
			if err != nil {
				t.Fatal(err)
			}

			for n, test := range schemaTest.Tests {
				// log.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~ #%d.%d: %s ~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n", i, n, test.Description)
				actual, err := schema.Validate(test.Data)

				if actual != test.Valid {
					errStr := fmt.Sprintf("expected validation to be %t, got: %t\n", test.Valid, actual)
					if err != nil {
						errStr += err.Error() + "\n\n"
					}

					if errStr != "" {
						errStr = "errors encountered:\n" + errStr

						t.Fatalf(`%s,
Test #%d.%d: "%s"

%sSchema:
%s

Test document:
%s`,
							filePath, i+1, n+1, test.Description, errStr, string(schemaTest.Schema), string(test.Data))
					}
				}
			}

		}
	}
}

// Helper for recursing the testdata dirs
func parseBenchmarkHelper(b *testing.B, dirPath, schemaVersion string) {
	b.Helper()

	files, err := os.ReadDir(dirPath)
	if err != nil {
		b.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			// Only go through files for now - optionals could / should be included though
			parseBenchmarkHelper(b, path.Join(dirPath, file.Name()), schemaVersion)
			continue
		}

		filePath := path.Join(dirPath, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			b.Fatal(err)
		}

		// Extract the tests into structs
		schemaTests := []schemaTests{}
		err = json.Unmarshal(data, &schemaTests)
		if err != nil {
			b.Fatalf("error while parsing: %s\nerror:%s", filePath, err.Error())
		}

		for _, schemaTest := range schemaTests {
			// Parse the schema
			b.Run(filePath, func(b *testing.B) {
				var err error
				for i := 0; i < b.N; i++ {
					var schema *Schema
					schema, err = New(schemaTest.Schema)
					if err != nil {
						b.Fatalf("error while parsing: %s, test #%d\nerror: %s", filePath, i+1, err.Error())
					}
					_ = schema
				}
			})

			b.Run(filePath+"Native", func(b *testing.B) {
				var err error
				for i := 0; i < b.N; i++ {
					var schema interface{}
					err = json.Unmarshal(schemaTest.Schema, &schema)
					if err != nil {
						b.Fatalf("error while parsing: %s, test #%d\nerror: %s", filePath, i+1, err.Error())
					}
					_ = schema
				}
			})

			b.Run(filePath+"GoJSON", func(b *testing.B) {
				var err error
				for i := 0; i < b.N; i++ {
					sl := gojsonschema.NewSchemaLoader()
					loader1 := gojsonschema.NewBytesLoader(schemaTest.Schema)
					err = sl.AddSchema("http://some_host.com/string.json", loader1)
					if err != nil {
						b.Fatalf("error while parsing: %s, test #%d\nerror: %s", filePath, i+1, err.Error())
					}
					_, _ = sl, loader1
				}
			})

		}
	}
}

// Helper for recursing the testdata dirs
func validateBenchmarkHelper(b *testing.B, dirPath, schemaVersion string) {
	b.Helper()

	files, err := os.ReadDir(dirPath)
	if err != nil {
		b.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			// Only go through files for now - optionals could / should be included though
			parseBenchmarkHelper(b, path.Join(dirPath, file.Name()), schemaVersion)
			continue
		}

		filePath := path.Join(dirPath, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			b.Fatal(err)
		}

		// Extract the tests into structs
		schemaTests := []schemaTests{}
		err = json.Unmarshal(data, &schemaTests)
		if err != nil {
			b.Fatalf("error while parsing: %s\nerror:%s", filePath, err.Error())
		}

		for i, schemaTest := range schemaTests {
			// Parse the schema
			var jsonSchema *Schema
			jsonSchema, err = New(schemaTest.Schema)
			if err != nil {
				b.Fatalf("error while parsing: %s, test #%d\nerror: %s", filePath, i+1, err.Error())
			}

			schemaLoader := gojsonschema.NewBytesLoader(schemaTest.Schema)
			gojsonSchema, err := gojsonschema.NewSchema(schemaLoader)
			if err != nil {
				b.Fatalf("error while parsing: %s, test #%d\nerror: %s", filePath, i+1, err.Error())
			}

			for n, test := range schemaTest.Tests {
				var res1 bool
				var err1 error
				b.Run(fmt.Sprintf("LOCAL: %s #%d.%d", filePath, i, n), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						res1, err1 = jsonSchema.Validate(test.Data)
					}
				})
				_, _ = res1, err1

				var err2 error
				var res2 *gojsonschema.Result
				b.Run(fmt.Sprintf("GOJSON: %s #%d.%d", filePath, i, n), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						docLoader := gojsonschema.NewBytesLoader(test.Data)
						res2, err2 = gojsonSchema.Validate(docLoader)
					}
				})
				_, _ = res2, err2

			}

		}
	}

}
