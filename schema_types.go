package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/buger/jsonparser"
)

type refs []*Ref

func (r *refs) String() string {
	str := ""
	if r != nil {
		for _, r := range *r {
			if r != nil && r.String != nil {
				str += fmt.Sprintf("ref %s\n", *r.String)
			} else {
				str += "empty ref found\n"
			}
		}
	}
	return str
}

type pointers map[string]*Schema

func (p *pointers) String() string {
	str := ""
	if p != nil {
		for k, s := range *p {
			if s != nil {
				str += fmt.Sprintf("pointer for %s\n%s\n", k, s.String())
			} else {
				str += fmt.Sprintf("pointer for %s is empty\n", k)
			}
		}
	}
	return str
}

type ValueType uint8
type SchemaProp uint8

// These are the same as jsonparser.ValueType - except Integer, which jsonparser does not have
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

// TODO: Update with 20xx-xx props
var nameToProp = map[string]SchemaProp{
	"$schema":              PropSchema,
	"$id":                  PropID,
	"id":                   PropIDDraft04,
	"$ref":                 PropRef,
	"$comment":             PropComment,
	"title":                PropTitle,
	"description":          PropDescription,
	"type":                 PropType,
	"enum":                 PropEnum,
	"default":              PropDefault,
	"const":                PropConst,
	"examples":             PropExamples,
	"readOnly":             PropReadOnly,
	"writeOnly":            PropWriteOnly,
	"definitions":          PropDefinitions,
	"if":                   PropIf,
	"then":                 PropThen,
	"else":                 PropElse,
	"allOf":                PropAllOf,
	"anyOf":                PropAnyOf,
	"oneOf":                PropOneOf,
	"not":                  PropNot,
	"contentEncoding":      PropContentEncoding,
	"contentMediaType":     PropContentMediaType,
	"properties":           PropProperties,
	"required":             PropRequired,
	"maxProperties":        PropMaxProperties,
	"minProperties":        PropMinProperties,
	"dependencies":         PropDependencies,
	"patternProperties":    PropPatternProperties,
	"additionalProperties": PropAdditionalProperties,
	"propertyNames":        PropPropertyNames,
	"items":                PropItems,
	"maxItems":             PropMaxItems,
	"minItems":             PropMinItems,
	"uniqueItems":          PropUniqueItems,
	"additionalItems":      PropAdditionalItems,
	"contains":             PropContains,
	"maxLength":            PropMaxLength,
	"minLength":            PropMinLength,
	"format":               PropFormat,
	"pattern":              PropPattern,
	"multipleOf":           PropMultipleOf,
	"maximum":              PropMaximum,
	"exclusiveMaximum":     PropExclusiveMaximum,
	"minimum":              PropMinimum,
	"exclusiveMinimum":     PropExclusiveMinimum,
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
				errs = addError(parseErr, errs)
				return
			}
			val, err := NewValue(value, dataType)
			if err != nil {
				errs = addError(err, errs)
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

// Properties are build like this, instead of map[string]*Schema,
// to make it possible to Marshal with original sort order of fields
type NamedProperty struct {
	Name     string
	Property *Schema
}
type Properties []*NamedProperty

// type tmpProperties Properties // To ensure MarshalJSON doesn't go haywire

func (p Properties) GetProperty(name string) (*NamedProperty, bool) {
	for _, prop := range p {
		if prop.Name == name {
			return prop, true
		}
	}
	return nil, false
}

func (p Properties) MarshalJSON() ([]byte, error) {
	propsData := []byte("{")
	buf := bytes.NewBuffer(propsData)
	for i, prop := range p {
		fieldVal, err := json.Marshal(prop.Property)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		escapedFieldName := strings.ReplaceAll(prop.Name, `\`, `\\`)
		escapedFieldName = strings.ReplaceAll(escapedFieldName, `"`, `\"`)
		escapedFieldName = strings.ReplaceAll(escapedFieldName, "\n", `\n`)
		escapedFieldName = strings.ReplaceAll(escapedFieldName, "\r", `\r`)
		escapedFieldName = strings.ReplaceAll(escapedFieldName, "\t", `\t`)
		escapedFieldName = strings.ReplaceAll(escapedFieldName, "\f", `\u000c`)
		fmt.Fprintf(buf, `"%s": %s`, escapedFieldName, string(fieldVal))
		if i < len(p)-1 {
			fmt.Fprint(buf, `,`)
		}
	}
	fmt.Fprint(buf, `}`)

	return buf.Bytes(), nil
}

func NewProperties(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Properties, error) {
	if vt == jsonparser.Object {
		props := Properties{}
		err := jsonparser.ObjectEach(jsonVal, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			var err error
			prop := &NamedProperty{
				Name: string(key),
			}
			prop.Property, err = parentSchema.Parse(value)
			if err != nil {
				return err
			}
			props = append(props, prop)
			return err
		})

		if err != nil {
			return nil, err
		}

		return &props, nil
	}

	return nil, fmt.Errorf("expected properties to be object, got: %s", vt.String())
}

// type Definitions map[string]*Schema
// type tmpDefinitions Definitions // To ensure MarshalJSON doesn't go haywire

// func (p Definitions) MarshalJSON() ([]byte, error) {
// 	b, err := json.Marshal(tmpDefinitions(p))
// 	return b, err
// }

// func NewDefinitions(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Properties, error) {
// 	if vt == jsonparser.Object {
// 		props := Properties{schemas: map[string]*Schema{}, sortedFields: []string{}}
// 		err := jsonparser.ObjectEach(jsonVal, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
// 			var err error
// 			props.schemas[string(key)], err = parentSchema.Parse(value)
// 			props.sortedFields = append(props.sortedFields, string(key))
// 			return err
// 		})

// 		if err != nil {
// 			return nil, err
// 		}

// 		return &props, nil
// 	}

// 	return nil, fmt.Errorf("expected properties to be object, got: %s", vt.String())
// }

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
				errs = addError(parseErr, errs)
				return
			}

			if dataType == jsonparser.Object || dataType == jsonparser.Boolean {
				schema, err := parentSchema.Parse(value)
				if err != nil {
					errs = addError(err, errs)
					return
				}
				schemas = append(schemas, schema)

			} else {
				errs = addError(fmt.Errorf("expected type to be object or boolean, got: %s", vt.String()), errs)
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
				errs = addError(parseErr, errs)
				return
			}

			if dataType == jsonparser.String {
				unescaped, err := jsonparser.Unescape(value, nil)
				if err != nil {
					errs = addError(parseErr, errs)
					return
				}
				tmpStr := string(unescaped)
				strings = append(strings, &tmpStr)

			} else {
				errs = addError(fmt.Errorf("expected type to be string, got: %s", vt.String()), errs)
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
				errs = addError(parseErr, errs)
				return
			}

			val, err := NewValue(value, dataType)
			if err != nil {
				errs = addError(err, errs)
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
