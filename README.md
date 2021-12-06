[![Go Reference](https://pkg.go.dev/badge/github.com/flowstack/go-jsonschema.svg)](https://pkg.go.dev/github.com/flowstack/go-jsonschema)

# go-jsonschema [WIP]
Go JSON Schema parser and validator

Although this is a work in progress, it already passes all mandatory tests and most optional tests in the test suites for Draft 4, Draft 6 and Draft 7.

Errors are not very informative.
E.g. line number, keys, values, etc. aren't reported back.  
The main focus has been on speed and correctness, but error reporting should get better over time.

## Usage
```go
import "github.com/flowstack/go-jsonschema"

func main() {
    schema := `{"properties": {"id": {"type": "string"}}}`

    // Validate a JSON Schema
    _, err := jsonschema.Validate(schema)
    if err != nil {
        log.Fatal(err)
    }

    // Create a validator
    validator, err := jsonschema.NewFromString(schema)
    // Or: validator, err := jsonschema.New([]byte(schema))
    if err != nil {
        log.Fatal(err)
    }

    // Validate a JSON document against the schema
    json := `{"id": "123abc"}`
    _, err = validator.Validate([]byte(json))
    if err != nil {
        log.Fatal(err)
    }
}
```


## Contributions
Contributions are very welcome! This project is young and could use more eyes and brains to make everything better.  
So please fork, code and make pull requests.  
At least the existing tests should pass, but you're welcome to change those too, as long as the JSON Schema test suite is run and passes.

Currently most test for Draft 2019-09 and Draft 2020-12 passes, but there is more code to be done, before those 2 will be fully functional.

The JSON Schema parser is fairly slow and could probably be made faster easily.


## Motivation for creating yet another JSON Schema parser / validator
The very nice [gojsonschema](http://github.com/xeipuuv/gojsonschema) was missing some features and we needed some internal functionality, that was hard to build on top of [gojsonschema](http://github.com/xeipuuv/gojsonschema).

Furthermore [gojsonschema](http://github.com/xeipuuv/gojsonschema) uses Go's JSON parser, which makes it relatively slow, when parsing JSON documents for validation.  
This module uses the excellent [jsonparser](https://github.com/buger/jsonparser), which is waaaay faster than Go's builtin parser.

