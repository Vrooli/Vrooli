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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const maxSimulatorRequestBytes = 512 * 1024
const maxSimulatorSourceBytes = 2 * 1024 * 1024
const maxSandboxProfileResponseBytes = 64 * 1024
const maxSandboxProfileTotalResponseBytes = 1024 * 1024

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

// HTTPMockProfile mirrors the bounded Workspace Sandbox declaration format.
// It is emitted from generic API descriptions so callers can persist/start the
// same exact synthetic operations outside the in-process simulator.
type HTTPMockProfile struct {
	SchemaVersion uint32                     `json:"schemaVersion"`
	Operations    []HTTPMockProfileOperation `json:"operations"`
}

type HTTPMockProfileOperation struct {
	Selector HTTPMockProfileSelector `json:"selector"`
	Route    HTTPMockRoute           `json:"route"`
}

type HTTPMockProfileSelector struct {
	NodeType  string `json:"nodeType"`
	Resource  string `json:"resource"`
	Operation string `json:"operation"`
}

type HTTPMockRoute struct {
	Host        string `json:"host"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Body        string `json:"body"`
}

var (
	sandboxProfileNodeTypePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]{0,127}$`)
	sandboxProfilePartPattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
	sandboxProfilePathPartPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

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
	nativeOperations map[string][]Step
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
	type bindingOwner struct {
		service string
		step    Step
	}
	bindings := map[string][]bindingOwner{}
	for _, service := range sim.services {
		for key, candidates := range service.nativeOperations {
			for _, candidate := range candidates {
				for _, previous := range bindings[key] {
					if inputMatchesOverlap(previous.step.InputMatch, candidate.InputMatch) {
						return nil, ErrInvalid{Field: "simulator_source_json.services", Reason: fmt.Sprintf("native operation mapping %q has overlapping input selectors across services %q and %q", key, previous.service, service.id)}
					}
				}
				bindings[key] = append(bindings[key], bindingOwner{service: service.id, step: candidate})
			}
		}
	}
	return sim, nil
}

