package agentharness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SandboxChange is one entry of a workspace-sandbox change record.
type SandboxChange struct {
	Path       string `json:"filePath"`
	ChangeType string `json:"changeType"`
}

// sandboxFetchTimeout keeps the hook fast: a slow or absent sandbox service
// means this resolver does not vouch, and the other resolvers and the existing
// guard decide.
const sandboxFetchTimeout = 800 * time.Millisecond

// FetchSandboxChanges reads the sandbox's live change record from the
// workspace-sandbox API (GET /api/v1/sandboxes/{id}/diff). baseURL empty
// means "not reachable from this process", which is an error, not an empty
// record.
func FetchSandboxChanges(ctx context.Context, baseURL, sandboxID string) ([]SandboxChange, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(sandboxID) == "" {
		return nil, errors.New("the workspace-sandbox service address or sandbox id is unknown")
	}
	ctx, cancel := context.WithTimeout(ctx, sandboxFetchTimeout)
	defer cancel()
	endpoint := baseURL + "/api/v1/sandboxes/" + url.PathEscape(sandboxID) + "/diff"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workspace-sandbox answered %d", response.StatusCode)
	}
	var body struct {
		Files []SandboxChange `json:"files"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 16<<20)).Decode(&body); err != nil {
		return nil, err
	}
	return body.Files, nil
}

// SandboxResolver vouches for paths a workspace sandbox recorded as added. The
// sandbox sees every write the agent makes, so this is the strongest source,
// but it exists only for runs launched into a sandbox.
type SandboxResolver struct {
	merged string
	added  map[string]bool
	other  map[string]bool
}

// NewSandboxResolver builds the resolver from the sandbox's merged workspace
// path and its change record.
func NewSandboxResolver(merged string, changes []SandboxChange) SandboxResolver {
	r := SandboxResolver{merged: resolveExisting(merged), added: map[string]bool{}, other: map[string]bool{}}
	for _, change := range changes {
		rel := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(change.Path, "/")))
		if strings.EqualFold(change.ChangeType, "added") {
			r.added[rel] = true
		} else {
			r.other[rel] = true
		}
	}
	return r
}

func (SandboxResolver) Name() string { return "workspace-sandbox" }

// Vouch says yes for a path inside the merged workspace that the record lists
// as added. A recursive query needs every file currently below the path to be
// listed as added, so a folder the sandbox only added to is not vouched.
func (r SandboxResolver) Vouch(q CreationQuery) (Vouch, bool) {
	if r.merged == "" {
		return Vouch{}, false
	}
	path := resolveExisting(q.Path)
	rel, err := filepath.Rel(r.merged, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return Vouch{}, false
	}
	info, statErr := os.Lstat(path)
	if statErr != nil {
		return Vouch{}, false
	}
	if !info.IsDir() {
		if r.added[rel] {
			return Vouch{Certainty: CertaintyRecorded, Source: "workspace-sandbox", Detail: "recorded as added in this run's sandbox"}, true
		}
		return Vouch{}, false
	}
	if r.other[rel] {
		return Vouch{}, false
	}
	budget, files, complete := treeBudget, 0, true
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			complete = false
			return fs.SkipAll
		}
		if budget--; budget < 0 {
			complete = false
			return fs.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		sub, relErr := filepath.Rel(r.merged, p)
		if relErr != nil || !r.added[sub] {
			complete = false
			return fs.SkipAll
		}
		files++
		return nil
	})
	if !complete || (files == 0 && !r.added[rel]) {
		return Vouch{}, false
	}
	return Vouch{Certainty: CertaintyRecorded, Source: "workspace-sandbox", Detail: "every file below it is recorded as added in this run's sandbox"}, true
}
