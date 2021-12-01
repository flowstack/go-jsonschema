package jsonschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/buger/jsonparser"
)

type Ref struct {
	String *string
	Schema *Schema `json:"-"`

	// This is needed for de-ref'ing
	parent *Schema

	// This is needed for marshalling
	marshalled int
}

func (r Ref) MarshalJSON() ([]byte, error) {
	if r.String != nil {
		b, err := json.Marshal(r.String)
		return b, err
	}

	return nil, nil
}

func NewRef(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Ref, error) {
	if parentSchema == nil {
		return nil, errors.New("unable to set ref on non-existing schema")
	}

	// Anything but a string is an error
	if vt != jsonparser.String {
		return nil, errors.New("unable to parse 1ref")
	}

	tmpVal := string(jsonVal)
	ref := &Ref{
		String: &tmpVal,
		parent: parentSchema,
	}

	parentSchema.setRef(ref)

	return ref, nil
}

type Type struct {
	String  *string
	Strings *[]*string
}

func (t *Type) Has(vt ValueType) bool {
	if t == nil {
		return false
	}
	if t.String != nil {
		if vt.String() == *t.String {
			return true
		}
	} else if t.Strings != nil {
		for _, str := range *t.Strings {
			if vt.String() == *str {
				return true
			}
		}
	}
	return false
}

func (t Type) MarshalJSON() ([]byte, error) {
	if t.String != nil {
		b, err := json.Marshal(t.String)
		return b, err
	} else if t.Strings != nil {
		b, err := json.Marshal(t.Strings)
		return b, err
	}

	return nil, nil
}

func NewType(jsonVal []byte, vt jsonparser.ValueType) (*Type, error) {
	if vt == jsonparser.Array {
		typ := &Type{Strings: &[]*string{}}
		var errs error

		jsonparser.ArrayEach(jsonVal, func(value []byte, dataType jsonparser.ValueType, offset int, parseErr error) {
			if parseErr != nil {
				errs = addError(parseErr, errs)
				return
			}

			tmpVal := string(value)

			*typ.Strings = append(*typ.Strings, &tmpVal)
		})

		return typ, errs

	} else {
		tmpVal := string(jsonVal)
		return &Type{String: &tmpVal}, nil
	}
}

type Items struct {
	Schema  *Schema
	Schemas *Schemas
	Boolean *bool
}

func (i Items) MarshalJSON() ([]byte, error) {
	if i.Schema != nil {
		b, err := json.Marshal(i.Schema)
		return b, err
	} else if i.Schemas != nil {
		b, err := json.Marshal(i.Schemas)
		return b, err
	} else if i.Boolean != nil {
		b, err := json.Marshal(i.Boolean)
		return b, err
	}

	return nil, nil
}

func NewItems(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Items, error) {
	var err error
	items := Items{}
	var errs error

	switch vt {
	case jsonparser.Object:
		items.Schema, err = parentSchema.Parse(jsonVal)
		if err != nil {
			errs = addError(err, errs)
		}

	case jsonparser.Array:
		items.Schemas, err = NewSubSchemas(jsonVal, vt, parentSchema)
		if err != nil {
			errs = addError(err, errs)
		}

	case jsonparser.Boolean:
		var tmpBool bool
		tmpBool, err = jsonparser.ParseBoolean(jsonVal)
		if err != nil {
			errs = addError(err, errs)
		}

		items.Boolean = &tmpBool

	default:
		errs = addError(fmt.Errorf("expected an object, got: %s", vt.String()), errs)
	}

	if errs != nil {
		return nil, errs
	}

	return &items, nil
}

type Dependency struct {
	Schema  *Schema
	Strings *Strings
}

func (i Dependency) MarshalJSON() ([]byte, error) {
	if i.Schema != nil {
		b, err := json.Marshal(i.Schema)
		return b, err
	} else if i.Strings != nil {
		b, err := json.Marshal(i.Strings)
		return b, err
	}

	return nil, nil
}

