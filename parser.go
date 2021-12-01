package jsonschema

import (
	"encoding/json"
	"regexp"

	"github.com/buger/jsonparser"
)

func (s *Schema) Parse(jsonSchema []byte) (*Schema, error) {
	schema := &Schema{raw: jsonSchema, circularThreshold: 3}

	if s == nil {
		schema.pointers = &pointers{}
		schema.refs = &refs{}

	} else {
		schema.parent = s

		// Set the base schema
		if s.baseURI != nil || s.base == nil {
			schema.base = s
		} else if s.base != nil {
			schema.base = s.base
		}

		// Set the root schema
		if s.root != nil {
			schema.root = s.root
		} else {
			schema.root = s
		}
	}

	if b, err := jsonparser.ParseBoolean(jsonSchema); err == nil {
		schema.boolean = &b

		schema.setupValidators()

		return schema, nil
	}

	var errs error

	// Simplest solution for getting ($)id, which is needed for separation and de-ref'ing
	// // Simplest solution for getting ($)id and $ref first, which is needed for separation and de-ref'ing
	jsonparser.EachKey(jsonSchema, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
		if err != nil {
			errs = addError(err, errs)
			return
		}

		switch SchemaProp(idx) {
		case PropID:
			schema.ID = NewStringPtr(value)
		case PropIDDraft04:
			schema.IDDraft04 = NewStringPtr(value)
		case PropRef:
			schema.Ref, err = NewRef(value, vt, schema)
			errs = addError(err, errs)
		}
	}, [][]string{PropID: {"$id"}, PropIDDraft04: {"id"}, PropRef: {"$ref"}}...)

	id := schema.GetID()

	if id != "" && schema.Ref == nil {
		if len(id) > 0 && id[:1] == "#" {
			// Do not expand or change base uri
			if s != nil {
				s.setPointer(id, schema)
			}
		} else {
			var err error
			schema.baseURI, err = s.ExpandURI(id)
			if err != nil {
				addError(err, errs)
				return nil, errs
			}

			schema.pointers = &pointers{"#": schema}

			if s != nil {
				s.setPointer(schema.baseURI.String(), schema)
			} else {
				schema.setPointer(schema.baseURI.String(), schema)
			}
		}
	}

	jsonparser.EachKey(jsonSchema, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
		if err != nil {
			errs = addError(err, errs)
			return
		}

		switch SchemaProp(idx) {
		// case PropID:
		// 	schema.ID = NewStringPtr(value)
		// case PropIDDraft04:
		// 	schema.IDDraft04 = NewStringPtr(value)
		// case PropRef:
		// 	schema.Ref, err = NewRef(value, vt, schema)
		// 	errs = addError(err, errs)
		case PropSchema:
			schema.Schema = NewStringPtr(value)
		case PropComment:
			schema.Comment = NewStringPtr(value)
		case PropTitle:
			schema.Title = NewStringPtr(value)
		case PropDescription:
			schema.Description = NewStringPtr(value)
		case PropType:
			schema.Type, err = NewType(value, vt)
			errs = addError(err, errs)
		case PropEnum:
			schema.Enum, err = NewEnum(value, vt)
			errs = addError(err, errs)
		case PropDefault:
			schema.Default, err = NewValue(value, vt)
			errs = addError(err, errs)
		case PropExamples:
			schema.Examples, err = NewValues(value, vt)
			errs = addError(err, errs)
		case PropConst:
			schema.Const, err = NewValue(value, vt)
			errs = addError(err, errs)
		case PropReadOnly:
			tmpBool, err := jsonparser.ParseBoolean(value)
			schema.ReadOnly = &tmpBool
			errs = addError(err, errs)
		case PropWriteOnly:
			tmpBool, err := jsonparser.ParseBoolean(value)
			schema.WriteOnly = &tmpBool
			errs = addError(err, errs)
		case PropDefinitions:
			schema.Definitions, err = NewProperties(value, vt, schema)
			errs = addError(err, errs)
		case PropIf:
			schema.If, err = schema.Parse(value)
			errs = addError(err, errs)
		case PropThen:
			schema.Then, err = schema.Parse(value)
			errs = addError(err, errs)
		case PropElse:
			schema.Else, err = schema.Parse(value)
			errs = addError(err, errs)
		case PropAllOf:
			schema.AllOf, err = NewSubSchemas(value, vt, schema)
			errs = addError(err, errs)
		case PropAnyOf:
			schema.AnyOf, err = NewSubSchemas(value, vt, schema)
			errs = addError(err, errs)
		case PropOneOf:
			schema.OneOf, err = NewSubSchemas(value, vt, schema)
			errs = addError(err, errs)
		case PropNot:
			schema.Not, err = schema.Parse(value)
			errs = addError(err, errs)
		case PropContentEncoding:
			schema.ContentEncoding = NewStringPtr(value)
		case PropContentMediaType:
			schema.ContentMediaType = NewStringPtr(value)
		case PropProperties:
			schema.Properties, err = NewProperties(value, vt, schema)
			errs = addError(err, errs)
		case PropRequired:
			schema.Required, err = NewStrings(value, vt)
			errs = addError(err, errs)
		case PropMaxProperties:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MaxProperties = &tmpInt
			errs = addError(err, errs)
		case PropMinProperties:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MinProperties = &tmpInt
			errs = addError(err, errs)
		case PropDependencies:
			schema.Dependencies, err = NewDependencies(value, vt, schema)
			errs = addError(err, errs)
		case PropPatternProperties:
			schema.PatternProperties, err = NewProperties(value, vt, schema)
			errs = addError(err, errs)
			// Pre-compile the regexps
			if schema.PatternProperties != nil {
				schema.patternPropertiesRegexps = &map[string]*regexp.Regexp{}
				for _, prop := range *schema.PatternProperties {
					reStr := convertRegexp(prop.Name)
					re, err := regexp.Compile(reStr)
					if err != nil {
						errs = addError(err, errs)
					} else {
						(*schema.patternPropertiesRegexps)[prop.Name] = re
					}
				}
			}
		case PropAdditionalProperties:
			schema.AdditionalProperties, err = schema.Parse(value)
			errs = addError(err, errs)
		case PropPropertyNames:
			schema.PropertyNames, err = schema.Parse(value)
			errs = addError(err, errs)
		case PropItems:
			schema.Items, err = NewItems(value, vt, schema)
			errs = addError(err, errs)
		case PropMaxItems:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MaxItems = &tmpInt
			errs = addError(err, errs)
		case PropMinItems:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MinItems = &tmpInt
			errs = addError(err, errs)
		case PropUniqueItems:
			tmpBool, err := jsonparser.ParseBoolean(value)
			schema.UniqueItems = &tmpBool
			errs = addError(err, errs)
		case PropAdditionalItems:
			schema.AdditionalItems, err = schema.Parse(value)
			errs = addError(err, errs)
		case PropContains:
			schema.Contains, err = schema.Parse(value)
			errs = addError(err, errs)
		case PropMaxLength:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MaxLength = &tmpInt
			errs = addError(err, errs)
		case PropMinLength:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MinLength = &tmpInt
			errs = addError(err, errs)
		case PropFormat:
			schema.Format = NewStringPtr(value)
		case PropPattern:
			schema.Pattern = NewStringPtr(value)
			reStr := convertRegexp(*schema.Pattern)
			re, err := regexp.Compile(reStr)
			if err != nil {
				errs = addError(err, errs)
			} else {
				schema.patternRegexp = re
			}
		case PropMultipleOf:
			tmpNum := json.Number(string(value))
			schema.MultipleOf = &tmpNum
		case PropMaximum:
			schema.Maximum, err = NewValue(value, vt)
			errs = addError(err, errs)
		case PropExclusiveMaximum:
			schema.ExclusiveMaximum, err = NewValue(value, vt)
			errs = addError(err, errs)
		case PropMinimum:
			schema.Minimum, err = NewValue(value, vt)
			errs = addError(err, errs)
		case PropExclusiveMinimum:
			schema.ExclusiveMinimum, err = NewValue(value, vt)
			errs = addError(err, errs)
		}
	}, propNames...)

	schema.setupValidators()

	return schema, errs
}

