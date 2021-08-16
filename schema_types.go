package jsonschema

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/buger/jsonparser"
)

type refs []*Ref

func (r *refs) String() {
	if r != nil {
		for _, r := range *r {
			if r != nil && r.String != nil {
				log.Printf("ref %s\n", *r.String)
			} else {
				log.Printf("empty ref found\n")
			}
		}
	}
}

type pointers map[string]*Schema

func (p *pointers) String() {
	if p != nil {
		for k, s := range *p {
			if s != nil {
				log.Printf("pointer for %s\n%s\n", k, s.String())
			} else {
				log.Printf("pointer for %s is empty\n", k)
			}
		}
	}
}

type ValueType uint8
type SchemaProp uint8

// These are the same as jsonparser.ValueType - except Integer, which jsonpparser does not have
const (
	NotExist ValueType = iota
	String
	Number
	Object
	Array
	Boolean
	Null
	Unknown
	Integer
)

func (v ValueType) String() string {
	switch v {
	case NotExist:
		return "non-existent"
	case String:
		return "string"
	case Number:
		return "number"
	case Integer:
		return "integer"
	case Object:
		return "object"
	case Array:
		return "array"
	case Boolean:
		return "boolean"
	case Null:
		return "null"
	case Unknown:
		fallthrough
	default:
		return "unknown"
	}
}

func (v ValueType) ParserValueType() jsonparser.ValueType {
	switch v {
	case NotExist:
		return jsonparser.NotExist
	case String:
		return jsonparser.String
	case Number:
		return jsonparser.Number
	case Integer:
		return jsonparser.Number
	case Object:
		return jsonparser.Object
	case Array:
		return jsonparser.Array
	case Boolean:
		return jsonparser.Boolean
	case Null:
		return jsonparser.Null
	case Unknown:
		fallthrough
	default:
		return jsonparser.Unknown
	}
}

// TODO: Update with 20xx-xx props
const (
	PropSchema SchemaProp = iota
	PropID
	PropIDDraft04
	PropRef
	PropComment
	PropTitle
	PropDescription
	PropType
	PropEnum
	PropDefault
	PropConst
	PropExamples
	PropReadOnly
	PropWriteOnly
	PropDefinitions
	PropIf
	PropThen
	PropElse
	PropAllOf
	PropAnyOf
	PropOneOf
	PropNot
	PropContentEncoding
	PropContentMediaType
	PropProperties
	PropRequired
	PropMaxProperties
	PropMinProperties
	PropDependencies
	PropPatternProperties
	PropAdditionalProperties
	PropPropertyNames
	PropItems
	PropMaxItems
	PropMinItems
	PropUniqueItems
	PropAdditionalItems
	PropContains
	PropMaxLength
	PropMinLength
	PropFormat
	PropPattern
	PropMultipleOf
	PropMaximum
	PropExclusiveMaximum
	PropMinimum
	PropExclusiveMinimum
)

