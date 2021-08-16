package jsonschema

import (
	"encoding/json"
	"regexp"

	"github.com/buger/jsonparser"
)

func (s *Schema) Parse(jsonSchema []byte) (*Schema, error) {
	schema := &Schema{raw: jsonSchema}

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
			errs = AddError(err, errs)
			return
		}

		switch SchemaProp(idx) {
		case PropID:
			schema.ID = NewStringPtr(value)
		case PropIDDraft04:
			schema.IDDraft04 = NewStringPtr(value)
		case PropRef:
			schema.Ref, err = NewRef(value, vt, schema)
			errs = AddError(err, errs)
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
				AddError(err, errs)
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
			errs = AddError(err, errs)
			return
		}

		switch SchemaProp(idx) {
		// case PropID:
		// 	schema.ID = NewStringPtr(value)
		// case PropIDDraft04:
		// 	schema.IDDraft04 = NewStringPtr(value)
		// case PropRef:
		// 	schema.Ref, err = NewRef(value, vt, schema)
		// 	errs = AddError(err, errs)
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
			errs = AddError(err, errs)
		case PropEnum:
			schema.Enum, err = NewEnum(value, vt)
			errs = AddError(err, errs)
		case PropDefault:
			schema.Default, err = NewValue(value, vt)
			errs = AddError(err, errs)
		case PropExamples:
			schema.Examples, err = NewValues(value, vt)
			errs = AddError(err, errs)
		case PropConst:
			schema.Const, err = NewValue(value, vt)
			errs = AddError(err, errs)
		case PropReadOnly:
			tmpBool, err := jsonparser.ParseBoolean(value)
			schema.ReadOnly = &tmpBool
			errs = AddError(err, errs)
		case PropWriteOnly:
			tmpBool, err := jsonparser.ParseBoolean(value)
			schema.WriteOnly = &tmpBool
			errs = AddError(err, errs)
		case PropDefinitions:
			schema.Definitions, err = NewDefinitions(value, vt, schema)
			errs = AddError(err, errs)
		case PropIf:
			schema.If, err = schema.Parse(value)
			errs = AddError(err, errs)
		case PropThen:
			schema.Then, err = schema.Parse(value)
			errs = AddError(err, errs)
		case PropElse:
			schema.Else, err = schema.Parse(value)
			errs = AddError(err, errs)
		case PropAllOf:
			schema.AllOf, err = NewSubSchemas(value, vt, schema)
			errs = AddError(err, errs)
		case PropAnyOf:
			schema.AnyOf, err = NewSubSchemas(value, vt, schema)
			errs = AddError(err, errs)
		case PropOneOf:
			schema.OneOf, err = NewSubSchemas(value, vt, schema)
			errs = AddError(err, errs)
		case PropNot:
			schema.Not, err = schema.Parse(value)
			errs = AddError(err, errs)
		case PropContentEncoding:
			schema.ContentEncoding = NewStringPtr(value)
		case PropContentMediaType:
			schema.ContentMediaType = NewStringPtr(value)
		case PropProperties:
			schema.Properties, err = NewProperties(value, vt, schema)
			errs = AddError(err, errs)
		case PropRequired:
			schema.Required, err = NewStrings(value, vt)
			errs = AddError(err, errs)
		case PropMaxProperties:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MaxProperties = &tmpInt
			errs = AddError(err, errs)
		case PropMinProperties:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MinProperties = &tmpInt
			errs = AddError(err, errs)
		case PropDependencies:
			schema.Dependencies, err = NewDependencies(value, vt, schema)
			errs = AddError(err, errs)
		case PropPatternProperties:
			schema.PatternProperties, err = NewProperties(value, vt, schema)
			errs = AddError(err, errs)
			// Pre-compile the regexps
			if schema.PatternProperties != nil {
				schema.patternPropertiesRegexps = &map[string]*regexp.Regexp{}
				for reStr := range *schema.PatternProperties {
					reStr = ConvertRegexp(reStr)
					re, err := regexp.Compile(reStr)
					if err != nil {
						errs = AddError(err, errs)
					} else {
						(*schema.patternPropertiesRegexps)[reStr] = re
					}
				}
			}
		case PropAdditionalProperties:
			schema.AdditionalProperties, err = schema.Parse(value)
			errs = AddError(err, errs)
		case PropPropertyNames:
			schema.PropertyNames, err = schema.Parse(value)
			errs = AddError(err, errs)
		case PropItems:
			schema.Items, err = NewItems(value, vt, schema)
			errs = AddError(err, errs)
		case PropMaxItems:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MaxItems = &tmpInt
			errs = AddError(err, errs)
		case PropMinItems:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MinItems = &tmpInt
			errs = AddError(err, errs)
		case PropUniqueItems:
			tmpBool, err := jsonparser.ParseBoolean(value)
			schema.UniqueItems = &tmpBool
			errs = AddError(err, errs)
		case PropAdditionalItems:
			schema.AdditionalItems, err = schema.Parse(value)
			errs = AddError(err, errs)
		case PropContains:
			schema.Contains, err = schema.Parse(value)
			errs = AddError(err, errs)
		case PropMaxLength:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MaxLength = &tmpInt
			errs = AddError(err, errs)
		case PropMinLength:
			tmpInt, err := jsonparser.ParseInt(value)
			schema.MinLength = &tmpInt
			errs = AddError(err, errs)
		case PropFormat:
			schema.Format = NewStringPtr(value)
		case PropPattern:
			schema.Pattern = NewStringPtr(value)
			reStr := ConvertRegexp(*schema.Pattern)
			re, err := regexp.Compile(reStr)
			if err != nil {
				errs = AddError(err, errs)
			} else {
				schema.patternRegexp = re
			}
		case PropMultipleOf:
			tmpNum := json.Number(string(value))
			schema.MultipleOf = &tmpNum
		case PropMaximum:
			schema.Maximum, err = NewValue(value, vt)
			errs = AddError(err, errs)
		case PropExclusiveMaximum:
			schema.ExclusiveMaximum, err = NewValue(value, vt)
			errs = AddError(err, errs)
		case PropMinimum:
			schema.Minimum, err = NewValue(value, vt)
			errs = AddError(err, errs)
		case PropExclusiveMinimum:
			schema.ExclusiveMinimum, err = NewValue(value, vt)
			errs = AddError(err, errs)
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
	s.validators = append(s.validators, ValidateValue)

	if s.boolean != nil {
		s.validators = append(s.validators, ValidateBoolean)
	}

	if s.Ref != nil {
		s.validators = append(s.validators, ValidateRef)
		return
	}

	if s.Items != nil || s.AdditionalItems != nil || s.MaxItems != nil || s.MinItems != nil || s.UniqueItems != nil || s.Contains != nil {
		s.validators = append(s.validators, ValidateItems)
	}

	// if s.AdditionalItems != nil {
	// 	s.validators = append(s.validators, ValidateAdditionalItems)
	// }
	// if s.MaxItems != nil {
	// 	s.validators = append(s.validators, ValidateMaxItems)
	// }
	// if s.MinItems != nil {
	// 	s.validators = append(s.validators, ValidateMinItems)
	// }
	// if s.UniqueItems != nil {
	// 	s.validators = append(s.validators, ValidateUniqueItems)
	// }
	// if s.Contains != nil {
	// 	s.validators = append(s.validators, ValidateContains)
	// }

	if s.Properties != nil || s.PatternProperties != nil || s.AdditionalProperties != nil || s.MaxProperties != nil || s.MinProperties != nil {
		s.validators = append(s.validators, ValidateProperties)
	}
	// if s.PatternProperties != nil {
	// 	s.validators = append(s.validators, ValidatePatternProperties)
	// }
	// if s.AdditionalProperties != nil {
	// 	s.validators = append(s.validators, ValidateAdditionalProperties)
	// }
	// if s.MaxProperties != nil {
	// 	s.validators = append(s.validators, ValidateMaxProperties)
	// }
	// if s.MinProperties != nil {
	// 	s.validators = append(s.validators, ValidateMinProperties)
	// }

	if s.PropertyNames != nil {
		s.validators = append(s.validators, ValidatePropertyNames)
	}

	if s.Type != nil {
		s.validators = append(s.validators, ValidateType)
	}

	if s.Pattern != nil {
		s.validators = append(s.validators, ValidatePattern)
	}

	if s.Required != nil {
		s.validators = append(s.validators, ValidateRequired)
	}

	if s.Dependencies != nil {
		s.validators = append(s.validators, ValidateDependencies)
	}

	if s.AllOf != nil {
		s.validators = append(s.validators, ValidateAllOf)
	}

	if s.AnyOf != nil {
		s.validators = append(s.validators, ValidateAnyOf)
	}

	if s.OneOf != nil {
		s.validators = append(s.validators, ValidateOneOf)
	}

	if s.Not != nil {
		s.validators = append(s.validators, ValidateNot)
	}

	if s.MultipleOf != nil {
		s.validators = append(s.validators, ValidateMultipleOf)
	}

	if s.Maximum != nil || s.ExclusiveMaximum != nil {
		s.validators = append(s.validators, ValidateMaximum)
	}
	// if s.ExclusiveMaximum != nil {
	// 	s.validators = append(s.validators, ValidateExclusiveMaximum)
	// }

	if s.Minimum != nil || s.ExclusiveMinimum != nil {
		s.validators = append(s.validators, ValidateMinimum)
	}
	// if s.ExclusiveMinimum != nil {
	// 	s.validators = append(s.validators, ValidateExclusiveMinimum)
	// }

	if s.MaxLength != nil {
		s.validators = append(s.validators, ValidateMaxLength)
	}

	if s.MinLength != nil {
		s.validators = append(s.validators, ValidateMinLength)
	}

	if s.Enum != nil {
		s.validators = append(s.validators, ValidateEnum)
	}

	if s.Const != nil {
		s.validators = append(s.validators, ValidateConst)
	}

	if s.If != nil {
		s.validators = append(s.validators, ValidateIf)
	}

	if s.Format != nil {
		s.validators = append(s.validators, ValidateFormat)
	}
}