// Cases
// - #/deffintions - no change of base uri
// - #... - no change of base uri
// - http.... - change of base uri
// - anything else - change of base uri
// - also: base uri changes, but id is not expanded
func (s *Schema) setPointer(key string, schema *Schema) {
	if s != nil {
		if len(key) > 7 && (key[0:7] != "http://" || key[0:8] != "https://") && s.root != nil && s.root.pointers != nil {
			(*s.root.pointers)[key] = schema
		} else if s.pointers != nil {
			(*s.pointers)[key] = schema
		} else if s.base != nil && s.base.pointers != nil {
			(*s.base.pointers)[key] = schema
		}
	}
}

func (s *Schema) getPointer(key string) *Schema {
	if s != nil {

		if s.baseURI != nil && s.baseURI.String() == key {
			return s
		}
		if len(key) > 7 && (key[0:7] != "http://" || key[0:8] != "https://") && s.root != nil && s.root.pointers != nil {
			return (*s.root.pointers)[key]
		}
		if s.pointers != nil && (*s.pointers)[key] != nil {
			return (*s.pointers)[key]
		}
		if s.base != nil && s.base.pointers != nil {
			return (*s.base.pointers)[key]
		}
	}
	return nil
}

func (s *Schema) setRef(ref *Ref) {
	if s != nil {
		if s.root != nil {
			(*s.root.refs) = append((*s.root.refs), ref)
		} else {
			(*s.refs) = append((*s.refs), ref)
		}
	}
}

