package httpjson

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const MaxBodyBytes int64 = 1 << 20

func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	return decode(w, r, dst, false)
}

func DecodeAllowEmpty(w http.ResponseWriter, r *http.Request, dst any) error {
	return decode(w, r, dst, true)
}

func decode(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) error {
	reader := http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if allowEmpty && errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}
