package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const maxSimulatorRequestBytes = 512 * 1024
const maxSimulatorSourceBytes = 2 * 1024 * 1024

const defaultSimulatorSource = `{"kind":"openapi-catalog","services":[{"id":"crm","document":{"openapi":"3.0.3","info":{"title":"Synthetic CRM","version":"1"},"servers":[{"url":"https://crm.synthetic.invalid"}],"paths":{"/v1/contacts":{"post":{"responses":{"200":{"content":{"application/json":{"example":{"id":"contact_synthetic_001","status":"upserted"}}}}}}}}}},{"id":"sheets","document":{"openapi":"3.0.3","info":{"title":"Synthetic Sheets","version":"1"},"servers":[{"url":"https://sheets.synthetic.invalid"}],"paths":{"/v1/spreadsheets/{sheetId}/rows":{"post":{"responses":{"200":{"content":{"application/json":{"example":{"updatedRange":"A2:B2","status":"appended"}}}}}}}}}}]}`

func simulatorSourceKind(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "synthetic-default"
	}
	var source struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal([]byte(raw), &source)
	if source.Kind == "" {
		return "openapi"
	}
	return source.Kind
}

func simulatorSourceSHA256(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		raw = defaultSimulatorSource
	}
	simulator, err := NewDescriptionSimulator(raw, false)
	if err != nil {
		return "", err
	}
	canonical, err := json.Marshal(simulator.sourceFingerprint())
	if err != nil {
		return "", fmt.Errorf("encode simulator source fingerprint: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

type DescriptionSimulator struct {
	kind         string
	services     []apiDescription
	exchanges    map[string]map[string]any
	faultPending bool
	faultMu      sync.Mutex
}

type apiDescription struct {
	id               string
	hosts            []string
	basePaths        map[string]string
	paths            map[string]map[string]any
	nativeOperations map[string]Step
}

// NewDescriptionSimulator builds a simulator from inert OpenAPI descriptions
// or recorded exchanges. No URL from workflow or profile data is opened.
func NewDescriptionSimulator(raw string, injectFault bool) (*DescriptionSimulator, error) {
	if strings.TrimSpace(raw) == "" {
		raw = defaultSimulatorSource
	}
	if len(raw) > maxSimulatorSourceBytes {
		return nil, ErrInvalid{Field: "simulator_source_json", Reason: "must not exceed 2 MiB"}
	}
	var source map[string]any
	if err := json.Unmarshal([]byte(raw), &source); err != nil {
		return nil, ErrInvalid{Field: "simulator_source_json", Reason: "must be a valid OpenAPI or recorded-exchange document"}
	}
	kind, _ := source["kind"].(string)
	sim := &DescriptionSimulator{kind: kind, exchanges: map[string]map[string]any{}, faultPending: injectFault}
	switch kind {
	case "openapi":
		doc, _ := source["document"].(map[string]any)
		service, err := parseAPIDescription("default", doc)
		if err != nil {
			return nil, err
		}
		sim.services = append(sim.services, service)
	case "openapi-catalog":
		services, _ := source["services"].([]any)
		if len(services) == 0 || len(services) > 128 {
			return nil, ErrInvalid{Field: "simulator_source_json.services", Reason: "catalog requires 1 to 128 API descriptions"}
		}
		for index, rawService := range services {
			entry, ok := rawService.(map[string]any)
			if !ok {
				return nil, ErrInvalid{Field: "simulator_source_json.services", Reason: "each catalog entry must be an object"}
			}
			id, _ := entry["id"].(string)
			if id == "" {
				id = fmt.Sprintf("service-%d", index+1)
			}
			doc, _ := entry["document"].(map[string]any)
			service, err := parseAPIDescription(id, doc)
			if err != nil {
				return nil, err
			}
			sim.services = append(sim.services, service)
		}
	case "recorded-exchanges":
		items, _ := source["exchanges"].([]any)
		if len(items) == 0 || len(items) > 4096 {
			return nil, ErrInvalid{Field: "simulator_source_json.exchanges", Reason: "requires 1 to 4096 recorded exchanges"}
		}
		for _, rawItem := range items {
			item, ok := rawItem.(map[string]any)
			if !ok {
				return nil, ErrInvalid{Field: "simulator_source_json.exchanges", Reason: "each exchange must be an object"}
			}
			method, _ := item["method"].(string)
			path, _ := item["path"].(string)
			host, _ := item["host"].(string)
			if method == "" || path == "" || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#") || !validSimulatorHost(host) {
				return nil, ErrInvalid{Field: "simulator_source_json.exchanges", Reason: "each exchange requires method, an exact host, and an absolute path without query or fragment"}
			}
			sim.exchanges[exchangeKey(method, host, path)] = item
		}
	default:
		return nil, ErrInvalid{Field: "simulator_source_json.kind", Reason: "must be openapi, openapi-catalog, or recorded-exchanges"}
	}
	bindings := map[string]string{}
	for _, service := range sim.services {
		for key := range service.nativeOperations {
			if previous, exists := bindings[key]; exists {
				return nil, ErrInvalid{Field: "simulator_source_json.services", Reason: fmt.Sprintf("native operation mapping %q is ambiguous across services %q and %q", key, previous, service.id)}
			}
			bindings[key] = service.id
		}
	}
	return sim, nil
}

func validSimulatorHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || strings.ContainsAny(host, "*/\\@") {
		return false
	}
	parsed, err := url.Parse("https://" + host)
	return err == nil && parsed.Host == host && parsed.Hostname() != ""
}

