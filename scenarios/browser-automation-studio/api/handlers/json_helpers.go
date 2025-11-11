package handlers

import (
	"net/http"

	"github.com/vrooli/browser-automation-studio/internal/httpjson"
)

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	return httpjson.Decode(w, r, dst)
}

func decodeJSONBodyAllowEmpty(w http.ResponseWriter, r *http.Request, dst any) error {
	return httpjson.DecodeAllowEmpty(w, r, dst)
}
