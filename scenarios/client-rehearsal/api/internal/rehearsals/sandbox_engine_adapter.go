package rehearsals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxSandboxOperationResponseBytes = 256 * 1024

type sandboxHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// SandboxEngineAdapter sends generic native service operations to the exact
// synthetic operation addresses returned by the attached Workspace Sandbox
// profile. Non-native HTTP operations continue through the configured
// in-process simulator.
type SandboxEngineAdapter struct {
	fallback Simulator
	receipt  WorkspaceSandboxProfileReceipt
	client   sandboxHTTPDoer
}

func NewSandboxEngineAdapter(fallback Simulator, receipt WorkspaceSandboxProfileReceipt, client sandboxHTTPDoer) (*SandboxEngineAdapter, error) {
	if fallback == nil {
		return nil, errors.New("Sandbox engine adapter requires a fallback simulator")
	}
	if receipt.Status != "running" || len(receipt.OperationAddresses) == 0 {
		return nil, errors.New("Sandbox engine adapter requires a running profile with operation addresses")
	}
	if client == nil {
		transport := &http.Transport{
			Proxy: nil, // Never send synthetic alias traffic through an environment proxy.
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}
		httpClient := &http.Client{Timeout: 10 * time.Second, Transport: transport}
		httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		client = httpClient
	}
	for _, operation := range receipt.OperationAddresses {
		if _, err := validatedSandboxOperationURL(operation); err != nil {
			return nil, err
		}
	}
	return &SandboxEngineAdapter{fallback: fallback, receipt: receipt, client: client}, nil
}

