package settings

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"flow-verifier/internal/httpx"
	"flow-verifier/internal/settings"
)

const maxBodyBytes = 1 << 16 // 64 KiB — defensive cap; the patch shape is small

func getHandler(svc *settings.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got, err := svc.Get(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, got)
	}
}

func putHandler(svc *settings.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "request body too large or unreadable")
			return
		}
		var patch settings.Patch
		if len(body) > 0 {
			dec := json.NewDecoder(bytesReader(body))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&patch); err != nil {
				httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "malformed body: "+err.Error())
				return
			}
		}
		updated, err := svc.Upsert(r.Context(), patch)
		if err != nil {
			var ve settings.ValidationError
			if errors.As(err, &ve) {
				httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, ve.Error())
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// bytesReader keeps the json.Decoder's DisallowUnknownFields contract
// without pulling in bytes.NewReader at every call site.
type byteSliceReader struct {
	b   []byte
	off int
}

func bytesReader(b []byte) *byteSliceReader { return &byteSliceReader{b: b} }

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if r.off >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.off:])
	r.off += n
	return n, nil
}
