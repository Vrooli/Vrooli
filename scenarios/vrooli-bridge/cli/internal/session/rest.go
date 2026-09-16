package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
)

// RESTRequest calls one of Bridge's owner-only REST endpoints (readiness,
// follow) through the same owner-session transport as the Connect commands.
// cli-core's own REST client sends only the legacy bearer token, which local
// enrollment clears, so those endpoints answered 401 to an enrolled operator.
func RESTRequest(app *cliapp.ScenarioApp, method, path string, query url.Values, body any) ([]byte, error) {
	client, baseURL := NewConnectHTTPClient(app)
	return restRequest(client, baseURL, app.APIPath(path), method, query, body)
}

func restRequest(client connect.HTTPClient, baseURL, apiPath, method string, query url.Values, body any) ([]byte, error) {
	target := strings.TrimRight(baseURL, "/") + apiPath
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, target, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("api error (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}
