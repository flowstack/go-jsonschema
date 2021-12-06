package jsonschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/buger/jsonparser"
	"golang.org/x/net/idna"
)

type validatorFunc func(value []byte, vt ValueType, schema *Schema) error

// TODO: Benchmark whether by ref or by pointer is the most performant

func validate(value []byte, vt ValueType, schema *Schema) error {
	var err error

	if schema == nil {
		return errors.New("no schema supplied")
	}

	if vt == String {
		value, err = jsonparser.Unescape(value, nil)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	if len(schema.validators) == 0 {
		return errors.New("no validators found - at least 1 was expected")
	}

	for _, validator := range schema.validators {
		err = validator(value, vt, schema)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateValue(value []byte, vt ValueType, schema *Schema) error {
	// If the we have an empty value and the schema is not boolean (false), then the doc is invalid
	// if len(value) == 0 && schema.boolean != nil && !*schema.boolean {
	if len(value) == 0 && schema.IsEmpty() {
		return errors.New(`empty document is not valid against any other schemas than "false"`)
	}
	return nil
}

func validateBooleanSchema(value []byte, vt ValueType, schema *Schema) error {
	// Start by checking for empty JSON value
	if len(value) == 0 {
		// If we have an empty value and a boolean false schema then the value is valid
		if !*schema.boolean {
			return nil
		}
		// If we do not have a boolean false schema, but have an empty value, then the doc is invalid
		return errors.New("empty document does not validate against the schema")
	}

	// If we have a value and a boolean true schema then the value is valid
	if *schema.boolean {
		return nil
	}
	return errors.New("document does not match the false schema")
}

func validateRef(value []byte, vt ValueType, schema *Schema) error {
	refSchema, err := schema.ResolveRef(schema.Ref)
	if err != nil {
		log.Println(err)
		return err
	}

	return validate(value, vt, refSchema)
}

func validateItems(value []byte, vt ValueType, schema *Schema) error {
	if schema == nil {
		return errors.New("empty schema")
	}

	// Ignore non-arrays
	if vt != Array {
		return nil
	}

	// Used to ensure uniqueness
	unique := newUniqueValidator()

	// Contains validator
	contains := false

	// Start by checking if we have boolean schema
	if schema.Items != nil && schema.Items.Boolean != nil {
		if *schema.Items.Boolean && len(value) > 0 {
			return nil
		} else if !*schema.Items.Boolean && len(value) <= 2 { // empty array matches boolean false schema
			return nil
		}
		return errors.New("items doesn't match schema")
	}

	idx := -1
	var errs error
	_, parseErr := jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, parseErr error) {
		idx++

		// Don't spent time validating, if we already have a parser error
		if parseErr != nil {
			errs = addError(parseErr, errs)
			return
		}

		if schema.UniqueItems != nil && *schema.UniqueItems {
			if unique.Exists(value, dataType) {
				errs = addError(errors.New("values are not unique"), errs)
				return
			}
		}

		if schema.Contains != nil {
			err := validate(value, ValueType(dataType), schema.Contains)
			if err == nil {
				contains = true
			}
		}

		if schema.Items == nil {
			// Items default to empty schema (anything is valid)
			// So do nothing

		} else if schema.Items.Schema != nil {
			err := validate(value, ValueType(dataType), schema.Items.Schema)
			errs = addError(err, errs)

		} else if schema.Items.Schemas != nil && idx < len(*schema.Items.Schemas) {
			err := validate(value, ValueType(dataType), (*schema.Items.Schemas)[idx])
			errs = addError(err, errs)

		} else if schema.AdditionalItems == nil {
			// It's allowed to have more items than schemas
			// So do nothing

		} else if schema.AdditionalItems != nil && (schema.IsDraft4() || len(*schema.Items.Schemas) > 0) {
			// Only draft 4 allows addtionalItems without items as well
			err := validate(value, ValueType(dataType), schema.AdditionalItems)
			errs = addError(err, errs)

		} else {
			errs = addError(fmt.Errorf("index %d has no schema to match against", idx), errs)
		}
	})

	if schema.Contains != nil && !contains {
		errs = addError(errors.New("no values matched the contains schema"), errs)
	}

	count := int64(idx + 1)

	if schema.MaxItems != nil {
		if count > *schema.MaxItems {
			errs = addError(errors.New("too many properties"), errs)
		}
	}

	if schema.MinItems != nil {
		if count < *schema.MinItems {
			errs = addError(errors.New("too few properties"), errs)
		}
	}

	if parseErr != nil {
		errs = addError(parseErr, errs)
	}

	return errs
}

// Unless required, any property can be left out
// Properties not defined in the schema are allowed, unless additionProperties == false
func validateProperties(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything other than Objects (probably an Array)
	if vt != Object {
		return nil
	}

	// Keep a counter for min and max properties checks
	var count int64
	errs := jsonparser.ObjectEach(value, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		count++

		var hasSchema bool
		var subSchema *Schema

		if schema.Properties != nil {
			var subProp *NamedProperty
			subProp, hasSchema = schema.Properties.GetProperty(string(key))
			if hasSchema {
				subSchema = subProp.Property
			}
		}

		if schema.PatternProperties != nil {
			subSchemas := schema.findPatternProperties(key)
			if len(subSchemas) > 0 {
				hasSchema = true
				for _, subSchema := range subSchemas {
					subSchema.name = string(key)
					if subSchema != nil {
						err := validate(value, ValueType(dataType), subSchema)
						if err != nil {
							return err
						}
					}
				}
			}
		}

		if !hasSchema && schema.AdditionalProperties != nil {
			subSchema = schema.AdditionalProperties
		}

		if subSchema != nil {
			subSchema.name = string(key)
			return validate(value, ValueType(dataType), subSchema)
		}
		return nil
	})

	if schema.MaxProperties != nil {
		if count > *schema.MaxProperties {
			errs = addError(errors.New("too many properties"), errs)
		}
	}

	if schema.MinProperties != nil {
		if count < *schema.MinProperties {
			errs = addError(errors.New("too few properties"), errs)
		}
	}

	if schema.Required != nil {
		err := validateRequired(value, vt, schema)
		errs = addError(err, errs)
	}

	return errs
}