// NOTE: There is probably a lot of performance to gain here,
//       e.g. by excluding everything else if $ref is set
func (s *Schema) setupValidators() {
	s.validators = []validatorFunc{}

	// Always start by validating the value
	s.validators = append(s.validators, validateValue)

	if s.boolean != nil {
		s.validators = append(s.validators, validateBooleanSchema)
	}

	if s.Ref != nil {
		s.validators = append(s.validators, validateRef)
		return
	}

	if s.Items != nil || s.AdditionalItems != nil || s.MaxItems != nil || s.MinItems != nil || s.UniqueItems != nil || s.Contains != nil {
		s.validators = append(s.validators, validateItems)
	}

	// if s.AdditionalItems != nil {
	// 	s.validators = append(s.validators, validateAdditionalItems)
	// }
	// if s.MaxItems != nil {
	// 	s.validators = append(s.validators, validateMaxItems)
	// }
	// if s.MinItems != nil {
	// 	s.validators = append(s.validators, validateMinItems)
	// }
	// if s.UniqueItems != nil {
	// 	s.validators = append(s.validators, validateUniqueItems)
	// }
	// if s.Contains != nil {
	// 	s.validators = append(s.validators, validateContains)
	// }

	if s.Properties != nil || s.PatternProperties != nil || s.AdditionalProperties != nil || s.MaxProperties != nil || s.MinProperties != nil {
		s.validators = append(s.validators, validateProperties)
	}
	// if s.PatternProperties != nil {
	// 	s.validators = append(s.validators, validatePatternProperties)
	// }
	// if s.AdditionalProperties != nil {
	// 	s.validators = append(s.validators, validateAdditionalProperties)
	// }
	// if s.MaxProperties != nil {
	// 	s.validators = append(s.validators, validateMaxProperties)
	// }
	// if s.MinProperties != nil {
	// 	s.validators = append(s.validators, validateMinProperties)
	// }

	if s.PropertyNames != nil {
		s.validators = append(s.validators, validatePropertyNames)
	}

	if s.Type != nil {
		s.validators = append(s.validators, validateType)
	}

	if s.Pattern != nil {
		s.validators = append(s.validators, validatePattern)
	}

	if s.Required != nil {
		s.validators = append(s.validators, validateRequired)
	}

	if s.Dependencies != nil {
		s.validators = append(s.validators, validateDependencies)
	}

	if s.AllOf != nil {
		s.validators = append(s.validators, validateAllOf)
	}

	if s.AnyOf != nil {
		s.validators = append(s.validators, validateAnyOf)
	}

	if s.OneOf != nil {
		s.validators = append(s.validators, validateOneOf)
	}

	if s.Not != nil {
		s.validators = append(s.validators, validateNot)
	}

	if s.MultipleOf != nil {
		s.validators = append(s.validators, validateMultipleOf)
	}

	if s.Maximum != nil || s.ExclusiveMaximum != nil {
		s.validators = append(s.validators, validateMaximum)
	}
	// if s.ExclusiveMaximum != nil {
	// 	s.validators = append(s.validators, validateExclusiveMaximum)
	// }

	if s.Minimum != nil || s.ExclusiveMinimum != nil {
		s.validators = append(s.validators, validateMinimum)
	}
	// if s.ExclusiveMinimum != nil {
	// 	s.validators = append(s.validators, validateExclusiveMinimum)
	// }

	if s.MaxLength != nil {
		s.validators = append(s.validators, validateMaxLength)
	}

	if s.MinLength != nil {
		s.validators = append(s.validators, validateMinLength)
	}

	if s.Enum != nil {
		s.validators = append(s.validators, validateEnum)
	}

	if s.Const != nil {
		s.validators = append(s.validators, validateConst)
	}

	if s.If != nil {
		s.validators = append(s.validators, validateIf)
	}

	if s.Format != nil {
		s.validators = append(s.validators, validateFormat)
	}
}
