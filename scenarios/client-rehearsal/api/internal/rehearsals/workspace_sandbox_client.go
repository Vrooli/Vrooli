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
	taskScopeIDPattern        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	sandboxIDPattern          = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	sandboxNamePattern        = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	sandboxVersionPattern     = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?$`)
	sandboxDigestPattern      = regexp.MustCompile(`^[a-f0-9]{64}$`)
	sandboxProfileQueryKey    = regexp.MustCompile(`^[A-Za-z0-9_.~-]{1,128}$`)
	sandboxProfileCredential  = regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}|\b(?:sk-[A-Za-z0-9]{16,}|gh[pousr]_[A-Za-z0-9]{16,})\b`)
	sandboxProfileEmail       = regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`)
	sandboxVirtualAPIHost     = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	sandboxVirtualAPIPathPart = regexp.MustCompile(`^[A-Za-z0-9._~!$&'()*+,;=:@-]+$`)
)

// WorkspaceSandboxProfileClient declares and starts a synthetic profile on an
// already-created task-owned Sandbox. It accepts only loopback API origins,
// refuses redirects and never accepts a source-controlled host or destination.
type WorkspaceSandboxProfileClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// WorkspaceSandboxTaskRequest selects a logical task identity. Workspace
// Sandbox resolves its directory and owner attribution from trusted server
// configuration; callers cannot choose either path or owner.
type WorkspaceSandboxTaskRequest struct {
	TaskID string `json:"taskId"`
	Name   string `json:"name,omitempty"`
}

type WorkspaceSandboxTaskRecord struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ProjectRoot    string `json:"projectRoot"`
	ScopePath      string `json:"scopePath"`
	Owner          string `json:"owner"`
	OwnerType      string `json:"ownerType"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type WorkspaceSandboxProfileLease struct {
	SandboxID  string
	Name       string
	Task       WorkspaceSandboxTaskRecord
	Runtime    N8NTaskRuntimeLease
	Receipt    WorkspaceSandboxProfileReceipt
	VirtualAPI *VirtualAPIProfile
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

type HTTPMockRouteHit struct {
	Host        string `json:"host"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       string `json:"query,omitempty"`
	QuerySHA256 string `json:"queryHash,omitempty"`
	Count       uint64 `json:"count"`
}

type HTTPMockRouteHitsReceipt struct {
	SandboxID          string             `json:"sandboxId"`
	CompanionRuntimeID string             `json:"companionRuntimeId"`
	Status             string             `json:"status"`
	TotalHits          uint64             `json:"totalHits"`
	Routes             []HTTPMockRouteHit `json:"routes"`
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
func (c *WorkspaceSandboxProfileClient) CreateTaskSandbox(ctx context.Context, input WorkspaceSandboxTaskRequest) (WorkspaceSandboxTaskRecord, error) {
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return WorkspaceSandboxTaskRecord{}, errors.New("Workspace Sandbox profile client is not initialized")
	}
	if !taskScopeIDPattern.MatchString(input.TaskID) || input.TaskID == "." || input.TaskID == ".." || (input.Name != "" && !sandboxNamePattern.MatchString(input.Name)) {
		return WorkspaceSandboxTaskRecord{}, errors.New("task sandbox requires a bounded task ID and optional name")
	}
	body, err := json.Marshal(input)
	if err != nil || len(body) > maxWorkspaceSandboxRequestBytes {
		return WorkspaceSandboxTaskRecord{}, errors.New("task sandbox declaration is invalid or exceeds 2 MiB")
	}
	var created WorkspaceSandboxTaskRecord
	if err := c.requestJSON(ctx, http.MethodPost, "/api/v1/task-scopes", body, &created); err != nil {
		return WorkspaceSandboxTaskRecord{}, fmt.Errorf("create server-owned task Sandbox: %w", err)
	}
	if !sandboxIDPattern.MatchString(created.ID) || created.OwnerType != "task" || created.IdempotencyKey != "task-scope:"+input.TaskID || !filepath.IsAbs(created.ProjectRoot) || !filepath.IsAbs(created.ScopePath) {
		return WorkspaceSandboxTaskRecord{}, errors.New("Workspace Sandbox returned an incomplete server-owned task scope")
	}
	if (input.Name != "" && created.Name != input.Name) || !strings.HasSuffix(created.Owner, ":"+input.TaskID) || len(created.Owner) <= len(input.TaskID)+1 {
		return WorkspaceSandboxTaskRecord{}, errors.New("Workspace Sandbox returned a task scope with unexpected name or missing server-owned attribution")
	}
	rel, err := filepath.Rel(created.ProjectRoot, created.ScopePath)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return WorkspaceSandboxTaskRecord{}, errors.New("Workspace Sandbox returned a task scope outside its project root")
	}
	return created, nil
}