func (a *SandboxEngineAdapter) Invoke(ctx context.Context, step Step, attempt uint32, injectFault bool) (string, error) {
	if step.NodeType == "" {
		if step.Host == "" {
			return a.fallback.Invoke(ctx, step, attempt, injectFault)
		}
	}
	var selected *HTTPMockOperationAddress
	for i := range a.receipt.OperationAddresses {
		candidate := &a.receipt.OperationAddresses[i]
		selector := candidate.Selector
		matches := false
		if step.NodeType == "" {
			matches = genericHTTPRequestMatches(step, *candidate)
		} else {
			matches = nativeOperationKey(selector.NodeType, selector.Resource, selector.Operation) == nativeOperationKey(step.NodeType, step.Resource, step.Action) && inputMatches(selector.InputMatch, step.Input)
		}
		if matches {
			if selected != nil {
				return "", errors.New("Sandbox profile contains ambiguous native operation selectors")
			}
			selected = candidate
		}
	}
	if selected == nil {
		if step.NodeType == "" && step.Host != "" {
			return "", errors.New("no attached Sandbox companion route matches the synthetic HTTP request")
		}
		return "", fmt.Errorf("no attached Sandbox operation matches native service %s:%s.%s", step.NodeType, step.Resource, step.Action)
	}
	target, err := validatedSandboxOperationURL(*selected)
	if err != nil {
		return "", err
	}
	var body io.Reader
	if len(step.Input) > 0 {
		encoded, marshalErr := json.Marshal(step.Input)
		if marshalErr != nil || len(encoded) > maxSimulatorRequestBytes {
			return "", errors.New("native service request input is invalid or exceeds the request limit")
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, strings.ToUpper(selected.Method), target.String(), body)
	if err != nil {
		return "", fmt.Errorf("build Sandbox native service request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := a.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("Sandbox synthetic service request failed: %w", err)
	}
	defer response.Body.Close()
	limited, err := io.ReadAll(io.LimitReader(response.Body, maxSandboxOperationResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("read Sandbox synthetic service response: %w", err)
	}
	if len(limited) > maxSandboxOperationResponseBytes {
		return "", errors.New("Sandbox synthetic service response exceeds 256 KiB")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("synthetic HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(limited)))
	}
	return string(limited), nil
}

func genericHTTPRequestMatches(step Step, candidate HTTPMockOperationAddress) bool {
	target, err := validatedSandboxOperationURL(candidate)
	if err != nil || target.RawQuery != "" {
		return false
	}
	return strings.EqualFold(target.Hostname(), step.Host) && strings.EqualFold(target.Path, step.URL) && strings.EqualFold(candidate.Method, step.Method)
}

// rewriteWorkflowHTTPRequestsForSandbox rewrites static synthetic HTTP Request
// URLs to the actual port and aliases returned by the task-owned companion.
// n8n runs inside that private network; the API host must not attempt to
// resolve the Docker-only alias itself.
func rewriteWorkflowHTTPRequestsForSandbox(raw string, operations []HTTPMockOperationAddress) (string, error) {
	var document map[string]any
	if err := json.Unmarshal([]byte(raw), &document); err != nil || document == nil {
		return "", errors.New("native task workflow must be a JSON object")
	}
	if err := rewriteWorkflowDocumentHTTPURLs(document, operations); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", errors.New("encode task workflow with companion aliases")
	}
	return string(encoded), nil
}

func rewriteWorkflowDocumentHTTPURLs(document map[string]any, operations []HTTPMockOperationAddress) error {
	if workflow, ok := document["workflow"].(map[string]any); ok {
		if err := rewriteWorkflowDocumentHTTPURLs(workflow, operations); err != nil {
			return err
		}
	}
	if workflows, ok := document["workflows"].([]any); ok {
		for _, rawWorkflow := range workflows {
			workflow, ok := rawWorkflow.(map[string]any)
			if !ok {
				return errors.New("task workflow bundle contains a non-object workflow")
			}
			if err := rewriteWorkflowDocumentHTTPURLs(workflow, operations); err != nil {
				return err
			}
		}
	}
	nodes, ok := document["nodes"].([]any)
	if !ok {
		return nil
	}
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok || node["type"] != "n8n-nodes-base.httpRequest" {
			continue
		}
		parameters, ok := node["parameters"].(map[string]any)
		if !ok {
			return errors.New("native HTTP Request node has no parameter object")
		}
		text, ok := parameters["url"].(string)
		if !ok || strings.Contains(text, "{{") {
			return errors.New("native HTTP Request nodes require a static URL so companion routing can be verified")
		}
		original, err := url.Parse(text)
		if err != nil || (original.Scheme != "http" && original.Scheme != "https") || original.User != nil || original.Fragment != "" || original.Port() != "" || !strings.HasSuffix(strings.ToLower(original.Hostname()), ".synthetic.invalid") {
			return errors.New("native HTTP Request URL must use a bounded synthetic.invalid host without credentials or a port")
		}
		if original.RawQuery != "" {
			return errors.New("native HTTP Request query parameters need an exact companion selector before runtime import")
		}
		method, _ := parameters["method"].(string)
		if method == "" {
			method = http.MethodGet
		}
		method = strings.ToUpper(method)
		path := original.EscapedPath()
		if path == "" {
			path = "/"
		}
		var selected *url.URL
		for _, candidate := range operations {
			address, validateErr := validatedSandboxOperationURL(candidate)
			if validateErr != nil || address.RawQuery != "" || !strings.EqualFold(candidate.Method, method) || !strings.EqualFold(address.Hostname(), original.Hostname()) || address.EscapedPath() != path {
				continue
			}
			if selected != nil {
				return errors.New("native HTTP Request URL matches multiple companion routes")
			}
			selected = address
		}
		if selected == nil {
			return errors.New("native HTTP Request URL has no exact task companion route")
		}
		parameters["url"] = selected.String()
	}
	return nil
}

func validatedSandboxOperationURL(operation HTTPMockOperationAddress) (*url.URL, error) {
	parsed, err := url.Parse(operation.URL)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Fragment != "" || !strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".synthetic.invalid") || parsed.Port() != "18080" || !validSandboxProfilePath(parsed.Path) || !operationQueryMatches(parsed.RawQuery, operation.Selector.InputMatch) || !oneOfFold(operation.Method, http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions) {
		return nil, errors.New("Sandbox operation address is not a bounded synthetic companion alias")
	}
	return parsed, nil
}

func operationQueryMatches(raw string, match map[string]any) bool {
	values, err := url.ParseQuery(raw)
	if err != nil || len(values) != len(match) {
		return false
	}
	for key, expected := range match {
		items := values[key]
		if !sandboxProfileQueryKey.MatchString(key) || secretKey(key) || len(items) != 1 || items[0] != fmt.Sprint(expected) || len(items[0]) > 512 || strings.ContainsAny(items[0], "\x00\r\n") || sandboxProfileCredential.MatchString(items[0]) {
			return false
		}
	}
	return true
}
