package jsonschema

import (
	"errors"
	"fmt"
	"strings"
)

func addError(err, errs error) error {
	if err == nil {
		return errs
	}
	if errs == nil {
		return err
	}

	return fmt.Errorf("%w\n%w", errs, err.Error())
}

type validationError struct {
	err        error
	originPath []string
}

func (e *validationError) addToPath(segment string) {
	e.originPath = append([]string{segment}, e.originPath...)
}

func (e *validationError) Error() string {
	pathParts := append([]string{"@"}, e.originPath...)
	return fmt.Sprintf("%v at %v", e.err, strings.Join(pathParts, "."))
}

func (e *validationError) Unwrap() error {
	return e.err
}

func addOriginPath(e error, name string) error {
	if e == nil {
		return nil
	}
	var ve *validationError
	if errors.As(e, &ve) {
		ve.addToPath(name)
		return e
	} else {
		ve = &validationError{
			err: e,
		}
		ve.addToPath(name)
		return ve
	}
}

func addOriginIndex(e error, idx int) error {
	return addOriginPath(e, fmt.Sprintf("%v", idx))
}