// CreateAndStartTaskProfile owns the reversible create→declare→start path.
// If declaration or start fails after creation, it asks Workspace Sandbox to
// delete that task record; Sandbox owns companion stop-before-delete cleanup.
func (c *WorkspaceSandboxProfileClient) CreateAndStartTaskProfile(ctx context.Context, task WorkspaceSandboxTaskRequest, profileName, version string, profile HTTPMockProfile) (WorkspaceSandboxProfileLease, error) {
	created, err := c.CreateTaskSandbox(ctx, task)
	if err != nil {
		return WorkspaceSandboxProfileLease{}, err
	}
	lease, err := c.StartTaskProfileOnSandbox(ctx, created, profileName, version, profile)
	if err == nil {
		return lease, nil
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if cleanupErr := c.DeleteTaskSandbox(cleanupCtx, created.ID); cleanupErr != nil {
		return WorkspaceSandboxProfileLease{}, fmt.Errorf("start Sandbox profile failed (%v); task cleanup also failed: %w", err, cleanupErr)
	}
	return WorkspaceSandboxProfileLease{}, fmt.Errorf("start Sandbox profile failed; created task was deleted: %w", err)
}

func (c *WorkspaceSandboxProfileClient) StartTaskProfileOnSandbox(ctx context.Context, task WorkspaceSandboxTaskRecord, profileName, version string, profile HTTPMockProfile) (WorkspaceSandboxProfileLease, error) {
	if !sandboxIDPattern.MatchString(task.ID) || task.OwnerType != "task" || task.ScopePath == "" || task.ProjectRoot == "" {
		return WorkspaceSandboxProfileLease{}, errors.New("server-owned Sandbox task record is incomplete")
	}
	receipt, err := c.DeclareAndStartProfile(ctx, task.ID, profileName, version, profile)
	if err != nil {
		return WorkspaceSandboxProfileLease{}, err
	}
	return WorkspaceSandboxProfileLease{SandboxID: task.ID, Name: profileName, Task: task, Receipt: receipt}, nil
}

func (c *WorkspaceSandboxProfileClient) CreateAndStartTaskVirtualAPIProfile(ctx context.Context, task WorkspaceSandboxTaskRequest, profileName, version string, profile VirtualAPIProfile) (WorkspaceSandboxProfileLease, error) {
	created, err := c.CreateTaskSandbox(ctx, task)
	if err != nil {
		return WorkspaceSandboxProfileLease{}, err
	}
	lease, err := c.StartTaskVirtualAPIProfileOnSandbox(ctx, created, profileName, version, profile)
	if err == nil {
		return lease, nil
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if cleanupErr := c.DeleteTaskSandbox(cleanupCtx, created.ID); cleanupErr != nil {
		return WorkspaceSandboxProfileLease{}, fmt.Errorf("start Sandbox virtual API profile failed (%v); task cleanup also failed: %w", err, cleanupErr)
	}
	return WorkspaceSandboxProfileLease{}, fmt.Errorf("start Sandbox virtual API profile failed; created task was deleted: %w", err)
}

func (c *WorkspaceSandboxProfileClient) StartTaskVirtualAPIProfileOnSandbox(ctx context.Context, task WorkspaceSandboxTaskRecord, profileName, version string, profile VirtualAPIProfile) (WorkspaceSandboxProfileLease, error) {
	if !sandboxIDPattern.MatchString(task.ID) || task.OwnerType != "task" || task.ScopePath == "" || task.ProjectRoot == "" {
		return WorkspaceSandboxProfileLease{}, errors.New("server-owned Sandbox task record is incomplete")
	}
	receipt, err := c.DeclareAndStartVirtualAPIProfile(ctx, task.ID, profileName, version, profile)
	if err != nil {
		return WorkspaceSandboxProfileLease{}, err
	}
	profileCopy := profile
	return WorkspaceSandboxProfileLease{SandboxID: task.ID, Name: profileName, Task: task, Receipt: receipt, VirtualAPI: &profileCopy}, nil
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

func (c *WorkspaceSandboxProfileClient) DeclareAndStartVirtualAPIProfile(ctx context.Context, sandboxID, name, version string, profile VirtualAPIProfile) (WorkspaceSandboxProfileReceipt, error) {
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return WorkspaceSandboxProfileReceipt{}, errors.New("Workspace Sandbox profile client is not initialized")
	}
	if !sandboxIDPattern.MatchString(sandboxID) || !sandboxNamePattern.MatchString(name) || !sandboxVersionPattern.MatchString(version) {
		return WorkspaceSandboxProfileReceipt{}, errors.New("Sandbox ID, companion name, or pinned version is invalid")
	}
	profile, err := normalizeExportedVirtualAPIProfile(profile)
	if err != nil {
		return WorkspaceSandboxProfileReceipt{}, err
	}
	body, err := json.Marshal(struct {
		Services []struct {
			Name       string            `json:"name"`
			Kind       string            `json:"kind"`
			Version    string            `json:"version"`
			VirtualAPI VirtualAPIProfile `json:"virtualAPI"`
		} `json:"services"`
	}{Services: []struct {
		Name       string            `json:"name"`
		Kind       string            `json:"kind"`
		Version    string            `json:"version"`
		VirtualAPI VirtualAPIProfile `json:"virtualAPI"`
	}{{Name: name, Kind: "virtual-api", Version: version, VirtualAPI: profile}}})
	if err != nil || len(body) > maxWorkspaceSandboxRequestBytes {
		return WorkspaceSandboxProfileReceipt{}, errors.New("virtual API companion declaration is invalid or exceeds 2 MiB")
	}
	path := "/api/v1/sandboxes/" + url.PathEscape(sandboxID) + "/companion-services"
	if err := c.requestJSON(ctx, http.MethodPut, path, body, nil); err != nil {
		return WorkspaceSandboxProfileReceipt{}, fmt.Errorf("declare Sandbox virtual API profile: %w", err)
	}
	startPath := path + "/" + url.PathEscape(name) + "/start"
	var receipt WorkspaceSandboxProfileReceipt
	if err := c.requestJSON(ctx, http.MethodPost, startPath, []byte(`{}`), &receipt); err != nil {
		return WorkspaceSandboxProfileReceipt{}, fmt.Errorf("start Sandbox virtual API profile: %w", err)
	}
	expectedDigest, err := virtualAPIProfileDigest(sandboxID, profile)
	if err != nil {
		return WorkspaceSandboxProfileReceipt{}, err
	}
	if receipt.Status != "running" || receipt.Spec.Name != name || receipt.Spec.Kind != "virtual-api" || receipt.Spec.Version != version || receipt.ProfileSHA256 != expectedDigest || len(receipt.OperationAddresses) != 0 || len(receipt.Addresses) != len(profile.Hosts) {
		return WorkspaceSandboxProfileReceipt{}, errors.New("Sandbox returned an incomplete virtual API profile start receipt")
	}
	for _, host := range profile.Hosts {
		if receipt.Addresses[host] != "https://"+host+":443" {
			return WorkspaceSandboxProfileReceipt{}, errors.New("Sandbox returned a virtual API address outside the task HTTPS alias map")
		}
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

func (c *WorkspaceSandboxProfileClient) ReadRouteHits(ctx context.Context, sandboxID, name string) (HTTPMockRouteHitsReceipt, error) {
	if c == nil || c.baseURL == nil || c.httpClient == nil || !sandboxIDPattern.MatchString(sandboxID) || !sandboxNamePattern.MatchString(name) {
		return HTTPMockRouteHitsReceipt{}, errors.New("Sandbox ID or companion name is invalid")
	}
	var receipt HTTPMockRouteHitsReceipt
	path := "/api/v1/sandboxes/" + url.PathEscape(sandboxID) + "/companion-services/" + url.PathEscape(name) + "/route-hits"
	if err := c.requestJSON(ctx, http.MethodGet, path, nil, &receipt); err != nil {
		return HTTPMockRouteHitsReceipt{}, fmt.Errorf("read Workspace Sandbox route-hit evidence: %w", err)
	}
	if receipt.SandboxID != sandboxID || receipt.Status != "available" || receipt.Routes == nil {
		return HTTPMockRouteHitsReceipt{}, errors.New("Workspace Sandbox returned an incomplete route-hit evidence receipt")
	}
	return receipt, nil
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
		var apiError struct {
			Error string `json:"error"`
		}
		message := ""
		if json.Unmarshal(payload, &apiError) == nil {
			message = strings.TrimSpace(apiError.Error)
		}
		if len(message) > 512 {
			message = message[:512]
		}
		message = strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return ' '
			}
			return r
		}, message)
		if message != "" {
			return fmt.Errorf("Workspace Sandbox returned HTTP %d: %s", response.StatusCode, message)
		}
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
		var rawInputMatch any
		if len(profile.Operations[index].Selector.InputMatch) > 0 {
			rawInputMatch = profile.Operations[index].Selector.InputMatch
		}
		match, err := parseNativeInputMatch(rawInputMatch)
		if err != nil {
			return HTTPMockProfile{}, errors.New("Sandbox profile contains an invalid input selector")
		}
		for key, value := range match {
			if secretKey(key) {
				return HTTPMockProfile{}, errors.New("Sandbox profile contains a sensitive input selector")
			}
			if text, ok := value.(string); ok {
				if sandboxProfileCredential.MatchString(text) {
					return HTTPMockProfile{}, errors.New("Sandbox profile contains a credential-shaped selector")
				}
				for _, email := range sandboxProfileEmail.FindAllString(text, -1) {
					if !strings.HasSuffix(strings.ToLower(email), ".invalid") {
						return HTTPMockProfile{}, errors.New("Sandbox profile contains a non-synthetic email selector")
					}
				}
			}
		}
		profile.Operations[index].Selector.InputMatch = match
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

func normalizeExportedVirtualAPIProfile(profile VirtualAPIProfile) (VirtualAPIProfile, error) {
	if len(profile.Hosts) == 0 || len(profile.Hosts) > 32 || len(profile.Routes) == 0 || len(profile.Routes) > 512 {
		return VirtualAPIProfile{}, errors.New("virtual API profile requires 1..32 hosts and 1..512 routes")
	}
	profile.Hosts = append([]string(nil), profile.Hosts...)
	hosts := map[string]bool{}
	for index, raw := range profile.Hosts {
		host := strings.ToLower(strings.TrimSpace(raw))
		if host != raw || !sandboxVirtualAPIHost.MatchString(host) || host == "localhost" || net.ParseIP(host) != nil || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".test") || strings.HasSuffix(host, ".example") || hosts[host] {
			return VirtualAPIProfile{}, errors.New("virtual API host list must contain unique exact public DNS names")
		}
		hosts[host] = true
		profile.Hosts[index] = host
	}
	sort.Strings(profile.Hosts)
	profile.Routes = append([]VirtualAPIRoute(nil), profile.Routes...)
	seen := map[string]bool{}
	total := 0
	for index := range profile.Routes {
		route := &profile.Routes[index]
		route.Host = strings.ToLower(strings.TrimSpace(route.Host))
		route.Method = strings.ToUpper(strings.TrimSpace(route.Method))
		route.Path = strings.TrimSpace(route.Path)
		route.ContentType = strings.ToLower(strings.TrimSpace(route.ContentType))
		if !hosts[route.Host] || !validVirtualAPIPath(route.Path) || !oneOfFold(route.Method, http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions) {
			return VirtualAPIProfile{}, errors.New("virtual API route host, method, or exact path is invalid")
		}
		query, err := normalizeFixtureQuery(route.Query)
		if err != nil {
			return VirtualAPIProfile{}, errors.New("virtual API route query is invalid or sensitive")
		}
		route.Query = query
		if route.Status < 200 || route.Status > 599 || route.Status >= 300 && route.Status < 400 || !oneOfFold(route.ContentType, "application/json", "text/plain", "application/xml") {
			return VirtualAPIProfile{}, errors.New("virtual API route status or content type is unsupported")
		}
		body := []byte(route.Body)
		if len(body) > 1<<20 || strings.ContainsRune(route.Body, '\x00') || sandboxProfileCredential.Match(body) {
			return VirtualAPIProfile{}, errors.New("virtual API response is oversized or credential-shaped")
		}
		for _, email := range sandboxProfileEmail.FindAllString(route.Body, -1) {
			if !strings.HasSuffix(strings.ToLower(email), ".invalid") {
				return VirtualAPIProfile{}, errors.New("virtual API response contains a non-synthetic email")
			}
		}
		if route.ContentType == "application/json" {
			var value any
			if !json.Valid(body) || json.Unmarshal(body, &value) != nil || containsSecretKey(value) {
				return VirtualAPIProfile{}, errors.New("virtual API response is malformed or contains sensitive fields")
			}
		}
		total += len(body)
		if total > 8<<20 {
			return VirtualAPIProfile{}, errors.New("virtual API responses exceed 8 MiB total")
		}
		key := route.Method + " " + route.Host + " " + route.Path + "?" + route.Query
		if seen[key] {
			return VirtualAPIProfile{}, errors.New("virtual API profile contains duplicate exact routes")
		}
		seen[key] = true
	}
	sort.Slice(profile.Routes, func(i, j int) bool {
		a, b := profile.Routes[i], profile.Routes[j]
		if a.Host != b.Host {
			return a.Host < b.Host
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Query < b.Query
	})
	return profile, nil
}

func validVirtualAPIPath(path string) bool {
	if len(path) == 0 || len(path) > 2048 || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#%\\\x00") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." || segment != "" && !sandboxVirtualAPIPathPart.MatchString(segment) {
			return false
		}
	}
	return true
}