// TODO: Update with 20xx-xx props
var propNames = [][]string{
	PropSchema:               {"$schema"},
	PropID:                   {"$id"},
	PropIDDraft04:            {"id"},
	PropRef:                  {"$ref"},
	PropComment:              {"$comment"},
	PropTitle:                {"title"},
	PropDescription:          {"description"},
	PropType:                 {"type"},
	PropEnum:                 {"enum"},
	PropDefault:              {"default"},
	PropConst:                {"const"},
	PropExamples:             {"examples"},
	PropReadOnly:             {"readOnly"},
	PropWriteOnly:            {"writeOnly"},
	PropDefinitions:          {"definitions"},
	PropIf:                   {"if"},
	PropThen:                 {"then"},
	PropElse:                 {"else"},
	PropAllOf:                {"allOf"},
	PropAnyOf:                {"anyOf"},
	PropOneOf:                {"oneOf"},
	PropNot:                  {"not"},
	PropContentEncoding:      {"contentEncoding"},
	PropContentMediaType:     {"contentMediaType"},
	PropProperties:           {"properties"},
	PropRequired:             {"required"},
	PropMaxProperties:        {"maxProperties"},
	PropMinProperties:        {"minProperties"},
	PropDependencies:         {"dependencies"},
	PropPatternProperties:    {"patternProperties"},
	PropAdditionalProperties: {"additionalProperties"},
	PropPropertyNames:        {"propertyNames"},
	PropItems:                {"items"},
	PropMaxItems:             {"maxItems"},
	PropMinItems:             {"minItems"},
	PropUniqueItems:          {"uniqueItems"},
	PropAdditionalItems:      {"additionalItems"},
	PropContains:             {"contains"},
	PropMaxLength:            {"maxLength"},
	PropMinLength:            {"minLength"},
	PropFormat:               {"format"},
	PropPattern:              {"pattern"},
	PropMultipleOf:           {"multipleOf"},
	PropMaximum:              {"maximum"},
	PropExclusiveMaximum:     {"exclusiveMaximum"},
	PropMinimum:              {"minimum"},
	PropExclusiveMinimum:     {"exclusiveMinimum"},
}

func NewStringPtr(b []byte) *string {
	unescaped, err := jsonparser.Unescape(b, nil)
	if err != nil {
		return nil
	}
	tmpStr := string(unescaped)
	return &tmpStr
}

// func (e Ref) MarshalJSON() ([]byte, error) {
// 	b, err := json.Marshal(tmpEnum(e))
// 	return b,err
// }

type Enum []*Value
type tmpEnum Enum // To ensure MarshalJSON doesn't go haywire

func (e Enum) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(tmpEnum(e))
	return b, err
}

func NewEnum(jsonVal []byte, vt jsonparser.ValueType) (*Enum, error) {
	var errs error

	if vt == jsonparser.Array {
		enum := Enum{}

		jsonparser.ArrayEach(jsonVal, func(value []byte, dataType jsonparser.ValueType, offset int, parseErr error) {
			if parseErr != nil {
				errs = AddError(parseErr, errs)
				return
			}
			val, err := NewValue(value, dataType)
			if err != nil {
				errs = AddError(err, errs)
				return
			}
			enum = append(enum, val)
		})

		if errs != nil {
			return nil, errs
		}

		return &enum, nil
	}

	return nil, fmt.Errorf("expected enum to be an array, got: %s", vt.String())
}

type Dependencies map[string]*Dependency
type tmpDependencies Dependencies // To ensure MarshalJSON doesn't go haywire

func (d Dependencies) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(tmpDependencies(d))
	return b, err
}

func NewDependencies(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Dependencies, error) {
	if vt == jsonparser.Object {
		dependencies := Dependencies{}
		err := jsonparser.ObjectEach(jsonVal, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			var err error
			dependencies[string(key)], err = NewDependency(value, dataType, parentSchema)
			return err
		})

		if err != nil {
			return nil, err
		}

		return &dependencies, nil
	}

	return nil, fmt.Errorf("expected properties to be object, got: %s", vt.String())
}

type Properties map[string]*Schema
type tmpProperties Properties // To ensure MarshalJSON doesn't go haywire

func (p Properties) MarshalJSON() ([]byte, error) {
	// tmp := tmpProperties{}

	// for k, v := range p {
	// 	log.Println(k)
	// 	if v.Ref != nil && v.Ref.Schema != nil {
	// 		log.Println("Yep")
	// 		tmp[k] = v.Ref.Schema
	// 	} else {
	// 		tmp[k] = v
	// 	}
	// }

	// b, err := json.Marshal(tmp)

	b, err := json.Marshal(tmpProperties(p))
	return b, err
}

