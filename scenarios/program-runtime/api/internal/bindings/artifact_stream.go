package bindings

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const testGenieArtifactBindingID = "test-genie/runs/artifact"

type TestGenieArtifact struct {
	Target           string
	RunID            string
	ArtifactID       string
	Kind             string
	SizeBytes        int64
	Legacy           bool
	AccessPath       string
	AccessCapability string
}

// OpenTestGenieArtifactStream first resolves the opaque artifact ID through
// Test Genie's declared metadata RPC, then opens its exact run-scoped stream
// path. The stream is returned to the host effect owner; bytes never enter a
// Program Runtime JSON binding response.
func (r *Registry) OpenTestGenieArtifactStream(ctx context.Context, _program, _digest, target, runID, artifactID string, grants []string, metadata InvocationMetadata, client *http.Client) (TestGenieArtifact, io.ReadCloser, error) {
	target = strings.TrimSpace(target)
	runID = strings.TrimSpace(runID)
	artifactID = strings.TrimSpace(artifactID)
	if target == "" || runID == "" || artifactID == "" {
		return TestGenieArtifact{}, nil, errors.New("target, run ID, and artifact ID are required")
	}
	if _, ok := r.Binding(testGenieArtifactBindingID); !ok {
		return TestGenieArtifact{}, nil, errors.New("Test Genie artifact metadata capability is unavailable")
	}
	result, err := r.Execute(ctx, testGenieArtifactBindingID,
		map[string]any{"target": target, "run_id": runID, "artifact_id": artifactID}, grants, false, metadata, client)
	if err != nil {
		return TestGenieArtifact{}, nil, fmt.Errorf("resolve Test Genie artifact metadata: %w", err)
	}
	artifactValue, ok := result["artifact"].(map[string]any)
	if !ok {
		return TestGenieArtifact{}, nil, errors.New("Test Genie metadata response has no artifact object")
	}
	ref := TestGenieArtifact{
		Target: target, RunID: runID, ArtifactID: artifactID,
		Kind:             stringValue(artifactValue["kind"]),
		SizeBytes:        int64Value(artifactValue["sizeBytes"]),
		Legacy:           boolValue(result["legacyDiscovered"]),
		AccessPath:       stringValue(artifactValue["accessPath"]),
		AccessCapability: stringValue(artifactValue["accessCapability"]),
	}
	if stringValue(artifactValue["id"]) != artifactID {
		return TestGenieArtifact{}, nil, errors.New("Test Genie metadata returned a different artifact ID")
	}
	if ref.AccessCapability != "ARTIFACT_ACCESS_CAPABILITY_STREAM" || ref.SizeBytes < 0 {
		return TestGenieArtifact{}, nil, errors.New("Test Genie artifact lacks stream capability or a valid declared size")
	}
	wantPath := "/api/v1/scenarios/" + url.PathEscape(target) + "/runs/" + url.PathEscape(runID) + "/artifacts/" + url.PathEscape(artifactID)
	parsed, err := url.Parse(ref.AccessPath)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.EscapedPath() != wantPath {
		return TestGenieArtifact{}, nil, errors.New("Test Genie artifact access path does not match the requested run and ID")
	}
	r = r.active()
	resolver := r.resolver
	if resolver == nil {
		resolver = ResolveTargetURLDefault
	}
	base, err := resolver(ctx, "test-genie")
	if err != nil {
		return TestGenieArtifact{}, nil, fmt.Errorf("resolve Test Genie stream owner: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+wantPath, nil)
	if err != nil {
		return TestGenieArtifact{}, nil, err
	}
	if r.requestPreparer != nil {
		if err := r.requestPreparer(ctx, testGenieArtifactBindingID, request); err != nil {
			return TestGenieArtifact{}, nil, fmt.Errorf("prepare Test Genie artifact stream: %w", err)
		}
	}
	if client == nil {
		client = &http.Client{}
	}
	strictClient := *client
	strictClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := strictClient.Do(request)
	if err != nil {
		return TestGenieArtifact{}, nil, fmt.Errorf("open Test Genie artifact stream: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return TestGenieArtifact{}, nil, fmt.Errorf("Test Genie artifact stream returned %s", response.Status)
	}
	return ref, response.Body, nil
}

func stringValue(value any) string { result, _ := value.(string); return result }

func boolValue(value any) bool { result, _ := value.(bool); return result }

func int64Value(value any) int64 {
	switch number := value.(type) {
	case string:
		parsed, err := strconv.ParseInt(number, 10, 64)
		if err == nil {
			return parsed
		}
	case int64:
		return number
	case int:
		return int64(number)
	case float64:
		if number == float64(int64(number)) {
			return int64(number)
		}
	}
	return -1
}