// Handled in ValidateProperties
// func validateMaxProperties(value []byte, vt ValueType, schema *Schema) error {
// 	return errors.New("ValidateMaxProperties is not implemented yet")
// }
// func validateMinProperties(value []byte, vt ValueType, schema *Schema) error {
// 	return errors.New("ValidateMinProperties is not implemented yet")
// }
// func validatePatternProperties(value []byte, vt ValueType, schema *Schema) error {
// 	return errors.New("ValidatePatternProperties is not implemented yet")
// }
// func validateAdditionalProperties(value []byte, vt ValueType, schema *Schema) error {
// 	return errors.New("ValidateAdditionalProperties is not implemented yet")
// }

func validatePattern(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything other than Strings
	if vt != String {
		return nil
	}

	if !schema.patternRegexp.Match(value) {
		return errors.New("value did not match pattern")
	}

	return nil
}

// TODO: It might be necessary and / or better to split ValidateProperties into it's
// 	     original multiple ValidateXxx methods, so that they do not depend on a
//       properties object to exist.
func validatePropertyNames(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything other than Objects (probably an Array)
	if vt != Object {
		return nil
	}

	return jsonparser.ObjectEach(value, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		return validate(key, String, schema.PropertyNames)
	})
}

func validateType(value []byte, vt ValueType, schema *Schema) error {
	if vt == Unknown {
		return errors.New("invalid value")
	}

	if schema.Type.String != nil {
		// A value detected as a number, may still be a valid integer
		if *schema.Type.String == "integer" && vt == Number && isInteger(value) {
			// In Draft 4 the value 1.0 can NOT be an integer all other drafts allows this
			if schema.IsDraft4() && strings.Contains(string(value), ".") {
				return fmt.Errorf(`value "%s" is of type %s, but should be of type: %s`, value, vt, *schema.Type.String)
			}
			return nil
		}
		// A detected Integer is also a valid Number
		if *schema.Type.String == "number" && vt == Integer {
			return nil
		}
		if *schema.Type.String == vt.String() {
			return nil
		}

		return fmt.Errorf(`value "%s" is of type %s, but should be of type: %s`, value, vt, *schema.Type.String)

	} else if schema.Type.Strings != nil {
		for _, t := range *schema.Type.Strings {
			// A value detected as a number, may still be a valid integer
			if *t == "integer" && vt == Number && isInteger(value) {
				return nil
			}
			// A detected Integer is also a valid Number
			if *t == "number" && vt == Integer {
				return nil
			}
			if *t == vt.String() {
				return nil
			}
		}

		return fmt.Errorf(`value "%v" is of type %s, but should be of type: %v`, value, vt, *schema.Type.Strings)
	}

	return fmt.Errorf("unknown type")
}