func (s *DescriptionSimulator) Invoke(_ context.Context, step Step, attempt uint32, injectFault bool) (string, error) {
	if injectFault && attempt == 1 && s.takeFault() {
		return "", fmt.Errorf("seeded simulator fault: synthetic 503")
	}
	status, body := s.resolve(step)
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("synthetic HTTP %d: %s", status, body)
	}
	return body, nil
}

func (s *DescriptionSimulator) takeFault() bool {
	s.faultMu.Lock()
	defer s.faultMu.Unlock()
	if !s.faultPending {
		return false
	}
	s.faultPending = false
	return true
}

func (s *DescriptionSimulator) resolve(step Step) (int, string) {
	if step.NodeType != "" {
		for _, service := range s.services {
			operation, ok := service.nativeOperations[nativeOperationKey(step.NodeType, step.Resource, step.Action)]
			if !ok {
				continue
			}
			return resolveDescriptionOperation(service, operation.Method, operation.URL)
		}
		return http.StatusNotFound, `{"error":"no matching synthetic native service operation"}`
	}
	if s.kind == "recorded-exchanges" {
		item := s.exchanges[exchangeKey(step.Method, step.Host, step.URL)]
		if item == nil {
			return http.StatusNotFound, `{"error":"no matching synthetic exchange"}`
		}
		status := http.StatusOK
		if number, ok := item["status"].(float64); ok && number >= 100 && number <= 599 {
			status = int(number)
		}
		return status, responseString(syntheticize(item["response"], ""))
	}
	for _, service := range s.services {
		if !serviceAcceptsHost(service, step.Host) {
			continue
		}
		path := step.URL
		if base := service.basePaths[strings.ToLower(step.Host)]; base != "" && (path == base || strings.HasPrefix(path, base+"/")) {
			path = strings.TrimPrefix(path, base)
			if path == "" {
				path = "/"
			}
		}
		if _, ok := matchOpenAPIOperation(service.paths, step.Method, path); !ok {
			continue
		}
		return resolveDescriptionOperation(service, step.Method, path)
	}
	return http.StatusNotFound, `{"error":"no matching synthetic API operation"}`
}

func resolveDescriptionOperation(service apiDescription, method, path string) (int, string) {
	operation, ok := matchOpenAPIOperation(service.paths, method, path)
	if !ok {
		return http.StatusNotFound, `{"error":"no matching synthetic API operation"}`
	}
	responses, _ := operation["responses"].(map[string]any)
	status, rawResponse := firstSuccessResponse(responses)
	if rawResponse == nil {
		return http.StatusNotImplemented, `{"error":"OpenAPI operation has no successful response fixture"}`
	}
	response, _ := rawResponse.(map[string]any)
	content, _ := response["content"].(map[string]any)
	jsonContent, _ := content["application/json"].(map[string]any)
	if example, ok := jsonContent["example"]; ok {
		return status, responseString(syntheticize(example, ""))
	}
	if schema, ok := jsonContent["schema"].(map[string]any); ok && len(schema) > 0 {
		return status, responseString(syntheticize(syntheticValue(schema), ""))
	}
	return status, `{"status":"synthetic-success"}`
}

func nativeOperationKey(nodeType, resource, action string) string {
	return strings.ToLower(strings.TrimSpace(nodeType)) + "|" + strings.ToLower(strings.TrimSpace(resource)) + "|" + strings.ToLower(strings.TrimSpace(action))
}

