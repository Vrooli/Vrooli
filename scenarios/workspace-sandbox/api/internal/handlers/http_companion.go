package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxCompanionConfigBytes       = 2 << 20
	maxHTTPMockFixtureCount       = 128
	maxHTTPMockResponseBytes      = 64 << 10
	maxHTTPMockTotalResponseBytes = 1 << 20
	maxHTTPMockQueryPairs         = 32
	maxHTTPMockQueryBytes         = 2 << 10
	maxCompanionOwnerStdinBytes   = 2 << 20
	maxCompanionOwnerStdoutBytes  = 32 << 10
	maxCompanionOwnerStderrBytes  = 8 << 10
	companionOwnerCallTimeout     = 30 * time.Second
	httpMockPort                  = 18080
)

var (
	syntheticFixtureHost   = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+synthetic\.invalid$`)
	fixturePathSegment     = regexp.MustCompile(`^[A-Za-z0-9._~!$&'()*+,;=:@-]+$`)
	fixtureEmail           = regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`)
	fixtureCredentialValue = regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}|\b(?:sk-[A-Za-z0-9]{16,}|gh[pousr]_[A-Za-z0-9]{16,})\b`)
	fixtureQueryKey        = regexp.MustCompile(`^[A-Za-z0-9_.~-]{1,128}$`)
)

// HTTPMockFixture is one exact, local API operation response. OpenAPI and
// recorded-exchange interpretation stays in client-rehearsal; this normalized
// profile is what Workspace Sandbox can enforce without parsing descriptions
// or opening URLs. A non-empty Query is canonicalized and must match exactly.
type HTTPMockFixture struct {
	Host        string `json:"host"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       string `json:"query,omitempty"`
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Body        string `json:"body"`
}

// HTTPMockOperationSelector is the generic native-operation identity shared
// by profiles; it contains no workflow or node-instance identity.
type HTTPMockOperationSelector struct {
	NodeType   string         `json:"nodeType"`
	Resource   string         `json:"resource"`
	Operation  string         `json:"operation"`
	InputMatch map[string]any `json:"inputMatch,omitempty"`
}

// HTTPMockOperation maps one reusable selector to one exact static route and
// response. The adapter derives these entries from an API description or
// recorded exchanges before calling Workspace Sandbox.
type HTTPMockOperation struct {
	Selector HTTPMockOperationSelector `json:"selector"`
	Route    HTTPMockFixture           `json:"route"`
}

type HTTPMockProfile struct {
	SchemaVersion uint32              `json:"schemaVersion"`
	Operations    []HTTPMockOperation `json:"operations"`
}

type HTTPMockOperationAddress struct {
	Selector HTTPMockOperationSelector `json:"selector"`
	Method   string                    `json:"method"`
	URL      string                    `json:"url"`
}

var (
	profileNodeType  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]{0,127}$`)
	profileComponent = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
)

