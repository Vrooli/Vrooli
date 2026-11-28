package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/types/known/structpb"
)

func (a *App) requestMultipart(method, path string, payload []byte, contentType string) ([]byte, error) {
	base := strings.TrimRight(strings.TrimSpace(a.core.APIRootBase()), "/")
	if base == "" {
		return nil, fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	endpoint := base + a.core.APIPath(path)

	req, err := http.NewRequest(method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range a.core.APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}
	if a.core.HTTPClient != nil {
		a.core.HTTPClient.ApplyRequestHeaders(req)
	}

	timeout := a.core.HTTPClient.Timeout()
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, cliutil.ParseAPIError(resp.StatusCode, body)
	}
	return body, nil
}

func printJSONIfRequested(enabled bool, body []byte) bool {
	if !enabled {
		return false
	}
	cliutil.PrintJSON(body)
	return true
}

func decodeResponse[T any](body []byte) (T, error) {
	var response T
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to parse response: %w", err)
	}
	return response, nil
}

func decodeJSONStrict[T any](body []byte, target *T) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid trailing JSON content")
		}
		return err
	}

	return nil
}

func requireFlag(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--%s is required", name)
	}
	return nil
}

func requireFlags(pairs ...string) error {
	if len(pairs)%2 != 0 {
		return fmt.Errorf("requireFlags: uneven pairs")
	}
	for i := 0; i < len(pairs); i += 2 {
		if err := requireFlag(pairs[i], pairs[i+1]); err != nil {
			return err
		}
	}
	return nil
}

// injectJSONField adds a key-value pair to a JSON object if the key is not
// already present. Returns the original payload on any error.
func injectJSONField(payload json.RawMessage, key, value string) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return payload
	}
	if _, exists := m[key]; exists {
		return payload
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return payload
	}
	m[key] = encoded
	result, err := json.Marshal(m)
	if err != nil {
		return payload
	}
	return result
}

func parseJSONString(raw string) (json.RawMessage, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("JSON payload is required")
	}

	if strings.HasPrefix(raw, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if path == "" {
			return nil, fmt.Errorf("JSON file path is required after @")
		}
		content, err := cliutil.ReadFileString(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		raw = strings.TrimSpace(content)
	}

	var parsed json.RawMessage
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return json.RawMessage(raw), nil
}

func parseProtoStructJSON(raw string) (*structpb.Struct, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var value map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("inputs must be a JSON object: %w", err)
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("normalize inputs: %w", err)
	}
	var protoValue map[string]any
	if err := json.Unmarshal(normalized, &protoValue); err != nil {
		return nil, fmt.Errorf("normalize inputs: %w", err)
	}
	result, err := structpb.NewStruct(protoValue)
	if err != nil {
		return nil, fmt.Errorf("inputs must contain JSON values: %w", err)
	}
	return result, nil
}

func printTree[T any](items []T, childFn func(T) []T, lineFn func(T) string, level int) {
	for _, item := range items {
		fmt.Printf("%s%s\n", strings.Repeat("  ", level), lineFn(item))
		children := childFn(item)
		if len(children) > 0 {
			printTree(children, childFn, lineFn, level+1)
		}
	}
}

func cliCommand(parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	segments = append(segments, appName)
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		segments = append(segments, trimmed)
	}
	return strings.Join(segments, " ")
}

func proposalTargetPayload(value, name string) (map[string]any, error) {
	targetType, targetRef, ok := strings.Cut(strings.TrimSpace(value), "/")
	if !ok || strings.TrimSpace(targetType) == "" || strings.TrimSpace(targetRef) == "" {
		return nil, fmt.Errorf("target %q must use TYPE/REF", value)
	}
	if strings.TrimSpace(name) == "" {
		name = targetRef
	}
	return map[string]any{"type": strings.TrimSpace(targetType), "ref": strings.TrimSpace(targetRef), "name": strings.TrimSpace(name)}, nil
}

func parseSessionEntities(values []string) ([]cliAgentSessionContextRef, error) {
	refs := make([]cliAgentSessionContextRef, 0, len(values))
	for _, raw := range values {
		contextType, ref, ok := strings.Cut(strings.TrimSpace(raw), "/")
		if !ok || strings.TrimSpace(contextType) == "" || strings.TrimSpace(ref) == "" {
			return nil, fmt.Errorf("entity %q must use TYPE/REF", raw)
		}
		refs = append(refs, cliAgentSessionContextRef{Type: strings.TrimSpace(contextType), Ref: strings.TrimSpace(ref)})
	}
	return refs, nil
}

func printSection(title string) {
	fmt.Printf("%s:\n", title)
}

// stringSlice implements flag.Value for repeatable string flags.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }

func (s *stringSlice) Set(val string) error {
	*s = append(*s, val)
	return nil
}

func printCommandListSection(title string, commands []string) {
	filtered := make([]string, 0, len(commands))
	for _, command := range commands {
		if strings.TrimSpace(command) == "" {
			continue
		}
		filtered = append(filtered, command)
	}
	if len(filtered) == 0 {
		return
	}
	fmt.Printf("\n%s:\n", title)
	for _, command := range filtered {
		fmt.Printf("  %s\n", command)
	}
}
