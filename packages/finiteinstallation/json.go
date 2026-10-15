package finiteinstallation

import (
	"bytes"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"io"
)

// Reject ambiguous duplicated properties before typed decoding of protected data.
func exactJSON(raw []byte) error {
	if len(raw) == 0 || len(raw) > 262144 {
		return effortauthority.ErrRefused
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 32 {
			return effortauthority.ErrRefused
		}
		token, e := decoder.Token()
		if e != nil {
			return effortauthority.ErrRefused
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, e := decoder.Token()
				name, ok := key.(string)
				if e != nil || !ok || seen[name] {
					return effortauthority.ErrRefused
				}
				seen[name] = true
				if value(depth+1) != nil {
					return effortauthority.ErrRefused
				}
			}
			end, e := decoder.Token()
			if e != nil || end != json.Delim('}') {
				return effortauthority.ErrRefused
			}
		case '[':
			for decoder.More() {
				if value(depth+1) != nil {
					return effortauthority.ErrRefused
				}
			}
			end, e := decoder.Token()
			if e != nil || end != json.Delim(']') {
				return effortauthority.ErrRefused
			}
		default:
			return effortauthority.ErrRefused
		}
		return nil
	}
	if value(0) != nil {
		return effortauthority.ErrRefused
	}
	if _, e := decoder.Token(); e != io.EOF {
		return effortauthority.ErrRefused
	}
	return nil
}