func NewProperties(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Properties, error) {
	if vt == jsonparser.Object {
		props := Properties{}
		err := jsonparser.ObjectEach(jsonVal, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			var err error
			props[string(key)], err = parentSchema.Parse(value)
			return err
		})

		if err != nil {
			return nil, err
		}

		return &props, nil
	}

	return nil, fmt.Errorf("expected properties to be object, got: %s", vt.String())
}

type Definitions map[string]*Schema
type tmpDefinitions Definitions // To ensure MarshalJSON doesn't go haywire

func (p Definitions) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(tmpDefinitions(p))
	return b, err
}

func NewDefinitions(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Properties, error) {
	if vt == jsonparser.Object {
		props := Properties{}
		err := jsonparser.ObjectEach(jsonVal, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			var err error
			props[string(key)], err = parentSchema.Parse(value)
			return err
		})

		if err != nil {
			return nil, err
		}

		return &props, nil
	}

	return nil, fmt.Errorf("expected properties to be object, got: %s", vt.String())
}

type Schemas []*Schema
type tmpSchemas Schemas // To ensure MarshalJSON doesn't go haywire

func (s Schemas) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(tmpSchemas(s))
	return b, err
}

func NewSubSchemas(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Schemas, error) {
	var errs error

	if vt == jsonparser.Array {
		schemas := Schemas{}

		jsonparser.ArrayEach(jsonVal, func(value []byte, dataType jsonparser.ValueType, offset int, parseErr error) {
			if parseErr != nil {
				errs = AddError(parseErr, errs)
				return
			}

			if dataType == jsonparser.Object || dataType == jsonparser.Boolean {
				schema, err := parentSchema.Parse(value)
				if err != nil {
					errs = AddError(err, errs)
					return
				}
				schemas = append(schemas, schema)

			} else {
				errs = AddError(fmt.Errorf("expected type to be object or boolean, got: %s", vt.String()), errs)
			}
		})

		if errs != nil {
			return nil, errs
		}

		return &schemas, nil
	}

	return nil, fmt.Errorf("expected enum to be an array, got: %s", vt.String())
}

type Strings []*string
type tmpStrings Strings // To ensure MarshalJSON doesn't go haywire

func (p Strings) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(tmpStrings(p))
	return b, err
}

func NewStrings(jsonVal []byte, vt jsonparser.ValueType) (*Strings, error) {
	var errs error

	if vt == jsonparser.Array {
		strings := Strings{}

		jsonparser.ArrayEach(jsonVal, func(value []byte, dataType jsonparser.ValueType, offset int, parseErr error) {
			if parseErr != nil {
				errs = AddError(parseErr, errs)
				return
			}

			if dataType == jsonparser.String {
				unescaped, err := jsonparser.Unescape(value, nil)
				if err != nil {
					errs = AddError(parseErr, errs)
					return
				}
				tmpStr := string(unescaped)
				strings = append(strings, &tmpStr)

			} else {
				errs = AddError(fmt.Errorf("expected type to be string, got: %s", vt.String()), errs)
			}
		})

		if errs != nil {
			return nil, errs
		}

		return &strings, nil
	}

	return nil, fmt.Errorf("expected enum to be an array, got: %s", vt.String())
}

type Values []*Value
type tmpValues Values // To ensure MarshalJSON doesn't go haywire

func (v Values) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(tmpValues(v))
	return b, err
}

func NewValues(jsonVal []byte, vt jsonparser.ValueType) (*Values, error) {
	var errs error

	if vt == jsonparser.Array {
		values := Values{}

		jsonparser.ArrayEach(jsonVal, func(value []byte, dataType jsonparser.ValueType, offset int, parseErr error) {
			if parseErr != nil {
				errs = AddError(parseErr, errs)
				return
			}

			val, err := NewValue(value, dataType)
			if err != nil {
				errs = AddError(err, errs)
				return
			}
			values = append(values, val)
		})

		if errs != nil {
			return nil, errs
		}

		return &values, nil
	}

	return nil, fmt.Errorf("expected enum to be an array, got: %s", vt.String())
}