// NormalizeHTTPMockProfile validates selector and route uniqueness, applies
// the same fixture safeguards as raw declarations, sorts by selector and
// returns a stable hash of the normalized profile.
func NormalizeHTTPMockProfile(profile HTTPMockProfile) (HTTPMockProfile, string, error) {
	if profile.SchemaVersion != 1 {
		return HTTPMockProfile{}, "", errors.New("HTTP mock profile schemaVersion must be 1")
	}
	if len(profile.Operations) == 0 || len(profile.Operations) > maxHTTPMockFixtureCount {
		return HTTPMockProfile{}, "", fmt.Errorf("HTTP mock profile requires 1 to %d operations", maxHTTPMockFixtureCount)
	}
	selectors := make(map[string]struct{}, len(profile.Operations))
	byOperation := make(map[string][]map[string]any, len(profile.Operations))
	profile.Operations = append([]HTTPMockOperation(nil), profile.Operations...)
	for index := range profile.Operations {
		operation := &profile.Operations[index]
		selector := &operation.Selector
		selector.NodeType = strings.TrimSpace(selector.NodeType)
		selector.Resource = strings.TrimSpace(selector.Resource)
		selector.Operation = strings.TrimSpace(selector.Operation)
		if !profileNodeType.MatchString(selector.NodeType) || !profileComponent.MatchString(selector.Resource) || !profileComponent.MatchString(selector.Operation) {
			return HTTPMockProfile{}, "", fmt.Errorf("profile operation %d selector must use bounded nodeType, resource and operation identifiers", index)
		}
		match, err := normalizeHTTPMockInputMatch(selector.InputMatch)
		if err != nil {
			return HTTPMockProfile{}, "", fmt.Errorf("profile operation %d selector: %w", index, err)
		}
		selector.InputMatch = match
		baseKey := profileBaseSelectorKey(*selector)
		for _, prior := range byOperation[baseKey] {
			if httpMockInputMatchesOverlap(prior, match) {
				return HTTPMockProfile{}, "", fmt.Errorf("profile operation %d has an overlapping input selector", index)
			}
		}
		byOperation[baseKey] = append(byOperation[baseKey], match)
		key := profileSelectorKey(*selector)
		if _, exists := selectors[key]; exists {
			return HTTPMockProfile{}, "", fmt.Errorf("profile operation %d duplicates a selector", index)
		}
		selectors[key] = struct{}{}
	}
	sort.Slice(profile.Operations, func(i, j int) bool {
		return profileSelectorKey(profile.Operations[i].Selector) < profileSelectorKey(profile.Operations[j].Selector)
	})
	fixtures := make([]HTTPMockFixture, len(profile.Operations))
	for index, operation := range profile.Operations {
		fixtures[index] = operation.Route
	}
	if err := ValidateHTTPMockFixtures(fixtures); err != nil {
		return HTTPMockProfile{}, "", fmt.Errorf("profile routes: %w", err)
	}
	for index := range profile.Operations {
		profile.Operations[index].Route = fixtures[index]
		query, err := httpMockSelectorQuery(profile.Operations[index].Selector.InputMatch)
		if err != nil || (len(profile.Operations[index].Selector.InputMatch) > 0 && query != profile.Operations[index].Route.Query) {
			return HTTPMockProfile{}, "", fmt.Errorf("profile operation %d route query must exactly encode its input selector", index)
		}
	}
	encoded, err := json.Marshal(profile)
	if err != nil {
		return HTTPMockProfile{}, "", fmt.Errorf("encode normalized HTTP mock profile: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return profile, hex.EncodeToString(digest[:]), nil
}

func profileSelectorKey(selector HTTPMockOperationSelector) string {
	encoded, _ := json.Marshal(selector.InputMatch)
	return profileBaseSelectorKey(selector) + ":" + string(encoded)
}

func profileBaseSelectorKey(selector HTTPMockOperationSelector) string {
	return fmt.Sprintf("%d:%s%d:%s%d:%s", len(selector.NodeType), selector.NodeType, len(selector.Resource), selector.Resource, len(selector.Operation), selector.Operation)
}

func normalizeHTTPMockInputMatch(match map[string]any) (map[string]any, error) {
	if len(match) == 0 {
		return nil, nil
	}
	if len(match) > 16 {
		return nil, errors.New("input selector has too many fields")
	}
	copyMatch := make(map[string]any, len(match))
	for key, raw := range match {
		key = strings.TrimSpace(key)
		if !fixtureQueryKey.MatchString(key) || sensitiveFixtureField(key) {
			return nil, errors.New("input selector contains an unsafe field name")
		}
		switch value := raw.(type) {
		case string:
			if value == "" || len(value) > 512 || strings.ContainsAny(value, "\x00\r\n") || fixtureCredentialValue.MatchString(value) {
				return nil, errors.New("input selector contains an unsafe string value")
			}
			for _, email := range fixtureEmail.FindAllString(value, -1) {
				if !strings.HasSuffix(strings.ToLower(email), ".invalid") {
					return nil, errors.New("input selector contains a non-synthetic email")
				}
			}
			copyMatch[key] = value
		case bool, float64:
			copyMatch[key] = value
		default:
			return nil, errors.New("input selector values must be bounded scalars")
		}
	}
	return copyMatch, nil
}

func httpMockInputMatchesOverlap(left, right map[string]any) bool {
	for key, value := range left {
		if other, ok := right[key]; ok && other != value {
			return false
		}
	}
	return true
}

func httpMockSelectorQuery(match map[string]any) (string, error) {
	values := url.Values{}
	for key, value := range match {
		values.Set(key, fmt.Sprint(value))
	}
	return normalizeFixtureQuery(values.Encode())
}

func profileFixtures(profile HTTPMockProfile) []HTTPMockFixture {
	fixtures := make([]HTTPMockFixture, len(profile.Operations))
	for index, operation := range profile.Operations {
		fixtures[index] = operation.Route
	}
	return fixtures
}

func profileOperationAddresses(profile HTTPMockProfile, port int) []HTTPMockOperationAddress {
	addresses := make([]HTTPMockOperationAddress, 0, len(profile.Operations))
	for _, operation := range profile.Operations {
		address := fmt.Sprintf("http://%s:%d%s", operation.Route.Host, port, operation.Route.Path)
		if operation.Route.Query != "" {
			address += "?" + operation.Route.Query
		}
		addresses = append(addresses, HTTPMockOperationAddress{Selector: operation.Selector, Method: operation.Route.Method, URL: address})
	}
	return addresses
}

// ValidateHTTPMockFixtures bounds the config and makes every route an
// unambiguous exact match on a synthetic-only host. Unmatched requests have
// no upstream or forwarding behavior.
func ValidateHTTPMockFixtures(fixtures []HTTPMockFixture) error {
	if len(fixtures) == 0 || len(fixtures) > maxHTTPMockFixtureCount {
		return fmt.Errorf("requires 1 to %d exact fixtures", maxHTTPMockFixtureCount)
	}
	seen := map[string]struct{}{}
	totalBodyBytes := 0
	for index := range fixtures {
		fixture := &fixtures[index]
		fixture.Host = strings.ToLower(strings.TrimSpace(fixture.Host))
		fixture.Method = strings.ToUpper(strings.TrimSpace(fixture.Method))
		fixture.Path = strings.TrimSpace(fixture.Path)
		fixture.Query = strings.TrimSpace(fixture.Query)
		fixture.ContentType = strings.ToLower(strings.TrimSpace(fixture.ContentType))
		if len(fixture.Host) > 253 || !syntheticFixtureHost.MatchString(fixture.Host) {
			return fmt.Errorf("fixture %d host must be a synthetic *.synthetic.invalid hostname", index)
		}
		if !validFixtureMethod(fixture.Method) {
			return fmt.Errorf("fixture %d method is unsupported", index)
		}
		if !validFixturePath(fixture.Path) {
			return fmt.Errorf("fixture %d path must be an exact absolute path without query, fragment, encoding or traversal segments", index)
		}
		query, err := normalizeFixtureQuery(fixture.Query)
		if err != nil {
			return fmt.Errorf("fixture %d query: %w", index, err)
		}
		fixture.Query = query
		if fixture.Status < 200 || fixture.Status > 599 || (fixture.Status >= 300 && fixture.Status < 400) {
			return fmt.Errorf("fixture %d status must be 200–299 or 400–599", index)
		}
		if fixture.ContentType == "" {
			fixture.ContentType = "application/json"
		}
		switch fixture.ContentType {
		case "application/json", "text/plain", "application/xml":
		default:
			return fmt.Errorf("fixture %d content type is not supported", index)
		}
		if len(fixture.Body) > maxHTTPMockResponseBytes {
			return fmt.Errorf("fixture %d response exceeds %d bytes", index, maxHTTPMockResponseBytes)
		}
		totalBodyBytes += len(fixture.Body)
		if totalBodyBytes > maxHTTPMockTotalResponseBytes {
			return fmt.Errorf("combined fixture responses exceed %d bytes", maxHTTPMockTotalResponseBytes)
		}
		if fixture.ContentType == "application/json" {
			if !json.Valid([]byte(fixture.Body)) {
				return fmt.Errorf("fixture %d JSON response body is invalid", index)
			}
			var value any
			if err := json.Unmarshal([]byte(fixture.Body), &value); err != nil {
				return fmt.Errorf("fixture %d JSON response body is invalid", index)
			}
			if err := rejectFixtureSecrets(value); err != nil {
				return fmt.Errorf("fixture %d response body: %w", index, err)
			}
		} else if fixture.ContentType == "application/xml" {
			if err := validateFixtureXML(fixture.Body); err != nil {
				return fmt.Errorf("fixture %d response body: %w", index, err)
			}
		} else if strings.ContainsAny(fixture.Body, "\x00") {
			return fmt.Errorf("fixture %d text response contains a NUL byte", index)
		}
		if fixtureCredentialValue.MatchString(fixture.Body) {
			return fmt.Errorf("fixture %d response contains a credential-shaped value", index)
		}
		if match := fixtureEmail.FindAllString(fixture.Body, -1); len(match) > 0 {
			for _, email := range match {
				if !strings.HasSuffix(strings.ToLower(email), ".invalid") {
					return fmt.Errorf("fixture %d response contains a non-synthetic email address", index)
				}
			}
		}
		key := fixtureRouteKey(*fixture)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("fixture %d duplicates an exact host/method/path route", index)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// normalizeFixtureQuery makes query-key ordering stable while preserving exact
// single-value semantics. Empty query means no query parameters are accepted.
func normalizeFixtureQuery(query string) (string, error) {
	if len(query) > maxHTTPMockQueryBytes {
		return "", fmt.Errorf("query exceeds %d bytes", maxHTTPMockQueryBytes)
	}
	if query == "" {
		return "", nil
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return "", errors.New("query must be valid form-encoded key/value pairs")
	}
	if len(values) == 0 || len(values) > maxHTTPMockQueryPairs {
		return "", fmt.Errorf("query requires 1 to %d unique keys", maxHTTPMockQueryPairs)
	}
	for key, items := range values {
		if !fixtureQueryKey.MatchString(key) || len(items) != 1 {
			return "", errors.New("query keys must be bounded identifiers with one value each")
		}
		if sensitiveFixtureField(key) {
			return "", fmt.Errorf("sensitive query key %q is not allowed", key)
		}
		if len(items[0]) > 512 || strings.ContainsAny(items[0], "\x00\r\n") {
			return "", errors.New("query values must be at most 512 bytes and contain no control characters")
		}
		if fixtureCredentialValue.MatchString(items[0]) {
			return "", errors.New("query value contains a credential-shaped value")
		}
		for _, email := range fixtureEmail.FindAllString(items[0], -1) {
			if !strings.HasSuffix(strings.ToLower(email), ".invalid") {
				return "", errors.New("query value contains a non-synthetic email address")
			}
		}
	}
	canonical := values.Encode()
	if len(canonical) > maxHTTPMockQueryBytes {
		return "", fmt.Errorf("canonical query exceeds %d bytes", maxHTTPMockQueryBytes)
	}
	return canonical, nil
}

func fixtureRouteKey(fixture HTTPMockFixture) string {
	return fixture.Method + " " + fixture.Host + " " + fixture.Path + "?" + fixture.Query
}

func validFixtureMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

func validFixturePath(path string) bool {
	if len(path) == 0 || len(path) > 2048 || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#%\\\x00") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." || segment == "." {
			return false
		}
		if segment != "" && !fixturePathSegment.MatchString(segment) {
			return false
		}
	}
	return true
}

func rejectFixtureSecrets(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if sensitiveFixtureField(key) {
				return fmt.Errorf("sensitive field %q is not allowed", key)
			}
			if err := rejectFixtureSecrets(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := rejectFixtureSecrets(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func sensitiveFixtureField(key string) bool {
	normalized := strings.ToLower(key)
	normalized = strings.NewReplacer("_", "", "-", "", ".", "").Replace(normalized)
	for _, marker := range []string{"password", "passwd", "secret", "token", "authorization", "cookie", "apikey", "clientsecret", "privatekey"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func validateFixtureXML(body string) error {
	decoder := xml.NewDecoder(strings.NewReader(body))
	decoder.Strict = true
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("invalid XML response: %w", err)
		}
		switch typed := token.(type) {
		case xml.StartElement:
			if sensitiveFixtureField(typed.Name.Local) {
				return fmt.Errorf("sensitive XML element %q is not allowed", typed.Name.Local)
			}
		case xml.Directive, xml.ProcInst:
			return errors.New("XML directives and processing instructions are not allowed")
		}
	}
}

type N8NHTTPMockOwner interface {
	StartHTTPMock(context.Context, uuid.UUID, string, string, []HTTPMockFixture) (N8NHTTPMockReceipt, error)
	StopHTTPMock(context.Context, uuid.UUID, string) error
}

type N8NHTTPMockReceipt struct {
	SandboxID          string   `json:"sandboxId"`
	CompanionRuntimeID string   `json:"companionRuntimeId"`
	NetworkAliases     []string `json:"networkAliases"`
	Port               int      `json:"port"`
	Status             string   `json:"status"`
	NoEgressVerified   bool     `json:"noEgressVerified"`
}

type N8NHTTPMockServiceProvider struct{ Owner N8NHTTPMockOwner }

func (p N8NHTTPMockServiceProvider) Start(ctx context.Context, sandboxID uuid.UUID, spec CompanionServiceSpec) (CompanionRuntime, error) {
	if p.Owner == nil {
		return CompanionRuntime{}, errors.New("n8n HTTP mock owner is unavailable")
	}
	if spec.Kind != "http-mock" || !companionName.MatchString(spec.Name) || !companionVersion.MatchString(spec.Version) {
		return CompanionRuntime{}, errors.New("HTTP mock declaration kind, name or pinned version is invalid")
	}
	fixtures := append([]HTTPMockFixture(nil), spec.HTTPFixtures...)
	var profileDigest string
	var operationAddresses []HTTPMockOperationAddress
	if spec.HTTPProfile != nil {
		profile, digest, err := NormalizeHTTPMockProfile(*spec.HTTPProfile)
		if err != nil {
			return CompanionRuntime{}, err
		}
		spec.HTTPProfile = &profile
		profileDigest = digest
		fixtures = profileFixtures(profile)
	}
	if err := ValidateHTTPMockFixtures(fixtures); err != nil {
		return CompanionRuntime{}, err
	}
	receipt, err := p.Owner.StartHTTPMock(ctx, sandboxID, spec.Name, spec.Version, fixtures)
	if err != nil {
		return CompanionRuntime{}, err
	}
	expected := fixtureHosts(fixtures)
	if receipt.SandboxID != sandboxID.String() || receipt.Status != "attached" || !receipt.NoEgressVerified || receipt.Port != httpMockPort || !ownerCompanionRuntimeID.MatchString(receipt.CompanionRuntimeID) || !equalSorted(expected, receipt.NetworkAliases) {
		if ownerCompanionRuntimeID.MatchString(receipt.CompanionRuntimeID) {
			_ = p.Owner.StopHTTPMock(ctx, sandboxID, receipt.CompanionRuntimeID)
		}
		return CompanionRuntime{}, errors.New("n8n owner returned an invalid HTTP mock attachment receipt")
	}
	addresses := make(map[string]string, len(expected))
	for _, host := range expected {
		addresses[host] = fmt.Sprintf("http://%s:%d", host, receipt.Port)
	}
	if spec.HTTPProfile != nil {
		operationAddresses = profileOperationAddresses(*spec.HTTPProfile, receipt.Port)
	}
	return CompanionRuntime{ID: receipt.CompanionRuntimeID, Addresses: addresses, OperationAddresses: operationAddresses, ProfileSHA256: profileDigest}, nil
}

func (p N8NHTTPMockServiceProvider) Stop(ctx context.Context, sandboxID uuid.UUID, state CompanionServiceState) error {
	if p.Owner == nil {
		return errors.New("n8n HTTP mock owner is unavailable")
	}
	if state.Spec.Kind != "http-mock" || !ownerCompanionRuntimeID.MatchString(strings.TrimSpace(state.RuntimeID)) {
		return errors.New("valid HTTP mock companion runtime ID is required for cleanup")
	}
	return p.Owner.StopHTTPMock(ctx, sandboxID, state.RuntimeID)
}

// N8NCompanionServiceRouter combines the supported Postgres and HTTP mock
// owner contracts while leaving unsupported kinds fail-closed.
type N8NCompanionServiceRouter struct {
	Postgres N8NCompanionServiceProvider
	HTTPMock N8NHTTPMockServiceProvider
}

func (r N8NCompanionServiceRouter) Start(ctx context.Context, sandboxID uuid.UUID, spec CompanionServiceSpec) (CompanionRuntime, error) {
	switch spec.Kind {
	case "postgres":
		return r.Postgres.Start(ctx, sandboxID, spec)
	case "http-mock":
		return r.HTTPMock.Start(ctx, sandboxID, spec)
	default:
		return CompanionRuntime{}, fmt.Errorf("unsupported companion service kind %q", spec.Kind)
	}
}

func (r N8NCompanionServiceRouter) Stop(ctx context.Context, sandboxID uuid.UUID, state CompanionServiceState) error {
	switch state.Spec.Kind {
	case "postgres":
		return r.Postgres.Stop(ctx, sandboxID, state)
	case "http-mock":
		return r.HTTPMock.Stop(ctx, sandboxID, state)
	default:
		return fmt.Errorf("unsupported companion service kind %q", state.Spec.Kind)
	}
}

func fixtureHosts(fixtures []HTTPMockFixture) []string {
	seen := map[string]struct{}{}
	hosts := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		if _, ok := seen[fixture.Host]; ok {
			continue
		}
		seen[fixture.Host] = struct{}{}
		hosts = append(hosts, fixture.Host)
	}
	sort.Strings(hosts)
	return hosts
}

func equalSorted(expected, actual []string) bool {
	got := append([]string(nil), actual...)
	sort.Strings(got)
	if len(expected) != len(got) {
		return false
	}
	for i := range expected {
		if expected[i] != got[i] {
			return false
		}
	}
	return true
}

type N8NHTTPMockCLI struct{ Binary string }

func (c N8NHTTPMockCLI) StartHTTPMock(ctx context.Context, sandboxID uuid.UUID, name, version string, fixtures []HTTPMockFixture) (N8NHTTPMockReceipt, error) {
	if c.Binary == "" {
		return N8NHTTPMockReceipt{}, errors.New("vrooli control-plane executable is not configured")
	}
	if err := ValidateHTTPMockFixtures(fixtures); err != nil {
		return N8NHTTPMockReceipt{}, err
	}
	input, err := json.Marshal(struct {
		Name     string            `json:"name"`
		Version  string            `json:"version"`
		Fixtures []HTTPMockFixture `json:"fixtures"`
	}{name, version, fixtures})
	if err != nil {
		return N8NHTTPMockReceipt{}, fmt.Errorf("encode HTTP mock config: %w", err)
	}
	if len(input) > maxCompanionOwnerStdinBytes {
		return N8NHTTPMockReceipt{}, errors.New("HTTP mock config exceeds owner input bound")
	}
	output, err := c.run(ctx, input, "resource", "n8n", "runtime", "companion-start-http-mock", "--sandbox-id", sandboxID.String(), "--config-stdin", "--json")
	if err != nil {
		return N8NHTTPMockReceipt{}, err
	}
	var receipt N8NHTTPMockReceipt
	if err := json.Unmarshal(output, &receipt); err != nil {
		return N8NHTTPMockReceipt{}, fmt.Errorf("decode HTTP mock owner receipt: %w", err)
	}
	return receipt, nil
}

func (c N8NHTTPMockCLI) StopHTTPMock(ctx context.Context, sandboxID uuid.UUID, runtimeID string) error {
	if c.Binary == "" {
		return errors.New("vrooli control-plane executable is not configured")
	}
	_, err := c.run(ctx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID.String(), "--companion-runtime-id", runtimeID, "--json")
	return err
}

func (c N8NHTTPMockCLI) run(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, companionOwnerCallTimeout)
	defer cancel()
	cmd := exec.CommandContext(callCtx, c.Binary, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	for _, key := range []string{"HOME", "VROOLI_HOME", "VROOLI_ROOT", "XDG_CONFIG_HOME"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	stdout, stderr := &boundedWriter{limit: maxCompanionOwnerStdoutBytes}, &boundedWriter{limit: maxCompanionOwnerStderrBytes}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("n8n HTTP mock owner command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

type boundedWriter struct {
	bytes.Buffer
	limit int
}

func (w *boundedWriter) Write(data []byte) (int, error) {
	if w.Len()+len(data) > w.limit {
		return 0, fmt.Errorf("owner command output exceeds %d bytes", w.limit)
	}
	return w.Buffer.Write(data)
}

var _ CompanionServiceAPI = N8NHTTPMockServiceProvider{}
var _ CompanionServiceAPI = N8NCompanionServiceRouter{}
var _ N8NHTTPMockOwner = N8NHTTPMockCLI{}
var _ io.Writer = (*boundedWriter)(nil)

// NormalizeFixtureHost is exported for the client-rehearsal adapter contract:
// real service domains should be mapped to deterministic *.synthetic.invalid
// aliases before Workspace Sandbox receives fixtures.
func NormalizeFixtureHost(host string) (string, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if _, _, err := net.SplitHostPort(host); err == nil {
		return "", errors.New("fixture host must not contain a port")
	}
	if !syntheticFixtureHost.MatchString(host) {
		return "", errors.New("fixture host must end in .synthetic.invalid")
	}
	return host, nil
}
