package rehearsals

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	maxWorkspaceSandboxResponseBytes = 256 * 1024
	maxWorkspaceSandboxRequestBytes  = 2 * 1024 * 1024
)

var (
	sandboxIDPattern          = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	sandboxNamePattern        = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	sandboxVersionPattern     = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?$`)
	sandboxDigestPattern      = regexp.MustCompile(`^[a-f0-9]{64}$`)
	sandboxOwnerPattern       = regexp.MustCompile(`^client-rehearsal:[A-Za-z0-9._-]{1,80}$`)
	sandboxIdempotencyPattern = regexp.MustCompile(`^client-rehearsal:[A-Za-z0-9._-]{1,80}$`)
	sandboxTagPattern         = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
)

// WorkspaceSandboxProfileClient declares and starts a synthetic profile on an
// already-created task-owned Sandbox. It accepts only loopback API origins,
// refuses redirects and never accepts a source-controlled host or destination.
type WorkspaceSandboxProfileClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// WorkspaceSandboxTaskRequest identifies an already-authorized empty workspace
// scope for one rehearsal run. Paths and ownership are supplied by the local
// caller; they are never derived from imported workflow data.
type WorkspaceSandboxTaskRequest struct {
	Name           string   `json:"name,omitempty"`
	ScopePath      string   `json:"scopePath"`
	ProjectRoot    string   `json:"projectRoot"`
	Owner          string   `json:"owner"`
	OwnerType      string   `json:"ownerType"`
	Tags           []string `json:"tags,omitempty"`
	IdempotencyKey string   `json:"idempotencyKey"`
}

type WorkspaceSandboxProfileLease struct {
	SandboxID string
	Name      string
	Receipt   WorkspaceSandboxProfileReceipt
}

type WorkspaceSandboxProfileReceipt struct {
	SandboxID string `json:"-"`
	Spec      struct {
		Name    string `json:"name"`
		Kind    string `json:"kind"`
		Version string `json:"version"`
	} `json:"spec"`
	Status             string                     `json:"status"`
	Addresses          map[string]string          `json:"addresses"`
	OperationAddresses []HTTPMockOperationAddress `json:"operationAddresses"`
	ProfileSHA256      string                     `json:"profileSha256"`
}

type HTTPMockOperationAddress struct {
	Selector HTTPMockProfileSelector `json:"selector"`
	Method   string                  `json:"method"`
	URL      string                  `json:"url"`
}

func NewWorkspaceSandboxProfileClient(base string, client *http.Client) (*WorkspaceSandboxProfileClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Workspace Sandbox API base must be an absolute loopback HTTP URL without credentials, query, or fragment")
	}
	ip := net.ParseIP(parsed.Hostname())
	if parsed.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, errors.New("Workspace Sandbox API base must use localhost or a loopback IP")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return nil, errors.New("Workspace Sandbox API base must not include a path prefix")
	}
	parsed.Path = ""
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	client = &copyClient
	return &WorkspaceSandboxProfileClient{baseURL: parsed, httpClient: client}, nil
}

// CreateTaskSandbox creates a bounded caller-owned workspace task record
// using the Workspace Sandbox API's existing idempotency contract.
func (c *WorkspaceSandboxProfileClient) CreateTaskSandbox(ctx context.Context, input WorkspaceSandboxTaskRequest) (string, error) {
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return "", errors.New("Workspace Sandbox profile client is not initialized")
	}
	if !sandboxNamePattern.MatchString(input.Name) || !filepath.IsAbs(input.ProjectRoot) || input.ProjectRoot == string(filepath.Separator) || filepath.Clean(input.ProjectRoot) != input.ProjectRoot || !filepath.IsAbs(input.ScopePath) || filepath.Clean(input.ScopePath) != input.ScopePath || !sandboxOwnerPattern.MatchString(input.Owner) || input.OwnerType != "task" || input.IdempotencyKey != input.Owner || !sandboxIdempotencyPattern.MatchString(input.IdempotencyKey) || len(input.Tags) > 8 {
		return "", errors.New("task sandbox requires a bounded name, clean absolute authorized paths, owner, task owner type and idempotency key")
	}
	for _, tag := range input.Tags {
		if !sandboxTagPattern.MatchString(tag) {
			return "", errors.New("task sandbox tags must be bounded lowercase labels")
		}
	}
	rel, err := filepath.Rel(input.ProjectRoot, input.ScopePath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", errors.New("task sandbox scope must remain inside its authorized project root")
	}
	body, err := json.Marshal(input)
	if err != nil || len(body) > maxWorkspaceSandboxRequestBytes {
		return "", errors.New("task sandbox declaration is invalid or exceeds 2 MiB")
	}
	var created struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		ProjectRoot    string `json:"projectRoot"`
		ScopePath      string `json:"scopePath"`
		Owner          string `json:"owner"`
		OwnerType      string `json:"ownerType"`
		IdempotencyKey string `json:"idempotencyKey"`
	}
	if err := c.requestJSON(ctx, http.MethodPost, "/api/v1/sandboxes", body, &created); err != nil {
		return "", fmt.Errorf("create task Sandbox: %w", err)
	}
	if !sandboxIDPattern.MatchString(created.ID) {
		return "", errors.New("Workspace Sandbox returned an invalid task ID")
	}
	if created.Name != input.Name || created.ProjectRoot != input.ProjectRoot || created.ScopePath != input.ScopePath || created.Owner != input.Owner || created.OwnerType != input.OwnerType || created.IdempotencyKey != input.IdempotencyKey {
		return "", errors.New("Workspace Sandbox returned an existing task with different name, scope, project root or owner")
	}
	return created.ID, nil
}

// CreateAndStartTaskProfile owns the reversible create→declare→start path.
// If declaration or start fails after creation, it asks Workspace Sandbox to
// delete that task record; Sandbox owns companion stop-before-delete cleanup.
func (c *WorkspaceSandboxProfileClient) CreateAndStartTaskProfile(ctx context.Context, task WorkspaceSandboxTaskRequest, profileName, version string, profile HTTPMockProfile) (WorkspaceSandboxProfileLease, error) {
	sandboxID, err := c.CreateTaskSandbox(ctx, task)
	if err != nil {
		return WorkspaceSandboxProfileLease{}, err
	}
	receipt, err := c.DeclareAndStartProfile(ctx, sandboxID, profileName, version, profile)
	if err == nil {
		return WorkspaceSandboxProfileLease{SandboxID: sandboxID, Name: profileName, Receipt: receipt}, nil
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if cleanupErr := c.DeleteTaskSandbox(cleanupCtx, sandboxID); cleanupErr != nil {
		return WorkspaceSandboxProfileLease{}, fmt.Errorf("start Sandbox profile failed (%v); task cleanup also failed: %w", err, cleanupErr)
	}
	return WorkspaceSandboxProfileLease{}, fmt.Errorf("start Sandbox profile failed; created task was deleted: %w", err)
}

func (c *WorkspaceSandboxProfileClient) StopAndDeleteTaskProfile(ctx context.Context, lease WorkspaceSandboxProfileLease) error {
	if !sandboxIDPattern.MatchString(lease.SandboxID) || !sandboxNamePattern.MatchString(lease.Name) {
		return errors.New("Sandbox profile lease target is invalid")
	}
	stopErr := c.StopProfile(ctx, lease.SandboxID, lease.Name)
	deleteErr := c.DeleteTaskSandbox(ctx, lease.SandboxID)
	if stopErr != nil || deleteErr != nil {
		return errors.Join(stopErr, deleteErr)
	}
	return nil
}

func (c *WorkspaceSandboxProfileClient) DeleteTaskSandbox(ctx context.Context, sandboxID string) error {
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return errors.New("Workspace Sandbox profile client is not initialized")
	}
	if !sandboxIDPattern.MatchString(sandboxID) {
		return errors.New("Sandbox ID is invalid")
	}
	return c.requestJSON(ctx, http.MethodDelete, "/api/v1/sandboxes/"+url.PathEscape(sandboxID), nil, nil)
}

// DeclareAndStartProfile persists the normalized profile on an existing
// Sandbox and starts its fixed `http-mock` companion. On start failure the
// bounded declaration is left in place for explicit inspection/retry.
func (c *WorkspaceSandboxProfileClient) DeclareAndStartProfile(ctx context.Context, sandboxID, name, version string, profile HTTPMockProfile) (WorkspaceSandboxProfileReceipt, error) {
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return WorkspaceSandboxProfileReceipt{}, errors.New("Workspace Sandbox profile client is not initialized")
	}
	if !sandboxIDPattern.MatchString(sandboxID) || !sandboxNamePattern.MatchString(name) || !sandboxVersionPattern.MatchString(version) {
		return WorkspaceSandboxProfileReceipt{}, errors.New("Sandbox ID, companion name, or pinned version is invalid")
	}
	profile, err := normalizeExportedHTTPMockProfile(profile)
	if err != nil {
		return WorkspaceSandboxProfileReceipt{}, err
	}
	body, err := json.Marshal(struct {
		Services []struct {
			Name        string          `json:"name"`
			Kind        string          `json:"kind"`
			Version     string          `json:"version"`
			HTTPProfile HTTPMockProfile `json:"httpProfile"`
		} `json:"services"`
	}{Services: []struct {
		Name        string          `json:"name"`
		Kind        string          `json:"kind"`
		Version     string          `json:"version"`
		HTTPProfile HTTPMockProfile `json:"httpProfile"`
	}{{Name: name, Kind: "http-mock", Version: version, HTTPProfile: profile}}})
	if err != nil {
		return WorkspaceSandboxProfileReceipt{}, fmt.Errorf("encode Sandbox profile declaration: %w", err)
	}
	if len(body) > maxWorkspaceSandboxRequestBytes {
		return WorkspaceSandboxProfileReceipt{}, errors.New("Sandbox profile declaration exceeds 2 MiB")
	}
	path := "/api/v1/sandboxes/" + url.PathEscape(sandboxID) + "/companion-services"
	if err := c.requestJSON(ctx, http.MethodPut, path, body, nil); err != nil {
		return WorkspaceSandboxProfileReceipt{}, fmt.Errorf("declare Sandbox HTTP profile: %w", err)
	}
	startPath := path + "/" + url.PathEscape(name) + "/start"
	var receipt WorkspaceSandboxProfileReceipt
	if err := c.requestJSON(ctx, http.MethodPost, startPath, []byte(`{}`), &receipt); err != nil {
		return WorkspaceSandboxProfileReceipt{}, fmt.Errorf("start Sandbox HTTP profile: %w", err)
	}
	if receipt.Status != "running" || len(receipt.OperationAddresses) != len(profile.Operations) || receipt.ProfileSHA256 == "" {
		return WorkspaceSandboxProfileReceipt{}, errors.New("Sandbox returned an incomplete HTTP profile start receipt")
	}
	expectedDigest, err := httpMockProfileDigest(profile)
	if err != nil {
		return WorkspaceSandboxProfileReceipt{}, err
	}
	if err := validateProfileReceipt(name, version, expectedDigest, profile, receipt); err != nil {
		return WorkspaceSandboxProfileReceipt{}, err
	}
	receipt.SandboxID = sandboxID
	return receipt, nil
}

func (c *WorkspaceSandboxProfileClient) StopProfile(ctx context.Context, sandboxID, name string) error {
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return errors.New("Workspace Sandbox profile client is not initialized")
	}
	if !sandboxIDPattern.MatchString(sandboxID) || !sandboxNamePattern.MatchString(name) {
		return errors.New("Sandbox ID or companion name is invalid")
	}
	path := "/api/v1/sandboxes/" + url.PathEscape(sandboxID) + "/companion-services/" + url.PathEscape(name) + "/stop"
	return c.requestJSON(ctx, http.MethodPost, path, []byte(`{}`), nil)
}

func (c *WorkspaceSandboxProfileClient) requestJSON(ctx context.Context, method, path string, body []byte, output any) error {
	endpoint := *c.baseURL
	endpoint.Path += path
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Sandbox request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("request Workspace Sandbox: %w", err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxWorkspaceSandboxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read Workspace Sandbox response: %w", err)
	}
	if len(payload) > maxWorkspaceSandboxResponseBytes {
		return errors.New("Workspace Sandbox response exceeds 256 KiB")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Workspace Sandbox returned HTTP %d", response.StatusCode)
	}
	if output != nil {
		if err := json.Unmarshal(payload, output); err != nil {
			return errors.New("Workspace Sandbox returned malformed JSON")
		}
	}
	return nil
}

func normalizeExportedHTTPMockProfile(profile HTTPMockProfile) (HTTPMockProfile, error) {
	if profile.SchemaVersion != 1 || len(profile.Operations) == 0 || len(profile.Operations) > 128 {
		return HTTPMockProfile{}, errors.New("Sandbox profile requires schemaVersion 1 and 1 to 128 operations")
	}
	profile.Operations = append([]HTTPMockProfileOperation(nil), profile.Operations...)
	for index := range profile.Operations {
		profile.Operations[index].Selector.NodeType = strings.TrimSpace(profile.Operations[index].Selector.NodeType)
		profile.Operations[index].Selector.Resource = strings.TrimSpace(profile.Operations[index].Selector.Resource)
		profile.Operations[index].Selector.Operation = strings.TrimSpace(profile.Operations[index].Selector.Operation)
		profile.Operations[index].Route.Host = strings.ToLower(strings.TrimSpace(profile.Operations[index].Route.Host))
		profile.Operations[index].Route.Method = strings.ToUpper(strings.TrimSpace(profile.Operations[index].Route.Method))
	}
	sort.Slice(profile.Operations, func(i, j int) bool {
		return workspaceProfileSelectorSortKey(profile.Operations[i].Selector) < workspaceProfileSelectorSortKey(profile.Operations[j].Selector)
	})
	if err := validateExportedHTTPMockProfile(profile); err != nil {
		return HTTPMockProfile{}, err
	}
	return profile, nil
}

func workspaceProfileSelectorSortKey(selector HTTPMockProfileSelector) string {
	return fmt.Sprintf("%d:%s%d:%s%d:%s", len(selector.NodeType), selector.NodeType, len(selector.Resource), selector.Resource, len(selector.Operation), selector.Operation)
}

func validateExportedHTTPMockProfile(profile HTTPMockProfile) error {
	selectors, routes := map[string]bool{}, map[string]bool{}
	totalResponseBytes := 0
	for _, operation := range profile.Operations {
		selector := operation.Selector
		if !sandboxProfileNodeTypePattern.MatchString(selector.NodeType) || !sandboxProfilePartPattern.MatchString(selector.Resource) || !sandboxProfilePartPattern.MatchString(selector.Operation) || secretKey(selector.NodeType+selector.Resource+selector.Operation) {
			return errors.New("Sandbox profile contains an invalid selector")
		}
		route := operation.Route
		totalResponseBytes += len(route.Body)
		if !validSimulatorHost(route.Host) || !strings.HasSuffix(route.Host, ".synthetic.invalid") || !oneOfFold(route.Method, http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions) || !validSandboxProfilePath(route.Path) || route.Status < 200 || route.Status >= 300 || route.ContentType != "application/json" || len(route.Body) > maxSandboxProfileResponseBytes || totalResponseBytes > maxSandboxProfileTotalResponseBytes {
			return errors.New("Sandbox profile contains an unsafe or unsupported exact route")
		}
		selectorKey := nativeOperationKey(selector.NodeType, selector.Resource, selector.Operation)
		routeKey := route.Method + " " + route.Host + " " + route.Path
		if selectors[selectorKey] || routes[routeKey] {
			return errors.New("Sandbox profile selectors and routes must be unique")
		}
		var value any
		if err := json.Unmarshal([]byte(route.Body), &value); err != nil || containsSecretKey(value) {
			return errors.New("Sandbox profile response must be safe JSON without credential-shaped fields")
		}
		selectors[selectorKey], routes[routeKey] = true, true
	}
	return nil
}

func httpMockProfileDigest(profile HTTPMockProfile) (string, error) {
	encoded, err := json.Marshal(profile)
	if err != nil {
		return "", fmt.Errorf("encode normalized HTTP mock profile: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func validateProfileReceipt(name, version, expectedDigest string, profile HTTPMockProfile, receipt WorkspaceSandboxProfileReceipt) error {
	if receipt.Spec.Name != name || receipt.Spec.Kind != "http-mock" || receipt.Spec.Version != version || receipt.ProfileSHA256 != expectedDigest || !sandboxDigestPattern.MatchString(receipt.ProfileSHA256) || len(receipt.Addresses) == 0 {
		return errors.New("Sandbox receipt identifies a different companion or invalid profile digest")
	}
	want := make(map[string]HTTPMockProfileOperation, len(profile.Operations))
	wantHosts := make(map[string]bool, len(profile.Operations))
	for _, operation := range profile.Operations {
		want[nativeOperationKey(operation.Selector.NodeType, operation.Selector.Resource, operation.Selector.Operation)] = operation
		wantHosts[strings.ToLower(operation.Route.Host)] = true
	}
	for host, address := range receipt.Addresses {
		normalizedHost := strings.ToLower(host)
		if !wantHosts[normalizedHost] || !strings.HasSuffix(normalizedHost, ".synthetic.invalid") {
			return errors.New("Sandbox receipt contains a non-synthetic host alias")
		}
		parsed, err := url.Parse(address)
		if err != nil || parsed.Scheme != "http" || strings.ToLower(parsed.Hostname()) != normalizedHost || parsed.Port() != "18080" || parsed.Path != "" || parsed.RawQuery != "" {
			return errors.New("Sandbox receipt contains a mismatched synthetic base address")
		}
		delete(wantHosts, normalizedHost)
	}
	if len(wantHosts) != 0 {
		return errors.New("Sandbox receipt omitted one or more synthetic host aliases")
	}
	for _, address := range receipt.OperationAddresses {
		key := nativeOperationKey(address.Selector.NodeType, address.Selector.Resource, address.Selector.Operation)
		operation, ok := want[key]
		if !ok || strings.ToUpper(address.Method) != strings.ToUpper(operation.Route.Method) {
			return errors.New("Sandbox receipt contains an unexpected operation address")
		}
		parsed, err := url.Parse(address.URL)
		if err != nil || parsed.Scheme != "http" || strings.ToLower(parsed.Hostname()) != strings.ToLower(operation.Route.Host) || parsed.Port() != "18080" || parsed.Path != operation.Route.Path || parsed.RawQuery != "" {
			return errors.New("Sandbox receipt contains a non-synthetic or mismatched operation address")
		}
		delete(want, key)
	}
	if len(want) != 0 {
		return errors.New("Sandbox receipt omitted one or more declared operations")
	}
	for _, address := range receipt.Addresses {
		parsed, err := url.Parse(address)
		if err != nil || parsed.Scheme != "http" || !strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".synthetic.invalid") {
			return errors.New("Sandbox receipt contains a non-synthetic base address")
		}
	}
	return nil
}
