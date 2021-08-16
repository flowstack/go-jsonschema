package jsonschema

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
)

// Relevant docs:
// https://json-schema.org/understanding-json-schema/structuring.html

// Anything starting with # means:
// Go to base schema -> find whatever is after #

func unescapeRefPath(refPath string) string {
	var err error
	var ref, frag string

	refParts := strings.Split(refPath, "#")
	if len(refParts) == 1 {
		frag = refParts[0]
	} else if len(refParts) > 1 {
		ref = refParts[0]
		frag = strings.Join(refParts[1:], "#")
	}

	frag, err = url.QueryUnescape(frag)
	if err != nil {
		return ""
	}
	frag = strings.ReplaceAll(frag, "~0", "~")
	frag = strings.ReplaceAll(frag, "~1", "/")

	if ref != "" && frag != "" {
		return ref + "#" + frag
	}

	return ref + frag
}

// ExpandURI attempts to resolve a uri against the current Base URI
func (s *Schema) ExpandURI(uri string) (*url.URL, error) {
	// If uri is empty, it is seen as invalid
	if len(uri) == 0 {
		return nil, errors.New("URI is invalid")
	}

	// If uri starts with #, it should not be expanded further
	if len(uri) > 0 && uri[:1] == "#" {
		return url.Parse(uri)
	}

	// Parse uri
	uriParsed, err := url.Parse(uri)
	if err != nil {
		return nil, errors.New("URI is invalid")
	}

	// If uri starts with http or https it is already expanded as much as possible
	if uriParsed.Scheme == "http" || uriParsed.Scheme == "https" {
		return uriParsed, nil
	}

	// Try to get the current Base URI
	var curBaseURI *url.URL
	if s != nil {
		if s.baseURI != nil {
			curBaseURI = s.baseURI
		} else if s.base != nil && s.base.baseURI != nil {
			curBaseURI = s.base.baseURI
		}
	}

	// If both Base URI and uri is valid, we'll merge and return the result
	if curBaseURI != nil {
		path := uriParsed.Path
		if uriParsed.Fragment != "" {
			path += "#" + uriParsed.Fragment
		}
		return curBaseURI.Parse(path)
	}

	// If Base URI is not found, uri is returned as is
	return uriParsed, nil
}

func (s *Schema) DeRef() error {
	var err error

	if s == nil {
		return nil
	}

	root := s
	if s.root != nil {
		root = s.root
	}

	if root.refs != nil {
		for _, ref := range *root.refs {
			if ref.Schema == nil {
				ref.Schema, err = ref.parent.ResolveRef(ref)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *Schema) ResolveRef(ref *Ref) (*Schema, error) {
	// If Schema is set, it's a cached version
	if ref.Schema != nil {
		return ref.Schema, nil
	}

	// If String is nil or empty, the $ref is invalid
	if ref.String == nil || len(*ref.String) == 0 {
		return nil, errors.New("$ref is invalid")
	}

	refStr := *ref.String

	baseSchema := s
	if s.baseURI == nil && s.base != nil {
		baseSchema = s.base
	}

	// Check for base schema reference
	if refStr[:1] == "#" {
		// Check if this is a base schema reference with no path to a sub schema
		if refStr == "#" {
			return baseSchema, nil
		}

		if len(refStr) > 1 && refStr[:2] != "#/" {
			refSchema := baseSchema.getPointer(refStr)
			if refSchema != nil {
				return refSchema, nil
			}

			return nil, fmt.Errorf("unable to find ref: %s", refStr)
		}

		// Find the path
		pathParts := strings.Split(strings.Trim(refStr[1:], "/"), "/")
		if len(pathParts) == 0 {
			return baseSchema, nil
		}

		for i, v := range pathParts {
			if strings.Contains("0123456789", v) {
				pathParts[i] = fmt.Sprintf("[%s]", v)
			} else {
				pathParts[i] = unescapeRefPath(pathParts[i])
			}

			rawBase, _, _, err := jsonparser.Get(baseSchema.raw, pathParts[i])
			if err != nil {
				return nil, fmt.Errorf("unable to find schema at path: %s", *ref.String)
			}

			baseSchema, err = baseSchema.Parse(rawBase)
			if err != nil {
				return nil, fmt.Errorf("unable to find schema at path: %s", *ref.String)
			}
		}

		return baseSchema, nil

	} else {
		refURI, err := baseSchema.ExpandURI(refStr)
		if err != nil {
			return nil, err
		}

		switch refURI.String() {
		case "http://json-schema.org/draft-04/schema":
			baseSchema = Draft04Schema
		case "http://json-schema.org/draft-05/schema":
			baseSchema = Draft04Schema
		case "http://json-schema.org/schema":
			baseSchema = Draft04Schema
		case "http://json-schema.org/draft-06/schema":
			baseSchema = Draft06Schema
		case "http://json-schema.org/draft-07/schema":
			baseSchema = Draft07Schema
		default:
			if s != nil {
				frag := refURI.Fragment
				refURI.Fragment = ""
				baseSchema = s.getPointer(refURI.String())
				if baseSchema != nil && frag != "" {
					frag = "#" + frag
					return baseSchema.ResolveRef(&Ref{String: &frag})
				}
				refURI.Fragment = frag
			}
		}

		// Fetch the schema
		if baseSchema == nil {
			if err != nil {
				return nil, addError(errors.New("ref contains invalid URL"), err)
			}

			// TODO: We should probably accept query params
			schemaURL := fmt.Sprintf(
				"%s://%s%s%s",
				refURI.Scheme,
				refURI.User.String(),
				refURI.Host,
				refURI.Path,
			)

			res, err := http.Get(schemaURL)
			if err != nil {
				return nil, err
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return nil, err
			}
			res.Body.Close()

			// Add the $id to the schema, if it does not exist
			idKey := "$id"
			if s != nil && s.root != nil {
				if s.root.IsDraft4() {
					idKey = "id"
				}
			}

			_, _, _, err = jsonparser.Get(body, idKey)
			if err != nil {
				id := []byte(`"` + strings.ReplaceAll(schemaURL, `"`, `\"`) + `"`)
				body, err = jsonparser.Set(body, id, idKey)
				if err != nil {
					return nil, err
				}
			}

			baseSchema, err := baseSchema.Parse(body)
			if err != nil {
				return nil, err
			}
			refURI.Fragment = ""
			baseSchema.baseURI = refURI
			// log.Println(baseSchema)
			s.setPointer(refURI.String(), baseSchema)

			return baseSchema, nil
		}

		if baseSchema == nil {
			return nil, errors.New("unable to fetch the specified schema: " + refStr)
		}

		if refURI.Fragment != "" {
			fragment := fmt.Sprintf("#%s", refURI.Fragment)
			return baseSchema.ResolveRef(&Ref{String: &fragment})
		}

		return baseSchema, err
	}
}