func firstSuccessResponse(responses map[string]any) (int, any) {
	keys := make([]string, 0, len(responses))
	for key := range responses {
		if strings.HasPrefix(key, "2") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return 0, nil
	}
	status := http.StatusOK
	if parsed, err := strconv.Atoi(keys[0]); err == nil {
		status = parsed
	}
	return status, responses[keys[0]]
}

func parseAPIDescription(id string, doc map[string]any) (apiDescription, error) {
	paths, _ := doc["paths"].(map[string]any)
	if len(paths) == 0 {
		return apiDescription{}, ErrInvalid{Field: "simulator_source_json.document.paths", Reason: "OpenAPI document requires paths"}
	}
	service := apiDescription{id: id, paths: map[string]map[string]any{}, basePaths: map[string]string{}, nativeOperations: map[string]Step{}}
	servers, _ := doc["servers"].([]any)
	for _, raw := range servers {
		item, _ := raw.(map[string]any)
		rawURL, _ := item["url"].(string)
		parsed, err := url.Parse(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || strings.Contains(parsed.Hostname(), "{") {
			continue
		}
		host := strings.ToLower(parsed.Hostname())
		service.hosts = append(service.hosts, host)
		if parsed.EscapedPath() != "" && parsed.EscapedPath() != "/" {
			service.basePaths[host] = strings.TrimSuffix(parsed.EscapedPath(), "/")
		}
	}
	for path, rawMethods := range paths {
		if !strings.HasPrefix(path, "/") || strings.Contains(path, "?") || strings.Contains(path, "#") {
			continue
		}
		methods, ok := rawMethods.(map[string]any)
		if ok {
			service.paths[path] = methods
			for method, rawOperation := range methods {
				operation, ok := rawOperation.(map[string]any)
				if !ok {
					continue
				}
				nodeType, _ := operation["x-n8n-node-type"].(string)
				resource, _ := operation["x-n8n-resource"].(string)
				action, _ := operation["x-n8n-operation"].(string)
				if nodeType == "" && resource == "" && action == "" {
					continue
				}
				if !isNativeServiceNode(nodeType) || resource == "" || action == "" {
					return apiDescription{}, ErrInvalid{Field: "simulator_source_json.document.paths", Reason: "native operation extensions require an n8n service node type, resource, and operation"}
				}
				key := nativeOperationKey(nodeType, resource, action)
				if _, exists := service.nativeOperations[key]; exists {
					return apiDescription{}, ErrInvalid{Field: "simulator_source_json.document.paths", Reason: "native node/resource/operation mappings must be unique per service"}
				}
				service.nativeOperations[key] = Step{Operation: key, NodeType: nodeType, Resource: resource, Action: action, Method: strings.ToUpper(method), URL: path}
			}
		}
	}
	if len(service.hosts) == 0 {
		return apiDescription{}, ErrInvalid{Field: "simulator_source_json.document.servers", Reason: "at least one absolute HTTP(S) server URL is required for host allowlisting"}
	}
	if len(service.paths) == 0 {
		return apiDescription{}, ErrInvalid{Field: "simulator_source_json.document.paths", Reason: "OpenAPI document requires valid path templates"}
	}
	return service, nil
}

func (s *DescriptionSimulator) Hosts() []string {
	set := map[string]struct{}{}
	for _, service := range s.services {
		for _, host := range service.hosts {
			set[host] = struct{}{}
		}
	}
	for key := range s.exchanges {
		parts := strings.SplitN(key, " ", 3)
		if len(parts) == 3 {
			set[parts[1]] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for host := range set {
		result = append(result, host)
	}
	sort.Strings(result)
	return result
}

func (s *DescriptionSimulator) sourceFingerprint() any {
	type operationShape struct {
		Host   string   `json:"host"`
		Base   string   `json:"base"`
		Path   string   `json:"path"`
		Method []string `json:"methods"`
	}
	type nativeOperationShape struct {
		Service  string `json:"service"`
		NodeType string `json:"node_type"`
		Resource string `json:"resource"`
		Action   string `json:"action"`
		Method   string `json:"method"`
		Path     string `json:"path"`
	}
	operations := make([]operationShape, 0)
	nativeOperations := make([]nativeOperationShape, 0)
	for _, service := range s.services {
		paths := make([]string, 0, len(service.paths))
		for path := range service.paths {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			methods := make([]string, 0, len(service.paths[path]))
			for method := range service.paths[path] {
				methods = append(methods, strings.ToUpper(method))
			}
			sort.Strings(methods)
			for _, host := range service.hosts {
				operations = append(operations, operationShape{Host: host, Base: service.basePaths[host], Path: path, Method: methods})
			}
			if len(service.hosts) == 0 {
				operations = append(operations, operationShape{Path: path, Method: methods})
			}
		}
		for _, binding := range service.nativeOperations {
			nativeOperations = append(nativeOperations, nativeOperationShape{Service: service.id, NodeType: binding.NodeType, Resource: binding.Resource, Action: binding.Action, Method: binding.Method, Path: binding.URL})
		}
	}
	sort.Slice(nativeOperations, func(i, j int) bool {
		left, right := nativeOperations[i], nativeOperations[j]
		return strings.Join([]string{left.Service, left.NodeType, left.Resource, left.Action}, "|") < strings.Join([]string{right.Service, right.NodeType, right.Resource, right.Action}, "|")
	})
	exchanges := make([]string, 0, len(s.exchanges))
	for key, item := range s.exchanges {
		status := http.StatusOK
		if number, ok := item["status"].(float64); ok {
			status = int(number)
		}
		exchanges = append(exchanges, fmt.Sprintf("%s status=%d", key, status))
	}
	sort.Strings(exchanges)
	return map[string]any{"kind": s.kind, "operations": operations, "native_operations": nativeOperations, "exchanges": exchanges}
}

// syntheticize keeps response shape while replacing all source values so a
// recorded response can never become stored rehearsal evidence verbatim.
func syntheticize(value any, key string) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for childKey, child := range typed {
			result[childKey] = syntheticize(child, childKey)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			result[index] = syntheticize(child, key)
		}
		return result
	case string:
		lower := strings.ToLower(key)
		switch {
		case strings.Contains(lower, "email"):
			return "person@example.invalid"
		case lower == "id" || strings.HasSuffix(lower, "id"):
			return "synthetic-id"
		case lower == "status":
			return "active"
		case lower == "updatedrange":
			return "A2:B2"
		default:
			return "synthetic-value"
		}
	case float64:
		return float64(1)
	case json.Number:
		return json.Number("1")
	case bool:
		return true
	default:
		return typed
	}
}

// ServeHTTP is the future companion-service bridge for n8n HTTP/API nodes.
// It serves only exact declared host/method/path fixtures, discards bounded
// request bodies, and never forwards requests. A runtime owner must still
// prove host aliases, TLS trust, and egress isolation before attaching it.
func (s *DescriptionSimulator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if r.ContentLength > maxSimulatorRequestBytes {
		http.Error(w, "synthetic request exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}
	if _, err := io.Copy(io.Discard, http.MaxBytesReader(w, r.Body, maxSimulatorRequestBytes)); err != nil {
		http.Error(w, "synthetic request exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}
	host := strings.ToLower(r.Host)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	status, body := s.resolve(Step{Method: r.Method, Host: host, URL: r.URL.EscapedPath()})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Workflow-Simulator", "synthetic")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func exchangeKey(method, host, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.ToLower(strings.TrimSpace(host)) + " " + path
}

func serviceAcceptsHost(service apiDescription, host string) bool {
	host = strings.ToLower(host)
	for _, allowed := range service.hosts {
		if allowed == host {
			return true
		}
	}
	return false
}

func responseString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return "{}"
		}
		return string(encoded)
	}
}

func matchOpenAPIOperation(paths map[string]map[string]any, method, path string) (map[string]any, bool) {
	method = strings.ToLower(method)
	for pattern, rawMethods := range paths {
		if pathMatch(pattern, path) {
			operation, ok := rawMethods[method].(map[string]any)
			if ok {
				return operation, true
			}
		}
	}
	return nil, false
}

func pathMatch(pattern, path string) bool {
	if pattern == path {
		return true
	}
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternSegments) != len(pathSegments) {
		return false
	}
	for index := range patternSegments {
		if strings.HasPrefix(patternSegments[index], "{") && strings.HasSuffix(patternSegments[index], "}") {
			continue
		}
		if patternSegments[index] != pathSegments[index] {
			return false
		}
	}
	return true
}

func syntheticValue(schema map[string]any) any {
	if example, ok := schema["example"]; ok {
		return example
	}
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		return enum[0]
	}
	switch schema["type"] {
	case "object":
		properties, _ := schema["properties"].(map[string]any)
		out := map[string]any{}
		for key, value := range properties {
			if child, ok := value.(map[string]any); ok {
				out[key] = syntheticValue(child)
			}
		}
		return out
	case "array":
		items, _ := schema["items"].(map[string]any)
		return []any{syntheticValue(items)}
	case "integer", "number":
		return 1
	case "boolean":
		return true
	default:
		return "synthetic-value"
	}
}
