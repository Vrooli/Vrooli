package mocks

import (
	"errors"
	"net/http"
)

type ErrorWriter struct {
	header   http.Header
	Statuses []int
}

func (e *ErrorWriter) Header() http.Header {
	if e.header == nil {
		e.header = make(http.Header)
	}
	return e.header
}

func (e *ErrorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func (e *ErrorWriter) WriteHeader(statusCode int) {
	e.Statuses = append(e.Statuses, statusCode)
}

func (e *ErrorWriter) HasStatus(code int) bool {
	for _, status := range e.Statuses {
		if status == code {
			return true
		}
	}
	return false
}
