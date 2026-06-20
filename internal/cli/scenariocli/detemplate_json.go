package scenariocli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Order-preserving JSON pruning for hand-authored files that cannot carry
// comment markers (i18n locales, the CLI manifest). The standard library has
// no order-preserving DOM, so this builds a minimal one: objects become
// *jsonObject (ordered key list + value map), arrays become []any, and scalars
// are kept as json.RawMessage so their exact literal (including non-ASCII
// UTF-8 like ノート / الملاحظات) round-trips byte-for-byte. Re-marshaling uses
// two-space indentation and disables HTML escaping to match the template's
// hand-authored locale style.

type jsonObject struct {
	keys []string
	vals map[string]any
}

func newJSONObject() *jsonObject {
	return &jsonObject{vals: map[string]any{}}
}

func (o *jsonObject) set(key string, v any) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
}

func (o *jsonObject) delete(key string) bool {
	if _, ok := o.vals[key]; !ok {
		return false
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
	return true
}

// decodeOrderedJSON parses data into the ordered DOM.
func decodeOrderedJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeOrderedValue(dec)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func decodeOrderedValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeFromToken(dec, tok)
}

func decodeFromToken(dec *json.Decoder, tok json.Token) (any, error) {
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			obj := newJSONObject()
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("expected object key, got %T", keyTok)
				}
				val, err := decodeOrderedValue(dec)
				if err != nil {
					return nil, err
				}
				obj.set(key, val)
			}
			if _, err := dec.Token(); err != nil { // consume '}'
				return nil, err
			}
			return obj, nil
		case '[':
			arr := []any{}
			for dec.More() {
				val, err := decodeOrderedValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil { // consume ']'
				return nil, err
			}
			return arr, nil
		default:
			return nil, fmt.Errorf("unexpected delimiter %q", t)
		}
	default:
		return scalarToken{tok}, nil
	}
}

// scalarToken wraps a leaf token (string/number/bool/null) so it round-trips.
type scalarToken struct{ v any }

// encodeOrderedJSON marshals the ordered DOM with two-space indentation and no
// HTML escaping, appending a trailing newline.
func encodeOrderedJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeOrderedValue(&buf, v, ""); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func encodeScalar(buf *bytes.Buffer, v any) error {
	// json.Encoder writes a trailing newline; trim it. SetEscapeHTML(false)
	// keeps <, >, & and non-ASCII UTF-8 literal.
	var tmp bytes.Buffer
	enc := json.NewEncoder(&tmp)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	buf.Write(bytes.TrimRight(tmp.Bytes(), "\n"))
	return nil
}

func encodeOrderedValue(buf *bytes.Buffer, v any, indent string) error {
	switch t := v.(type) {
	case *jsonObject:
		if len(t.keys) == 0 {
			buf.WriteString("{}")
			return nil
		}
		buf.WriteString("{\n")
		inner := indent + "  "
		for i, k := range t.keys {
			buf.WriteString(inner)
			if err := encodeScalar(buf, k); err != nil {
				return err
			}
			buf.WriteString(": ")
			if err := encodeOrderedValue(buf, t.vals[k], inner); err != nil {
				return err
			}
			if i < len(t.keys)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(indent + "}")
		return nil
	case []any:
		if len(t) == 0 {
			buf.WriteString("[]")
			return nil
		}
		buf.WriteString("[\n")
		inner := indent + "  "
		for i, el := range t {
			buf.WriteString(inner)
			if err := encodeOrderedValue(buf, el, inner); err != nil {
				return err
			}
			if i < len(t)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(indent + "]")
		return nil
	case scalarToken:
		return encodeScalar(buf, t.v)
	default:
		return encodeScalar(buf, v)
	}
}

// PruneJSON removes the given object key-paths and array-element matches from
// JSON data, preserving key order and UTF-8 content. It returns the rewritten
// bytes and the number of deletions (deleted keys + removed array elements).
func PruneJSON(data []byte, keys []string, matches []TemplateJSONArrayMatch) ([]byte, int, error) {
	root, err := decodeOrderedJSON(data)
	if err != nil {
		return nil, 0, fmt.Errorf("parse JSON: %w", err)
	}
	removed := 0
	for _, key := range keys {
		if deleteJSONPath(root, strings.Split(key, ".")) {
			removed++
		}
	}
	for _, m := range matches {
		removed += deleteJSONArrayMatch(root, strings.Split(m.Path, "."), m.Where)
	}
	if removed == 0 {
		return data, 0, nil
	}
	out, err := encodeOrderedJSON(root)
	if err != nil {
		return nil, 0, fmt.Errorf("re-encode JSON: %w", err)
	}
	return out, removed, nil
}

// deleteJSONPath navigates a dotted path and deletes the final key. Returns
// true if a deletion occurred.
func deleteJSONPath(root any, path []string) bool {
	if len(path) == 0 {
		return false
	}
	cur := root
	for _, seg := range path[:len(path)-1] {
		obj, ok := cur.(*jsonObject)
		if !ok {
			return false
		}
		next, ok := obj.vals[seg]
		if !ok {
			return false
		}
		cur = next
	}
	obj, ok := cur.(*jsonObject)
	if !ok {
		return false
	}
	return obj.delete(path[len(path)-1])
}

// deleteJSONArrayMatch navigates to an array and removes every object element
// whose fields all equal the Where map. Returns the number of elements removed.
func deleteJSONArrayMatch(root any, path []string, where map[string]string) int {
	if len(path) == 0 {
		return 0
	}
	parentPath := path[:len(path)-1]
	arrKey := path[len(path)-1]
	pNode := root
	for _, seg := range parentPath {
		obj, ok := pNode.(*jsonObject)
		if !ok {
			return 0
		}
		next, ok := obj.vals[seg]
		if !ok {
			return 0
		}
		pNode = next
	}
	parent, ok := pNode.(*jsonObject)
	if !ok {
		return 0
	}
	arr, ok := parent.vals[arrKey].([]any)
	if !ok {
		return 0
	}
	kept := make([]any, 0, len(arr))
	removed := 0
	for _, el := range arr {
		if elementMatches(el, where) {
			removed++
			continue
		}
		kept = append(kept, el)
	}
	if removed > 0 {
		parent.set(arrKey, kept)
	}
	return removed
}

func elementMatches(el any, where map[string]string) bool {
	if len(where) == 0 {
		return false
	}
	obj, ok := el.(*jsonObject)
	if !ok {
		return false
	}
	for field, want := range where {
		v, ok := obj.vals[field]
		if !ok {
			return false
		}
		sv, ok := v.(scalarToken)
		if !ok {
			return false
		}
		s, ok := sv.v.(string)
		if !ok || s != want {
			return false
		}
	}
	return true
}
