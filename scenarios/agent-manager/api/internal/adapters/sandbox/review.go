package sandbox

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
)

// ReviewProvider is the retained-input capability of the existing sandbox owner.
// Providers without it cannot substitute a fresh canonical tree for review.
type ReviewProvider interface {
	PrepareReview(context.Context, uuid.UUID, uuid.UUID, []string) (*ReviewInput, error)
}

type ReviewInput struct {
	SandboxID uuid.UUID `json:"sandboxId"`
	RequestID uuid.UUID `json:"requestId"`
	SHA256    string    `json:"sha256"`
	Root      string    `json:"root"`
}

func (p *WorkspaceSandboxProvider) PrepareReview(ctx context.Context, sandboxID, requestID uuid.UUID, paths []string) (*ReviewInput, error) {
	base := fmt.Sprintf("/api/v1/sandboxes/%s/reviews", sandboxID)
	read, err := p.doRequest(ctx, http.MethodGet, base+"/"+requestID.String(), nil)
	if err != nil {
		return nil, err
	}
	exists := read.StatusCode == http.StatusOK
	if !exists && read.StatusCode != http.StatusNotFound {
		defer read.Body.Close()
		return nil, p.parseError("review_read", &sandboxID, read)
	}
	read.Body.Close()
	if !exists {
		if err := p.Stop(ctx, sandboxID); err != nil {
			return nil, err
		}
		processes, err := p.ListProcesses(ctx, sandboxID)
		if err != nil {
			return nil, err
		}
		// An empty inventory after a provider restart is not proof of drain.
		if len(processes) == 0 {
			return nil, fmt.Errorf("review source lacks retained process exit evidence")
		}
		for _, process := range processes {
			if process.ExitCode == nil {
				return nil, fmt.Errorf("review source process %d has no terminal receipt", process.PID)
			}
		}
	}
	// Same-key capture validates the literal selection on replay and returns
	// retained bytes even if the old worker overlay has since disappeared.
	captured, err := p.doRequest(ctx, http.MethodPost, base, map[string]any{"requestId": requestID, "paths": paths})
	if err != nil {
		return nil, err
	}
	defer captured.Body.Close()
	if captured.StatusCode != http.StatusOK {
		return nil, p.parseError("review_capture", &sandboxID, captured)
	}
	var input ReviewInput
	if err := json.NewDecoder(captured.Body).Decode(&input); err != nil {
		return nil, err
	}
	digest, err := hex.DecodeString(input.SHA256)
	if err != nil || len(digest) != 32 || input.SandboxID != sandboxID || input.RequestID != requestID {
		return nil, fmt.Errorf("sandbox owner returned different review identity")
	}
	materialized, err := p.doRequest(ctx, http.MethodPost, base+"/"+requestID.String()+"/workspace", map[string]string{"expectedSha256": input.SHA256})
	if err != nil {
		return nil, err
	}
	defer materialized.Body.Close()
	if materialized.StatusCode != http.StatusOK {
		return nil, p.parseError("review_materialize", &sandboxID, materialized)
	}
	var workspace ReviewInput
	if err := json.NewDecoder(materialized.Body).Decode(&workspace); err != nil {
		return nil, err
	}
	if workspace.SHA256 != input.SHA256 || !filepath.IsAbs(workspace.Root) {
		return nil, fmt.Errorf("sandbox owner returned invalid review workspace")
	}
	input.Root = workspace.Root
	return &input, nil
}
