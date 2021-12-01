package testtools

import "testing"

func TestCompareJSON(t *testing.T) {
	j1 := []byte(`{ "a": "a", "b": "b" }`)
	j2 := []byte(`{ "b": "b", "a": "a" }`)

	eq, err := CompareJSON(j1, j2)
	if err != nil {
		t.Fatal(err)
	}

	if !eq {
		t.Fatal("expected j1 and j2 to be equal")
	}
}
