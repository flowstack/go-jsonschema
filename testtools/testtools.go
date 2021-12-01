package testtools

import (
	"bytes"
	"encoding/json"
)

func CompareJSON(j1, j2 []byte) (bool, error) {
	var err error

	j1, err = SortAndCompactJSON(j1)
	if err != nil {
		return false, err
	}

	j2, err = SortAndCompactJSON(j2)
	if err != nil {
		return false, err
	}

	return (string(j1) == string(j2)), nil
}

func SortAndCompactJSON(j []byte) ([]byte, error) {
	// Unmarshalling to an interface and marshalling back, will cause the fields to be sorted
	var err error

	var tmp interface{}
	if err = json.Unmarshal(j, &tmp); err != nil {
		return nil, err
	}

	j, err = json.Marshal(tmp)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	json.Compact(&out, j)

	return out.Bytes(), nil
}
