package jsonschema

import (
	_ "embed"
	"fmt"
)

// TODO: Update with 20xx-xx props

var (
	//go:embed schemas/draft-04.json
	draft04Source []byte

	//go:embed schemas/draft-06.json
	draft06Source []byte

	//go:embed schemas/draft-07.json
	draft07Source []byte

	draft04Schema *Schema
	draft06Schema *Schema
	draft07Schema *Schema
)

func init() {
	if draft04Source == nil {
		panic("can't start without schemas/draft-04.json")
	}
	if draft06Source == nil {
		panic("can't start without schemas/draft-06.json")
	}
	if draft07Source == nil {
		panic("can't start without schemas/draft-07.json")
	}

	var err error

	draft04Schema, err = New(draft04Source)
	if err != nil {
		panic(fmt.Errorf("can't start without schemas/draft-04.json\n%s", err.Error()))
	}
	draft06Schema, err = New(draft06Source)
	if err != nil {
		panic(fmt.Errorf("can't start without schemas/draft-06.json\n%s", err.Error()))
	}
	draft07Schema, err = New(draft07Source)
	if err != nil {
		panic(fmt.Errorf("can't start without schemas/draft-07.json\n%s", err.Error()))
	}
}