func validateRequired(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything other than Objects
	if vt != Object {
		return nil
	}

	var errs error

	paths := [][]string{}
	for _, str := range *schema.Required {
		paths = append(paths, []string{*str})
	}

	found := 0
	jsonparser.EachKey(value, func(idx int, value []byte, vt jsonparser.ValueType, parseErr error) {
		// Don't spent time validating, if we already have a parser error
		if parseErr != nil {
			errs = addError(parseErr, errs)
			return
		}

		if len(value) == 0 {
			errs = addError(fmt.Errorf("required value not found"), errs)
			return
		}

		found++
	}, paths...)

	if errs != nil {
		return errs
	}

	if found != len(*schema.Required) {
		return errors.New("not all required properties were found")
	}

	return nil
}

func validateDependencies(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything other than Objects
	if vt != Object {
		return nil
	}

	paths := [][]string{}
	for key := range *schema.Dependencies {
		paths = append(paths, []string{key})
	}

	var errs error

	jsonparser.EachKey(value, func(idx int, subVal []byte, dataType jsonparser.ValueType, parseErr error) {
		// Don't spent time validating, if we already have a parser error
		if parseErr != nil {
			errs = addError(parseErr, errs)
			return
		}

		if len(paths[idx]) != 1 {
			errs = addError(errors.New("unexpected path"), errs)
			return
		}

		path := paths[idx][0]
		dep, ok := (*schema.Dependencies)[path]
		if !ok {
			errs = addError(errors.New("path not found in dependencies"), errs)
			return
		}

		if dep.Strings != nil {
			err := validateRequired(value, vt, &Schema{Required: dep.Strings})
			errs = addError(err, errs)
		} else if dep.Schema != nil {
			err := validate(value, vt, dep.Schema)
			errs = addError(err, errs)
		}
	}, paths...)

	return errs
}