// ExportHTTPMockProfile derives exact static routes from OpenAPI operations
// explicitly bound to n8n node/resource/operation selectors. It refuses
// parameterized paths, non-synthetic hosts, duplicate routes, and non-catalog
// sources. Recorded exchanges do not carry selectors and cannot be exported
// without an explicit API-description binding.
func (s *DescriptionSimulator) ExportHTTPMockProfile() (HTTPMockProfile, error) {
	if s == nil || len(s.services) == 0 {
		return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json", Reason: "HTTP mock export requires an OpenAPI API catalog"}
	}
	profile := HTTPMockProfile{SchemaVersion: 1}
	for _, service := range s.services {
		if len(service.nativeOperations) == 0 {
			continue
		}
		if len(service.hosts) != 1 {
			return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.servers", Reason: "native API operation export requires exactly one declared host per service"}
		}
		host := service.hosts[0]
		if !strings.HasSuffix(host, ".synthetic.invalid") {
			return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.servers", Reason: "HTTP mock export requires synthetic.invalid hosts"}
		}
		for _, candidateBindings := range service.nativeOperations {
			for _, binding := range candidateBindings {
				if len(binding.InputMatch) > 0 {
					return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.document.paths", Reason: "HTTP mock export does not yet preserve input-conditioned native selectors"}
				}
				if strings.ContainsAny(binding.URL, "{}?#%\\") || !strings.HasPrefix(binding.URL, "/") {
					return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.paths", Reason: "HTTP mock export requires exact static paths"}
				}
				pathItem := service.paths[binding.URL]
				if hasQueryParameter(pathItem) {
					return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.parameters", Reason: "HTTP mock export does not yet support variable query parameters"}
				}
				operation, ok := matchOpenAPIOperation(service.paths, binding.Method, binding.URL)
				if !ok {
					return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.paths", Reason: "bound API operation is missing"}
				}
				responses, _ := operation["responses"].(map[string]any)
				status, raw := firstSuccessResponse(responses)
				if raw == nil {
					return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.responses", Reason: "HTTP mock export requires a successful response fixture"}
				}
				response, _ := raw.(map[string]any)
				content, _ := response["content"].(map[string]any)
				jsonContent, _ := content["application/json"].(map[string]any)
				var value any
				if example, exists := jsonContent["example"]; exists {
					value = example
				} else if schema, exists := jsonContent["schema"].(map[string]any); exists && len(schema) > 0 {
					value = syntheticValue(schema)
				} else if len(jsonContent) > 0 {
					value = map[string]any{"status": "synthetic-success"}
				} else {
					return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.responses", Reason: "HTTP mock export supports only JSON response fixtures"}
				}
				if containsSecretKey(value) {
					return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json.responses", Reason: "HTTP mock response contains a credential-shaped field"}
				}
				body := responseString(syntheticize(value, ""))
				path := strings.TrimSuffix(service.basePaths[host], "/") + binding.URL
				profile.Operations = append(profile.Operations, HTTPMockProfileOperation{
					Selector: HTTPMockProfileSelector{NodeType: binding.NodeType, Resource: binding.Resource, Operation: binding.Action},
					Route:    HTTPMockRoute{Host: host, Method: binding.Method, Path: path, Status: status, ContentType: "application/json", Body: body},
				})
			}
		}
	}
	if len(profile.Operations) == 0 || len(profile.Operations) > 128 {
		return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json", Reason: "HTTP mock export requires 1 to 128 bound operations"}
	}
	sort.Slice(profile.Operations, func(i, j int) bool {
		return workspaceProfileSelectorSortKey(profile.Operations[i].Selector) < workspaceProfileSelectorSortKey(profile.Operations[j].Selector)
	})
	routes := map[string]bool{}
	selectors := map[string]bool{}
	totalResponseBytes := 0
	for _, operation := range profile.Operations {
		selector := operation.Selector
		if !sandboxProfileNodeTypePattern.MatchString(selector.NodeType) || !sandboxProfilePartPattern.MatchString(selector.Resource) || !sandboxProfilePartPattern.MatchString(selector.Operation) {
			return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json", Reason: "HTTP mock export contains an unsupported native selector"}
		}
		route := operation.Route
		totalResponseBytes += len(route.Body)
		if !validSandboxProfilePath(route.Path) || !oneOfFold(route.Method, http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions) || route.Status < 200 || route.Status >= 300 || len(route.Body) > maxSandboxProfileResponseBytes || totalResponseBytes > maxSandboxProfileTotalResponseBytes || route.ContentType != "application/json" {
			return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json", Reason: "HTTP mock export contains an unsupported static route or response"}
		}
		selectorKey := nativeOperationKey(selector.NodeType, selector.Resource, selector.Operation)
		routeKey := route.Method + " " + route.Host + " " + route.Path
		if selectors[selectorKey] || routes[routeKey] {
			return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json", Reason: "HTTP mock export has duplicate selectors or exact routes"}
		}
		selectors[selectorKey], routes[routeKey] = true, true
	}
	encoded, err := json.Marshal(profile)
	if err != nil || len(encoded) > maxSimulatorSourceBytes {
		return HTTPMockProfile{}, ErrInvalid{Field: "simulator_source_json", Reason: "HTTP mock export exceeds the 2 MiB profile bound"}
	}
	return profile, nil
}

func validSandboxProfilePath(path string) bool {
	if len(path) == 0 || len(path) > 2048 || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#%\\\x00") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." || (segment != "" && !sandboxProfilePathPartPattern.MatchString(segment)) {
			return false
		}
	}
	return true
}

func hasQueryParameter(pathItem map[string]any) bool {
	for _, raw := range []any{pathItem["parameters"]} {
		if parameterListHasQuery(raw) {
			return true
		}
	}
	for _, rawOperation := range pathItem {
		operation, ok := rawOperation.(map[string]any)
		if !ok {
			continue
		}
		if parameterListHasQuery(operation["parameters"]) {
			return true
		}
	}
	return false
}

func parameterListHasQuery(raw any) bool {
	parameters, _ := raw.([]any)
	for _, item := range parameters {
		parameter, _ := item.(map[string]any)
		if location, _ := parameter["in"].(string); strings.EqualFold(location, "query") {
			return true
		}
	}
	return false
}

func containsSecretKey(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if secretKey(key) || containsSecretKey(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsSecretKey(child) {
				return true
			}
		}
	}
	return false
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
			candidates := service.nativeOperations[nativeOperationKey(step.NodeType, step.Resource, step.Action)]
			for _, operation := range candidates {
				if inputMatches(operation.InputMatch, step.Input) {
					return resolveDescriptionOperation(service, operation.Method, operation.URL)
				}
			}
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

