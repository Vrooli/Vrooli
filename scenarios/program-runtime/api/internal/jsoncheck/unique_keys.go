// Package jsoncheck validates JSON properties that encoding/json otherwise
// accepts ambiguously.
package jsoncheck

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// UniqueKeys requires exactly one JSON value and rejects duplicate object keys
// at every depth. Numbers stay as json.Number so validation never rounds them.
func UniqueKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var walk func() error
	walk = func() error {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]struct{}{}
			for dec.More() {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return errors.New("invalid JSON object key")
				}
				if _, duplicate := seen[name]; duplicate {
					return errors.New("duplicate JSON object key")
				}
				seen[name] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
		case '[':
			for dec.More() {
				if err := walk(); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid JSON delimiter")
		}
		_, err = dec.Token()
		return err
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("multiple JSON values")
	}
	return nil
}
