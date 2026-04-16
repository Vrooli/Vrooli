// Package appctx provides shared context for CLI commands.
package appctx

import (
	"encoding/json"
	"net/url"

	"github.com/vrooli/cli-core/cliapp"
)

// Context provides methods that commands can use to interact with the API.
type Context interface {
	// Get performs a GET request and unmarshals the response into result.
	Get(path string, result interface{}) error

	// GetWithQuery performs a GET request with query parameters.
	GetWithQuery(path string, query url.Values, result interface{}) error

	// Post performs a POST request with the given payload.
	Post(path string, payload interface{}, result interface{}) error

	// Put performs a PUT request with the given payload.
	Put(path string, payload interface{}, result interface{}) error

	// Delete performs a DELETE request.
	Delete(path string) error
}

// Runtime adapts cli-core's ScenarioApp to the prompt-manager domain command
// interface so commands can share one standard request substrate.
type Runtime struct {
	Core *cliapp.ScenarioApp
}

var _ Context = Runtime{}

// Get performs a GET request and unmarshals the response into result.
func (r Runtime) Get(path string, result interface{}) error {
	return r.GetWithQuery(path, nil, result)
}

// GetWithQuery performs a GET request with query parameters.
func (r Runtime) GetWithQuery(path string, query url.Values, result interface{}) error {
	body, err := r.Core.Get(path, query)
	if err != nil {
		return err
	}
	return decode(body, result)
}

// Post performs a POST request with the given payload.
func (r Runtime) Post(path string, payload interface{}, result interface{}) error {
	body, err := r.Core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	return decode(body, result)
}

// Put performs a PUT request with the given payload.
func (r Runtime) Put(path string, payload interface{}, result interface{}) error {
	body, err := r.Core.Request("PUT", path, nil, payload)
	if err != nil {
		return err
	}
	return decode(body, result)
}

// Delete performs a DELETE request.
func (r Runtime) Delete(path string) error {
	_, err := r.Core.Request("DELETE", path, nil, nil)
	return err
}

func decode(body []byte, result interface{}) error {
	if result == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, result)
}