func validateAllOf(value []byte, vt ValueType, schema *Schema) error {
	for _, subSchema := range *schema.AllOf {
		err := validate(value, vt, subSchema)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateAnyOf(value []byte, vt ValueType, schema *Schema) error {
	for _, subSchema := range *schema.AnyOf {
		err := validate(value, vt, subSchema)
		if err == nil {
			return nil
		}
	}

	return errors.New("value does not match any of the schemas")
}

func validateOneOf(value []byte, vt ValueType, schema *Schema) error {
	valid := false

	for _, subSchema := range *schema.OneOf {
		err := validate(value, vt, subSchema)
		if err == nil {
			if !valid {
				valid = true
			} else {
				return errors.New("value matches more than one of the schemas")
			}
		}
	}

	if valid {
		return nil
	}

	return errors.New("value does not match one of the schemas")
}

func validateNot(value []byte, vt ValueType, schema *Schema) error {
	err := validate(value, vt, schema.Not)
	if err == nil {
		return errors.New("value should NOT match schema")
	}
	return nil
}

func validateMultipleOf(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything but numbers and integers
	if vt != Number && vt != Integer {
		return nil
	}

	floatVal, _ := new(big.Rat).SetString(string(value))
	mul, _ := new(big.Rat).SetString(string(*schema.MultipleOf))

	if q := new(big.Rat).Quo(floatVal, mul); !q.IsInt() {
		return fmt.Errorf("value (%s) is not a multiple of %s", floatVal.String(), mul.String())
	}

	return nil
}

func validateMaximum(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything but numbers and integers
	if vt != Number && vt != Integer {
		return nil
	}

	floatVal, ok := new(big.Float).SetString(string(value))
	if !ok {
		return errors.New("invalid float value")
	}

	if schema.Maximum != nil && schema.Maximum.Number != nil {
		if schema.ExclusiveMaximum != nil && schema.ExclusiveMaximum.Boolean != nil && *schema.ExclusiveMaximum.Boolean {
			if floatVal.Cmp(schema.Maximum.Number) < 0 {
				return nil
			}
		} else if floatVal.Cmp(schema.Maximum.Number) <= 0 {
			return nil
		}
	}

	if schema.ExclusiveMaximum != nil && schema.ExclusiveMaximum.Number != nil {
		if floatVal.Cmp(schema.ExclusiveMaximum.Number) < 0 {
			return nil
		}
	}

	return errors.New("value is more than maximum")
}

func validateMinimum(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything but numbers and integers
	if vt != Number && vt != Integer {
		return nil
	}

	floatVal, ok := new(big.Float).SetString(string(value))
	if !ok {
		return errors.New("invalid float value")
	}

	if schema.Minimum != nil && schema.Minimum.Number != nil {
		if schema.ExclusiveMinimum != nil && schema.ExclusiveMinimum.Boolean != nil && *schema.ExclusiveMinimum.Boolean {
			if floatVal.Cmp(schema.Minimum.Number) > 0 {
				return nil
			}
		} else if floatVal.Cmp(schema.Minimum.Number) >= 0 {
			return nil
		}
	}

	if schema.ExclusiveMinimum != nil && schema.ExclusiveMinimum.Number != nil {
		if floatVal.Cmp(schema.ExclusiveMinimum.Number) > 0 {
			return nil
		}
	}

	return errors.New("value is more than minimum")
}

func validateMaxLength(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything but strings
	if vt != String {
		return nil
	}
	if utf8.RuneCount(value) > int(*schema.MaxLength) {
		return fmt.Errorf("length of value is more than %d", *schema.MaxLength)
	}
	return nil
}

func validateMinLength(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything but strings
	if vt != String {
		return nil
	}
	if utf8.RuneCount(value) < int(*schema.MinLength) {
		return fmt.Errorf("length of value is less than %d", *schema.MinLength)
	}
	return nil
}

func validateEnum(value []byte, vt ValueType, schema *Schema) error {
	val, err := NewValue(value, vt.ParserValueType())
	if err != nil {
		return err
	}

	for _, e := range *schema.Enum {
		if e.Equal(val) {
			return nil
		}
	}
	return errors.New("value is not part of the enum set")
}

func validateConst(value []byte, vt ValueType, schema *Schema) error {
	if vt == Integer || vt == Number {
		if schema.Const.valueType == Integer || schema.Const.valueType == Number {
			floatVal, _ := new(big.Float).SetString(string(value))
			if schema.Const.Number != nil && floatVal.Cmp(schema.Const.Number) == 0 {
				return nil
			}
		}
	}

	if vt != schema.Const.valueType {
		return errors.New("value type doesn't match const value type in schema")
	}

	if vt == Object || vt == Array {
		buf := &bytes.Buffer{}
		json.Compact(buf, value)
		value = buf.Bytes()
		value = sortObject(value)

	} else if vt == String {
		if schema.Const.String != nil && *schema.Const.String == string(value) {
			return nil
		}
	}

	if bytes.Equal(value, schema.Const.raw) {
		return nil
	}
	return errors.New("values does not match const")
}

func validateIf(value []byte, vt ValueType, schema *Schema) error {
	err := validate(value, vt, schema.If)
	if err == nil && schema.Then != nil {
		return validate(value, vt, schema.Then)

	} else if err == nil && schema.Then == nil {
		// Same as Then being true (valid or schema true?)
		return nil

	} else if err != nil && schema.Else != nil {
		return validate(value, vt, schema.Else)

	} else if err != nil && schema.Else == nil {
		// Same as Else being true (valid or schema true?)
		return nil
	}

	return errors.New("unable to validate value against if / then / else")
}

var reHostname = regexp.MustCompile(`^(?:[a-z0-9]{0,63}|[a-z0-9][a-z0-9\-]{0,61}[a-z0-9])(?:\.(?:[\pL\pN\-]{0,63}|[a-z0-9][a-z0-9\-]{0,61}[a-z0-9]))*?$`)
var reNonEscapedJSONPointerChars = regexp.MustCompile(`~(?:[^0-1]|$)`)
var reCurlyBracketsMatch = regexp.MustCompile(`(?:{\w+.*?})*`)
var reDuration = regexp.MustCompile(`^P(?:\d+W|(?:\d+Y){0,1}(?:\d+M){0,1}(?:\d+D){0,1}(?:T(?:\d+H){0,1}(?:\d+M){0,1}(?:\d+S){0,1}){0,1})$`)
var reUUID = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

func validateFormat(value []byte, vt ValueType, schema *Schema) error {
	// Ignore anything that is not a string
	if vt != String {
		return nil
	}

	// Parse takes a layout string, which defines the format by showing how the reference time,
	// should be interpreted. The reference time is:
	// Mon Jan 2 15:04:05 -0700 MST 2006

	switch *schema.Format {

	case "date-time":
		// Date and time together, for example, 2006-01-02T15:04:05-07:00.
		valueStr := strings.ToUpper(string(value))
		valueStr = strings.Replace(valueStr, "15:59:60", "15:59:59", 1)
		valueStr = strings.Replace(valueStr, "23:59:60", "23:59:59", 1)
		if _, err := time.Parse(time.RFC3339, valueStr); err != nil {
			return err
		}
		// JSON Schema doesn't allow a couple of weird offset cases for some reason, e.g:
		// 24:00, -24:00, -00:60
		if len(valueStr) >= 5 {
			if valueStr[len(valueStr)-5:] == "24:00" {
				return errors.New("invalid offset")
			}
			if valueStr[len(valueStr)-5:] == "00:60" {
				return errors.New("invalid offset")
			}
		}
		return nil

	case "time":
		// Time, for example, 15:04:05-07:00
		// replace leapseconds
		valueStr := strings.ToUpper(string(value))
		valueStr = strings.Replace(valueStr, "23:59:60", "23:59:59", 1)
		valueStr = strings.Replace(valueStr, "15:59:60", "15:59:59", 1)
		t := fmt.Sprintf("1970-01-01T%s", valueStr)
		if _, err := time.Parse(time.RFC3339, t); err != nil {
			return err
		}
		// JSON Schema doesn't allow a couple of weird offset cases for some reason, e.g:
		// 24:00, -24:00, -00:60
		if len(valueStr) >= 5 {
			if valueStr[len(valueStr)-5:] == "24:00" {
				return errors.New("invalid offset")
			}
			if valueStr[len(valueStr)-5:] == "00:60" {
				return errors.New("invalid offset")
			}
		}
		return nil

	case "date":
		// Date, for example, 2006-01-02.
		_, err := time.Parse("2006-01-02", strings.ToUpper(string(value)))
		return err

	case "duration":
		// A duration as defined by the ISO 8601 ABNF for “duration”. For example, P3D expresses a duration of 3 days.
		str := string(value)
		if reDuration.MatchString(str) && len(str) > 1 && str[len(str)-1:] != "T" {
			return nil
		}
		return errors.New("value is not a valid duration")

	case "email":
		// Internet email address, see RFC 5322, section 3.4.1.
		fallthrough
	case "idn-email":
		// The internationalized form of an Internet email address, see RFC 6531.
		_, err := mail.ParseAddress(string(value))
		return err

	case "hostname":
		// Internet host name, see RFC 1123, section 2.1.
		if reHostname.MatchString(strings.ToLower(string(value))) {
			return nil
		}
		return errors.New("value is not a valid hostname")

	case "idn-hostname":
		// An internationalized Internet host name, see RFC5890, section 2.3.2.3.
		// A domain label must contain a minimum of 1 character and a maximum of 63 characters.

		i := idna.New(
			idna.BidiRule(),
			idna.CheckHyphens(true),
			idna.CheckJoiners(true),
			idna.MapForLookup(),
			idna.RemoveLeadingDots(true),
			idna.StrictDomainName(true),
			idna.Transitional(true),
			idna.ValidateForRegistration(),
			idna.ValidateLabels(true),
			idna.VerifyDNSLength(true),
		)
		_, err := i.ToASCII(string(value))
		if err == nil {
			return nil
		}
		return err

	case "ipv4":
		// IPv4 address, according to dotted-quad ABNF syntax as defined in RFC 2673, section 3.2.
		ip := net.ParseIP(string(value))
		if ip.To4() != nil && strings.Count(string(value), ".") == 3 {
			return nil
		}
		return errors.New("value is not a valid IPv4 address")

	case "ipv6":
		// IPv6 address, as defined in RFC 2373, section 2.2.
		ip := net.ParseIP(string(value))
		if ip.To16() != nil && !strings.Contains(string(value), ".") {
			return nil
		}
		return errors.New("value is not a valid IPv6 address")

	case "uuid":
		// A Universally Unique Identifier as defined by RFC 4122. Example: 3e4666bf-d5e5-4aa7-b8ce-cefe41c7568a
		if reUUID.MatchString(strings.ToLower(string(value))) {
			return nil
		}
		return errors.New("value is not a valid UUID")

	case "uri":
		// A universal resource identifier (URI), according to RFC3986.
		u, err := url.Parse(string(value))
		if err == nil {
			if u.Scheme == "" || u.Host == "" {
				err = errors.New("valus is not a valid IRI")
			}
		}
		return err

	case "uri-reference":
		// A URI Reference (either a URI or a relative-reference), according to RFC3986, section 4.1.
		_, err := url.Parse(string(value))
		return err

	case "iri":
		// The internationalized equivalent of a “uri”, according to RFC3987.
		u, err := url.Parse(string(value))
		if err == nil {
			if u.Scheme == "" || u.Host == "" {
				err = errors.New("valus is not a valid IRI")
			}
		}
		return err

	case "iri-reference":
		// The internationalized equivalent of a “uri-reference”, according to RFC3987
		_, err := url.Parse(string(value))
		return err

	case "uri-template":
		// A URI Template (of any level) according to RFC6570. If you don’t already know what a URI Template is, you probably don’t need this value.
		_, err := url.Parse(string(value))
		if err != nil {
			return err
		}
		// Brackets must match
		if !reCurlyBracketsMatch.MatchString(string(value)) {
			return errors.New("brackets doesn't match")
		}
		if strings.Count(string(value), "{") != strings.Count(string(value), "}") {
			return errors.New("brackets doesn't match")
		}
		return err

	case "json-pointer":
		// A JSON Pointer, according to RFC6901.
		if len(value) > 0 && string(value[:1]) != "/" {
			return errors.New("value should start with / or be empty")
		}
		if reNonEscapedJSONPointerChars.Match(value) {
			return errors.New("value contains unescaped ~")
		}
		pointer := url.QueryEscape(string(value))
		_, err := url.Parse(pointer)
		return err

	case "relative-json-pointer":
		// A relative JSON pointer.
		if len(value) > 0 && string(value[:1]) == "/" {
			return errors.New("value should not start with /")
		}
		pointer := url.QueryEscape(string(value))
		_, err := url.Parse(pointer)
		return err

	case "regex":
		// A regular expression, which should be valid according to the ECMA 262 dialect.
		_, err := regexp.Compile(string(value))
		return err

	default:
		return fmt.Errorf("unknown format: %s", *schema.Format)
	}
}