func normalizeFixtureQuery(raw string) (string, error) {
	if len(raw) > 2048 {
		return "", errors.New("query exceeds byte limit")
	}
	values, err := url.ParseQuery(raw)
	if err != nil || len(values) > 32 {
		return "", errors.New("query is malformed or has too many keys")
	}
	keys := make([]string, 0, len(values))
	for key, items := range values {
		if key == "" || secretKey(key) || sandboxProfileCredential.MatchString(key) {
			return "", errors.New("query contains unsafe key")
		}
		for _, value := range items {
			if strings.ContainsAny(key+value, "\x00\r\n\x7f") || sandboxProfileCredential.MatchString(value) || sandboxProfileEmail.MatchString(value) {
				return "", errors.New("query contains unsafe or credential-like value")
			}
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var canonical url.Values = make(url.Values, len(values))
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		canonical[key] = items
	}
	return canonical.Encode(), nil
}

func virtualAPIProfileDigest(sandboxID string, profile VirtualAPIProfile) (string, error) {
	profile, err := normalizeExportedVirtualAPIProfile(profile)
	if err != nil {
		return "", err
	}
	type route struct {
		Host        string `json:"host"`
		Method      string `json:"method"`
		Path        string `json:"path"`
		Query       string `json:"query,omitempty"`
		Status      int    `json:"status"`
		ContentType string `json:"contentType"`
		Body        []byte `json:"body"`
	}
	contract := struct {
		TaskID string   `json:"taskId"`
		Hosts  []string `json:"hosts"`
		Routes []route  `json:"routes"`
	}{TaskID: sandboxID, Hosts: profile.Hosts, Routes: make([]route, 0, len(profile.Routes))}
	for _, item := range profile.Routes {
		contract.Routes = append(contract.Routes, route{Host: item.Host, Method: item.Method, Path: item.Path, Query: item.Query, Status: item.Status, ContentType: item.ContentType, Body: []byte(item.Body)})
	}
	encoded, err := json.Marshal(contract)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func workspaceProfileSelectorSortKey(selector HTTPMockProfileSelector) string {
	return fmt.Sprintf("%d:%s%d:%s%d:%s%s", len(selector.NodeType), selector.NodeType, len(selector.Resource), selector.Resource, len(selector.Operation), selector.Operation, canonicalInputMatch(selector.InputMatch))
}

func validateExportedHTTPMockProfile(profile HTTPMockProfile) error {
	selectors, routes := map[string]bool{}, map[string]bool{}
	byOperation := map[string][]map[string]any{}
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
		baseKey := nativeOperationKey(selector.NodeType, selector.Resource, selector.Operation)
		for _, previous := range byOperation[baseKey] {
			if inputMatchesOverlap(previous, selector.InputMatch) {
				return errors.New("Sandbox profile input selectors overlap")
			}
		}
		byOperation[baseKey] = append(byOperation[baseKey], selector.InputMatch)
		selectorKey := baseKey + ":" + canonicalInputMatch(selector.InputMatch)
		selectorQuery := url.Values{}
		for key, value := range selector.InputMatch {
			selectorQuery.Set(key, fmt.Sprint(value))
		}
		if len(selector.InputMatch) > 0 && selectorQuery.Encode() != route.Query {
			return errors.New("Sandbox profile query must exactly encode its operation selector")
		}
		if err := validateSandboxProfileQuery(route.Query); err != nil {
			return err
		}
		routeKey := route.Method + " " + route.Host + " " + route.Path + "?" + route.Query
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

func validateSandboxProfileQuery(query string) error {
	if query == "" {
		return nil
	}
	if len(query) > 2048 {
		return errors.New("Sandbox profile query exceeds its bound")
	}
	values, err := url.ParseQuery(query)
	if err != nil || len(values) == 0 || len(values) > 32 || values.Encode() != query {
		return errors.New("Sandbox profile query must be canonical and bounded")
	}
	for key, items := range values {
		if !sandboxProfileQueryKey.MatchString(key) || secretKey(key) || len(items) != 1 || len(items[0]) > 512 || strings.ContainsAny(items[0], "\x00\r\n") || sandboxProfileCredential.MatchString(items[0]) {
			return errors.New("Sandbox profile query contains an unsafe key or value")
		}
		for _, email := range sandboxProfileEmail.FindAllString(items[0], -1) {
			if !strings.HasSuffix(strings.ToLower(email), ".invalid") {
				return errors.New("Sandbox profile query contains a non-synthetic email")
			}
		}
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
		want[nativeOperationKey(operation.Selector.NodeType, operation.Selector.Resource, operation.Selector.Operation)+":"+canonicalInputMatch(operation.Selector.InputMatch)] = operation
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
		key := nativeOperationKey(address.Selector.NodeType, address.Selector.Resource, address.Selector.Operation) + ":" + canonicalInputMatch(address.Selector.InputMatch)
		operation, ok := want[key]
		if !ok || strings.ToUpper(address.Method) != strings.ToUpper(operation.Route.Method) {
			return errors.New("Sandbox receipt contains an unexpected operation address")
		}
		parsed, err := url.Parse(address.URL)
		if err != nil || parsed.Scheme != "http" || strings.ToLower(parsed.Hostname()) != strings.ToLower(operation.Route.Host) || parsed.Port() != "18080" || parsed.Path != operation.Route.Path || parsed.RawQuery != operation.Route.Query {
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