func NewDependency(jsonVal []byte, vt jsonparser.ValueType, parentSchema *Schema) (*Dependency, error) {
	var err error
	dependency := Dependency{}
	var errs error

	switch vt {
	case jsonparser.Boolean:
		fallthrough
	case jsonparser.Object:
		dependency.Schema, err = parentSchema.Parse(jsonVal)
		if err != nil {
			errs = addError(err, errs)
		}

	case jsonparser.Array:
		strings := Strings{}

		jsonparser.ArrayEach(jsonVal, func(value []byte, dataType jsonparser.ValueType, offset int, parseErr error) {
			if parseErr != nil {
				errs = addError(parseErr, errs)
				return
			}

			if dataType == jsonparser.String {
				strings = append(strings, NewStringPtr(value))
			} else {
				errs = addError(fmt.Errorf("expected a string, got: %s", dataType.String()), errs)
			}
		})

		dependency.Strings = &strings

	default:
		errs = addError(fmt.Errorf("expected an array or object, got: %s", vt.String()), errs)
	}

	if errs != nil {
		return nil, errs
	}

	return &dependency, nil
}

type Value struct {
	String  *string
	Number  *big.Float
	Boolean *bool
	Null    *bool
	Object  *map[string]interface{}
	Array   *[]interface{}

	raw       []byte
	valueType ValueType
}

func (v Value) MarshalJSON() ([]byte, error) {
	if v.String != nil {
		return json.Marshal(v.String)
	} else if v.Number != nil {
		return []byte(v.Number.Text('g', -1)), nil
	} else if v.Boolean != nil {
		return json.Marshal(v.Boolean)
	} else if v.Null != nil {
		return []byte("null"), nil
	} else if v.Object != nil {
		return json.Marshal(v.Object)
	} else if v.Array != nil {
		return json.Marshal(v.Array)
	}

	return nil, nil
}

func NewValue(jsonVal []byte, vt jsonparser.ValueType) (*Value, error) {
	var err error

	val := Value{valueType: ValueType(vt)}

	if vt == jsonparser.Object || vt == jsonparser.Array {
		buf := &bytes.Buffer{}
		json.Compact(buf, jsonVal)
		val.raw = buf.Bytes()
	} else {
		val.raw = jsonVal
	}

	switch vt {
	case jsonparser.String:
		var str string
		str, err = jsonparser.ParseString(jsonVal)
		val.String = &str
	case jsonparser.Number:
		var num *big.Float
		num, ok := new(big.Float).SetString(string(jsonVal))
		if !ok {
			err = errors.New("unable to parse value as number")
		}
		val.Number = num
	case jsonparser.Boolean:
		var boolean bool
		boolean, err = jsonparser.ParseBoolean(jsonVal)
		val.Boolean = &boolean
	case jsonparser.Null:
		null := true
		val.Null = &null
	case jsonparser.Object:
		val.raw = sortObject(val.raw)

		tmpObject := map[string]interface{}{}

		err := jsonparser.ObjectEach(jsonVal, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			var err error
			tmpObject[string(key)], err = NewValue(value, dataType)
			return err
		})

		if err != nil {
			return nil, err
		}

		val.Object = &tmpObject

	case jsonparser.Array:
		tmpArray := []interface{}{}
		var errs error

		jsonparser.ArrayEach(jsonVal, func(value []byte, dataType jsonparser.ValueType, offset int, parseErr error) {
			if parseErr != nil {
				errs = addError(parseErr, errs)
				return
			}

			tmpVal, err := NewValue(value, dataType)
			if err != nil {
				errs = addError(err, errs)
				return
			}

			tmpArray = append(tmpArray, tmpVal)
		})

		if errs != nil {
			return nil, errs
		}

		val.Array = &tmpArray

	default:
		return nil, fmt.Errorf("unexpexted type: %s", vt.String())
	}

	if err != nil {
		return nil, err
	}

	return &val, nil
}

func (v *Value) Equal(val *Value) bool {
	if v == nil && val == nil {
		return true
	}
	if v != nil && val == nil {
		return false
	}
	if v.String != nil && val.String != nil {
		return (*v.String == *val.String)
	}
	if v.Number != nil && val.Number != nil {
		return v.Number.Cmp(val.Number) == 0
	}
	if v.Boolean != nil && val.Boolean != nil {
		return (*v.Boolean == *val.Boolean)
	}
	if v.Null != nil && val.Null != nil {
		return (*v.Null == *val.Null)
	}
	if v.Object != nil && val.Object != nil {
		return reflect.DeepEqual(*v.Object, *val.Object)
	}
	if v.Array != nil && val.Array != nil {
		return reflect.DeepEqual(*v.Array, *val.Array)
	}

	return false
}
