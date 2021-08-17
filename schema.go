package jsonschema

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
)

func New(schema []byte) (*Schema, error) {
	var nilSchema *Schema
	return nilSchema.Parse(schema)
	// return new(Schema).Parse(schema)
}

func NewFromString(schema string) (*Schema, error) {
	return New([]byte(schema))
}

type tmpSchema Schema // To ensure MarshalJSON doesn't go haywire

type Schema struct {
	// These are used to be able to reference and de-reference.

	// Raw contains the ray json schema - necessary in some special cases
	raw []byte

	// Root schema is the top most schema.
	root *Schema

	// Base schema is the nearest schema, up the stack, with a non-pointer (#xxx) ($)id set
	base *Schema

	// Parent schema is the nearest schema, up the stack
	parent *Schema

	// This is to make it easier to deal with true / false schemas and avoid having a Schema
	boolean *bool

	// pointers holds references to schemas with ($)id, collected during parsing - the map key is ($)id
	pointers *pointers

	// refs holds pointers to $ref objects to make de-ref'ing easier.
	// These should only be present on the root schema.
	refs *refs

	// baseURI is present on any schema with an $id
	baseURI *url.URL

	// Not sure this is the way to go
	// Array of validator functions.
	// These are added after checking for all possible constraints
	validators []validatorFunc

	Schema    *string `json:"$schema,omitempty"` // If set, must be http://json-schema.org/draft-07/schema#
	ID        *string `json:"$id,omitempty"`     // NOTE: draft-04 has id instead if $id
	IDDraft04 *string `json:"id,omitempty"`      // NOTE: draft-04 has id instead if $id
	Ref       *Ref    `json:"$ref,omitempty"`
	Comment   *string `json:"$comment,omitempty"`

	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`

	Type *Type `json:"type,omitempty"`

	/* Common / shared */

	// Must have at least 1 value
	Enum     *Enum   `json:"enum,omitempty"`
	Default  *Value  `json:"default,omitempty"`
	Examples *Values `json:"examples,omitempty"`
	// Draft 6
	// Only allow 1 value
	Const *Value `json:"const,omitempty"`
	// Draft 7
	ReadOnly    *bool       `json:"readOnly,omitempty"`
	WriteOnly   *bool       `json:"writeOnly,omitempty"`
	Definitions *Properties `json:"definitions,omitempty"`
	// If schemas should look something like (const being the important part):
	//  { "if": { "properties": { "propertyX": { "const": "ValueX" } }, "required": ["propertyX"] } }
	If *Schema `json:"if,omitempty"`
	// One (or both?) of these can be omitted.
	// Both then and else will be ignore, if If is not defined.
	// If any of them are omitted, the value true is used in their place.
	// NOTE: It's not entirely obvious in the documentation, if both can be omitted:
	// https://json-schema.org/understanding-json-schema/reference/conditionals.html#if-then-else
	Then *Schema `json:"then,omitempty"`
	Else *Schema `json:"else,omitempty"`

	AllOf *Schemas `json:"allOf,omitempty"`
	AnyOf *Schemas `json:"anyOf,omitempty"`
	OneOf *Schemas `json:"oneOf,omitempty"`
	Not   *Schema  `json:"not,omitempty"`

	ContentEncoding  *string `json:"contentEncoding,omitempty"`  // e.g. base64
	ContentMediaType *string `json:"contentMediaType,omitempty"` // e.g. image/png

	/* Objects */

	Properties *Properties `json:"properties,omitempty"`
	// Draft 4 requires at least 1 string
	Required      *Strings `json:"required,omitempty"`
	MaxProperties *int64   `json:"maxProperties,omitempty"`
	MinProperties *int64   `json:"minProperties,omitempty"`
	// Dependencies is either:
	//  - if propertyX is set, then propertyY and propertyZ is required
	//     e.g.: { "propertyX": ["propertyY", "propertyZ"] }
	//  - if propertyX is set, then schemaX is also required to match
	//     e.g.: { "propertyX": { "properties": { "propertyY": { "type": "string" } }, "required": ["propertyY"] } }
	Dependencies *Dependencies `json:"dependencies,omitempty"`
	// patternProperties is used to match property names against a regex and for each a schema.
	// It's basically a map of schemas, but with regex instead of property names.
	PatternProperties        *Properties `json:"patternProperties,omitempty"`
	patternPropertiesRegexps *map[string]*regexp.Regexp
	// additionalProperties is a schema that will be used to validate any properties
	//  in the instance that are not matched by properties or patternProperties.
	// Setting it to false means no additional properties will be allowed.
	AdditionalProperties *Schema `json:"additionalProperties,omitempty"`
	//
	// Draft 6
	// Useful for enforcing a certain property name format
	// Property names implies { "type": "string" }
	// "propertyNames": { "pattern": "^[A-Za-z_][A-Za-z0-9_]*$"}
	PropertyNames *Schema `json:"propertyNames,omitempty"`

	/* Arrays */

	// When items is an array of multiples Schemas, each refers to their own index.
	Items       *Items `json:"items,omitempty"` // TODO: Can actually also be boolean
	MaxItems    *int64 `json:"maxItems,omitempty"`
	MinItems    *int64 `json:"minItems,omitempty"`
	UniqueItems *bool  `json:"uniqueItems,omitempty"`
	// Should only be evaluated when items is multiple schemas.
	// Any values that does not have an explicit schmea (multi schema),
	//  will validate according to this schema.
	// Setting it to false, means that no other values are allowed.
	AdditionalItems *Schema `json:"additionalItems,omitempty"`
	// contains only need to match 1 item in the documents array
	Contains *Schema `json:"contains,omitempty"`

	/* String */

	MaxLength     *int64  `json:"maxLength,omitempty"`
	MinLength     *int64  `json:"minLength,omitempty"`
	Format        *string `json:"format,omitempty"`
	Pattern       *string `json:"pattern,omitempty"`
	patternRegexp *regexp.Regexp

	/* Integer / number */

	// The type (int/float) should of course match the type of the property
	MultipleOf *json.Number `json:"multipleOf,omitempty"`
	// Draft 4: x ≥ minimum unless exclusiveMinimum == true, x ≤ maximum unless exclusiveMaximum == true
	// Draft 6: x ≥ minimum, x > exclusiveMinimum, x ≤ maximum, x < exclusiveMaximum
	Maximum          *Value `json:"maximum,omitempty"`
	ExclusiveMaximum *Value `json:"exclusiveMaximum,omitempty"` // bool in draft 4
	Minimum          *Value `json:"minimum,omitempty"`
	ExclusiveMinimum *Value `json:"exclusiveMinimum,omitempty"` // bool in draft 4
}

func (s Schema) MarshalJSON() ([]byte, error) {
	if s.boolean != nil {
		return []byte(strconv.FormatBool(*s.boolean)), nil
	}

	if s.root == nil && s.refs != nil {
		for _, ref := range *s.refs {
			ref.marshalled = 0
		}
	}

	if s.Ref != nil && s.Ref.Schema != nil {
		if s.Ref.marshalled > 2 {
			return []byte(fmt.Sprintf(`{"$ref": "%s"}`, *s.Ref.String)), nil
		}

		s.Ref.marshalled++

		return json.Marshal(tmpSchema(*s.Ref.Schema))
	}

	b, err := json.Marshal(tmpSchema(s))
	return b, err
}

func (s Schema) String() string {
	schema, err := s.MarshalJSON()
	if err != nil {
		return ""
	}

	return string(schema)
}

func (s *Schema) findPatternProperties(key []byte) []*Schema {
	if s.patternPropertiesRegexps == nil {
		return nil
	}

	schemas := []*Schema{}
	for reStr, re := range *s.patternPropertiesRegexps {
		if re.Match(key) {
			schemas = append(schemas, (*s.PatternProperties)[reStr])
		}
	}

	if len(schemas) > 0 {
		return schemas
	}

	return nil
}

// func (s *Schema) findPatternProperty(key []byte) (*Schema, bool) {
// 	if s.patternPropertiesRegexps == nil {
// 		return nil, false
// 	}

// 	for reStr, re := range *s.patternPropertiesRegexps {
// 		if re.Match(key) {
// 			return (*s.PatternProperties)[reStr], true
// 		}
// 	}
// 	return nil, false
// }

func (s Schema) IsDraft4() bool {
	if s.Schema != nil {
		if *s.Schema == "http://json-schema.org/draft-04/schema#" {
			return true
		}
		if *s.Schema == "http://json-schema.org/draft-05/schema#" {
			// Draft 5 was a no-change patch for Draft 4
			return true
		}
		if *s.Schema == "http://json-schema.org/schema#" {
			// Means "latest schema", this was deprectaed after Draft 4
			return true
		}
	}

	return (s.Schema != nil && *s.Schema == "http://json-schema.org/draft-04/schema#")
}

func (s Schema) IsDraft6() bool {
	return (s.Schema != nil && *s.Schema == "http://json-schema.org/draft-06/schema#")
}

func (s Schema) IsDraft7() bool {
	return (s.Schema != nil && *s.Schema == "http://json-schema.org/draft-07/schema#")
}

func (s Schema) GetID() string {
	if s.ID != nil {
		return *s.ID
	}
	if s.IDDraft04 != nil {
		return *s.IDDraft04
	}

	return ""
}

// Checks if everything is nil and thereby an empty schema, similar to a "true" schema
// TODO: Update with 20xx-xx props
func (s Schema) IsEmpty() bool {
	return ((s.boolean == nil) &&
		(s.Schema == nil) &&
		(s.ID == nil) &&
		(s.IDDraft04 == nil) &&
		(s.Ref == nil) &&
		(s.Comment == nil) &&
		(s.Title == nil) &&
		(s.Description == nil) &&
		(s.Type == nil) &&
		(s.Enum == nil) &&
		(s.Default == nil) &&
		(s.Const == nil) &&
		(s.Examples == nil) &&
		(s.ReadOnly == nil) &&
		(s.WriteOnly == nil) &&
		(s.Definitions == nil) &&
		(s.If == nil) &&
		(s.Then == nil) &&
		(s.Else == nil) &&
		(s.AllOf == nil) &&
		(s.AnyOf == nil) &&
		(s.OneOf == nil) &&
		(s.Not == nil) &&
		(s.ContentEncoding == nil) &&
		(s.ContentMediaType == nil) &&
		(s.Properties == nil) &&
		(s.Required == nil) &&
		(s.MaxProperties == nil) &&
		(s.MinProperties == nil) &&
		(s.Dependencies == nil) &&
		(s.PatternProperties == nil) &&
		(s.AdditionalProperties == nil) &&
		(s.PropertyNames == nil) &&
		(s.Items == nil) &&
		(s.MaxItems == nil) &&
		(s.MinItems == nil) &&
		(s.UniqueItems == nil) &&
		(s.AdditionalItems == nil) &&
		(s.Contains == nil) &&
		(s.MaxLength == nil) &&
		(s.MinLength == nil) &&
		(s.Format == nil) &&
		(s.Pattern == nil) &&
		(s.MultipleOf == nil) &&
		(s.Maximum == nil) &&
		(s.ExclusiveMaximum == nil) &&
		(s.Minimum == nil) &&
		(s.ExclusiveMinimum == nil))
}
