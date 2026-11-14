// Package appctx provides shared context for CLI commands.
package appctx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliapp"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
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

	// DeleteWithQuery performs a DELETE request with query parameters and
	// decodes the response body into result (if non-nil).
	DeleteWithQuery(path string, query url.Values, result interface{}) error
}

// Runtime adapts cli-core's ScenarioApp to the prompt-manager domain command
// interface so commands can share one standard request substrate.
type Runtime struct {
	Core *cliapp.ScenarioApp
}

// HasDurableProduct reports whether the member's run has a recent Source
// Ledger journal receipt. It is deliberately an optional Context capability so
// command tests and isolated API fakes do not need a live Source Ledger.
func (r Runtime) HasDurableProduct(teamID, runID string, cutoff time.Time) (bool, error) {
	base, err := discovery.ResolveScenarioURLDefault(context.Background(), "source-ledger")
	if err != nil {
		return false, err
	}
	client := journalconnect.NewJournalServiceClient(http.DefaultClient, strings.TrimRight(base, "/"))
	response, err := client.ListEntries(context.Background(), connect.NewRequest(&journalv1.ListEntriesRequest{
		Scope: "team:" + strings.TrimPrefix(strings.TrimSpace(teamID), "team:"), Limit: 500,
	}))
	if err != nil {
		return false, err
	}
	for _, entry := range response.Msg.GetEntries() {
		if entry.GetCreatedAt() == nil || entry.GetCreatedAt().AsTime().Before(cutoff) {
			continue
		}
		if strings.TrimSpace(entry.GetCorrelation().GetRunId()) == strings.TrimSpace(runID) {
			return true, nil
		}
	}
	return false, nil
}

var _ Context = Runtime{}

// Get performs a GET request and unmarshals the response into result.
func (r Runtime) Get(path string, result interface{}) error {
	return r.GetWithQuery(path, nil, result)
}

// GetWithQuery performs a GET request with query parameters.
func (r Runtime) GetWithQuery(path string, query url.Values, result interface{}) error {
	if body, handled, err := r.connectRequest("GET", path, query, nil); handled {
		if err != nil {
			return err
		}
		return decode(body, result)
	}
	body, err := r.Core.Get(path, query)
	if err != nil {
		return err
	}
	return decode(body, result)
}

// GetRawWithQuery performs a GET request with query parameters and returns the
// raw response body.
func (r Runtime) GetRawWithQuery(path string, query url.Values) ([]byte, error) {
	if body, handled, err := r.connectRequest("GET", path, query, nil); handled {
		return body, err
	}
	return r.Core.Get(path, query)
}

// Post performs a POST request with the given payload.
func (r Runtime) Post(path string, payload interface{}, result interface{}) error {
	if body, handled, err := r.connectRequest("POST", path, nil, payload); handled {
		if err != nil {
			return err
		}
		return decode(body, result)
	}
	body, err := r.Core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	return decode(body, result)
}

// Put performs a PUT request with the given payload.
func (r Runtime) Put(path string, payload interface{}, result interface{}) error {
	if body, handled, err := r.connectRequest("PUT", path, nil, payload); handled {
		if err != nil {
			return err
		}
		return decode(body, result)
	}
	body, err := r.Core.Request("PUT", path, nil, payload)
	if err != nil {
		return err
	}
	return decode(body, result)
}

// Delete performs a DELETE request.
func (r Runtime) Delete(path string) error {
	if _, handled, err := r.connectRequest("DELETE", path, nil, nil); handled {
		return err
	}
	_, err := r.Core.Request("DELETE", path, nil, nil)
	return err
}

// DeleteWithQuery performs a DELETE request with query parameters and
// decodes the response body into result (if non-nil).
func (r Runtime) DeleteWithQuery(path string, query url.Values, result interface{}) error {
	if body, handled, err := r.connectRequest("DELETE", path, query, nil); handled {
		if err != nil {
			return err
		}
		return decode(body, result)
	}
	body, err := r.Core.Request("DELETE", path, query, nil)
	if err != nil {
		return err
	}
	return decode(body, result)
}

func decode(body []byte, result interface{}) error {
	if result == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, result)
}