func parseNativeInputMatch(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	match, ok := raw.(map[string]any)
	if !ok || len(match) == 0 || len(match) > 16 {
		return nil, ErrInvalid{Field: "simulator_source_json.x-n8n-input-match", Reason: "must be an object with 1 to 16 exact scalar selectors"}
	}
	validated := make(map[string]any, len(match))
	for key, value := range match {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || len(key) > 64 || strings.ContainsAny(key, ".[]") || key != trimmedKey {
			return nil, ErrInvalid{Field: "simulator_source_json.x-n8n-input-match", Reason: "selector keys must be short top-level input field names"}
		}
		key = trimmedKey
		switch typed := value.(type) {
		case string:
			if typed == "" || len(typed) > 512 || strings.ContainsRune(typed, '\x00') {
				return nil, ErrInvalid{Field: "simulator_source_json.x-n8n-input-match", Reason: "string selector values must be nonempty and at most 512 bytes"}
			}
		case bool, float64:
		default:
			return nil, ErrInvalid{Field: "simulator_source_json.x-n8n-input-match", Reason: "selector values must be strings, numbers, or booleans"}
		}
		validated[key] = value
	}
	return validated, nil
}

func inputMatches(match, input map[string]any) bool {
	for key, want := range match {
		got, ok := input[key]
		if !ok || !sameSimulatorScalar(got, want) {
			return false
		}
	}
	return true
}

// Two predicates overlap when no shared field excludes a common input. Empty
// predicates are unconditional and therefore overlap every binding.
func inputMatchesOverlap(left, right map[string]any) bool {
	for key, leftValue := range left {
		if rightValue, exists := right[key]; exists && !sameSimulatorScalar(leftValue, rightValue) {
			return false
		}
	}
	return true
}

func sameSimulatorScalar(left, right any) bool {
	switch leftValue := left.(type) {
	case string:
		rightValue, ok := right.(string)
		return ok && leftValue == rightValue
	case bool:
		rightValue, ok := right.(bool)
		return ok && leftValue == rightValue
	case float64:
		rightValue, ok := right.(float64)
		return ok && leftValue == rightValue
	default:
		return false
	}
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
	service := apiDescription{id: id, paths: map[string]map[string]any{}, basePaths: map[string]string{}, nativeOperations: map[string][]Step{}}
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
				inputMatch, matchErr := parseNativeInputMatch(operation["x-n8n-input-match"])
				if matchErr != nil {
					return apiDescription{}, matchErr
				}
				if nodeType == "" && resource == "" && action == "" {
					continue
				}
				if !isNativeServiceNode(nodeType) || resource == "" || action == "" {
					return apiDescription{}, ErrInvalid{Field: "simulator_source_json.document.paths", Reason: "native operation extensions require an n8n service node type, resource, and operation"}
				}
				key := nativeOperationKey(nodeType, resource, action)
				for _, previous := range service.nativeOperations[key] {
					if inputMatchesOverlap(previous.InputMatch, inputMatch) {
						return apiDescription{}, ErrInvalid{Field: "simulator_source_json.document.paths", Reason: "native operation input selectors must be disjoint"}
					}
				}
				service.nativeOperations[key] = append(service.nativeOperations[key], Step{Operation: key, NodeType: nodeType, Resource: resource, Action: action, Method: strings.ToUpper(method), URL: path, InputMatch: inputMatch})
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
		Service    string         `json:"service"`
		NodeType   string         `json:"node_type"`
		Resource   string         `json:"resource"`
		Action     string         `json:"action"`
		Method     string         `json:"method"`
		Path       string         `json:"path"`
		InputMatch map[string]any `json:"input_match,omitempty"`
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
		for _, candidates := range service.nativeOperations {
			for _, binding := range candidates {
				nativeOperations = append(nativeOperations, nativeOperationShape{Service: service.id, NodeType: binding.NodeType, Resource: binding.Resource, Action: binding.Action, Method: binding.Method, Path: binding.URL, InputMatch: binding.InputMatch})
			}
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
		case lower == "type" && (typed == "person" || typed == "bot"):
			// Preserve these low-risk category enums so generic Switch nodes can
			// exercise people-vs-automation paths without retaining identity data.
			return typed
		case strings.Contains(lower, "status") && oneOfFold(typed, "To develop", "Done", "In progress", "active", "completed", "open", "closed"):
			// Recognized workflow-state enums carry branch meaning but no identity
			// or secret material. Other status strings remain syntheticized below.
			return typed
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

func oneOfFold(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.EqualFold(value, candidate) {
			return true
		}
	}
	return false
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
