// Package svcedit performs structured, format-preserving edits of a scenario's
// service.json. It is deliberately NOT a blind map round-trip: it preserves key
// order, unknown fields, HTML escaping, 2-space indentation, and the trailing
// newline so an auto-fix only changes the bytes it intends to change.
//
// It is built on github.com/iancoleman/orderedmap. After Parse, every nested
// object is normalized to a *orderedmap.OrderedMap so edits mutate in place, and
// EscapeHTML is forced on (the generated service.json files escape `&`/`<`/`>`),
// which makes an untouched document round-trip byte-for-byte identical.
package svcedit

import (
	"encoding/json"
	"fmt"

	"github.com/iancoleman/orderedmap"
)

// Doc is a parsed, in-place-editable service.json document.
type Doc struct {
	root *orderedmap.OrderedMap
}

// Parse decodes service.json bytes into an editable Doc.
func Parse(b []byte) (*Doc, error) {
	root := orderedmap.New()
	if err := json.Unmarshal(b, root); err != nil {
		return nil, fmt.Errorf("parse service.json: %w", err)
	}
	normalize(root)
	return &Doc{root: root}, nil
}

// Root returns the document root for navigation/editing.
func (d *Doc) Root() *orderedmap.OrderedMap { return d.root }

// Bytes serializes the document with the canonical 2-space indent + trailing
// newline (matching the generated service.json style).
func (d *Doc) Bytes() ([]byte, error) {
	out, err := json.MarshalIndent(d.root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal service.json: %w", err)
	}
	return append(out, '\n'), nil
}

// EnsureMap returns the nested object at key under parent, creating an empty
// ordered object (in document-append position) if it is absent or not an object.
func EnsureMap(parent *orderedmap.OrderedMap, key string) *orderedmap.OrderedMap {
	if m, ok := GetMap(parent, key); ok {
		return m
	}
	m := orderedmap.New()
	m.SetEscapeHTML(true)
	parent.Set(key, m)
	return m
}

// GetMap returns the nested object at key if present.
func GetMap(parent *orderedmap.OrderedMap, key string) (*orderedmap.OrderedMap, bool) {
	v, ok := parent.Get(key)
	if !ok {
		return nil, false
	}
	switch m := v.(type) {
	case *orderedmap.OrderedMap:
		return m, true
	case orderedmap.OrderedMap:
		return &m, true
	default:
		return nil, false
	}
}

// GetSlice returns the array at key if present.
func GetSlice(parent *orderedmap.OrderedMap, key string) ([]interface{}, bool) {
	v, ok := parent.Get(key)
	if !ok {
		return nil, false
	}
	s, ok := v.([]interface{})
	return s, ok
}

// AppendToSlice appends item to the array at key under parent, creating the
// array if it is absent.
func AppendToSlice(parent *orderedmap.OrderedMap, key string, item interface{}) {
	s, _ := GetSlice(parent, key)
	parent.Set(key, append(s, item))
}

// NewObject builds an ordered object from alternating key/value pairs. Keys must
// be strings; an odd-length or non-string-key call panics (programmer error).
func NewObject(pairs ...interface{}) *orderedmap.OrderedMap {
	if len(pairs)%2 != 0 {
		panic("svcedit.NewObject: odd number of arguments")
	}
	m := orderedmap.New()
	m.SetEscapeHTML(true)
	for i := 0; i < len(pairs); i += 2 {
		k, ok := pairs[i].(string)
		if !ok {
			panic("svcedit.NewObject: non-string key")
		}
		m.Set(k, pairs[i+1])
	}
	return m
}

// normalize recursively converts nested OrderedMap values to pointers (so edits
// mutate in place) and forces EscapeHTML on at every level.
func normalize(om *orderedmap.OrderedMap) {
	om.SetEscapeHTML(true)
	for _, k := range om.Keys() {
		v, _ := om.Get(k)
		switch t := v.(type) {
		case orderedmap.OrderedMap:
			p := &t
			normalize(p)
			om.Set(k, p)
		case *orderedmap.OrderedMap:
			normalize(t)
		case []interface{}:
			normalizeSlice(t)
		}
	}
}

func normalizeSlice(s []interface{}) {
	for i, v := range s {
		switch t := v.(type) {
		case orderedmap.OrderedMap:
			p := &t
			normalize(p)
			s[i] = p
		case *orderedmap.OrderedMap:
			normalize(t)
		case []interface{}:
			normalizeSlice(t)
		}
	}
}
