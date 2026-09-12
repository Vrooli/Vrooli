package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type InventoryRequest struct {
	Roots    []string `json:"roots"`
	Includes []string `json:"includes,omitempty"`
	Excludes []string `json:"excludes,omitempty"`
	MaxFiles int      `json:"max_files,omitempty"`
	MaxBytes int64    `json:"max_bytes,omitempty"`
	Revision string   `json:"revision,omitempty"`
}

type InventoryResponse struct {
	Status     string               `json:"status"`
	Revision   string               `json:"revision"`
	Candidates []InventoryCandidate `json:"candidates"`
	Truncated  bool                 `json:"truncated"`
	Warnings   []string             `json:"warnings,omitempty"`
}

type InventoryCandidate struct {
	Path               string   `json:"path"`
	ArtifactClass      string   `json:"artifact_class"`
	Sha256             string   `json:"sha256"`
	Bytes              int64    `json:"bytes"`
	OwnerHint          string   `json:"owner_hint,omitempty"`
	Action             string   `json:"action"`
	PortabilitySignals []string `json:"portability_signals,omitempty"`
	ReferenceStatus    string   `json:"reference_status"`
	EvidenceStanding   string   `json:"evidence_standing"`
	InspectHandle      string   `json:"inspect_handle"`
	ProposalHandle     string   `json:"proposal_handle"`
	Excerpt            string   `json:"excerpt,omitempty"`
}

type ProposalRequest struct {
	Candidates    []InventoryCandidate `json:"candidates"`
	ReaderTask    string               `json:"reader_task,omitempty"`
	Authorization string               `json:"authorization,omitempty"`
}

type ProposalResponse struct {
	Status    string                `json:"status"`
	Proposals []DispositionProposal `json:"proposals"`
	Warnings  []string              `json:"warnings,omitempty"`
}

type DispositionProposal struct {
	Source                string   `json:"source"`
	SourceSha256          string   `json:"source_sha256"`
	Action                string   `json:"action"`
	Decision              string   `json:"decision"`
	Confidence            float64  `json:"confidence"`
	Uncertainty           []string `json:"uncertainty"`
	Citations             []string `json:"citations"`
	Preservation          []string `json:"preservation"`
	AuthorizationRequired string   `json:"authorization_required"`
	PromptInjectionSignal bool     `json:"prompt_injection_signal"`
}

type RouteRequest struct {
	Proposal             DispositionProposal `json:"proposal"`
	ExpectedSourceSha256 string              `json:"expected_source_sha256"`
	Authorization        string              `json:"authorization,omitempty"`
	IdempotencyKey       string              `json:"idempotency_key"`
	DryRun               bool                `json:"dry_run"`
}

type RouteResponse struct {
	Status    string `json:"status"`
	Owner     string `json:"owner"`
	Operation string `json:"operation"`
	RequestID string `json:"request_id"`
	Receipt   string `json:"receipt"`
	Reason    string `json:"reason,omitempty"`
}

func routeDisposition(req RouteRequest) RouteResponse {
	payload, _ := json.Marshal(req)
	digest := sha256.Sum256(payload)
	result := RouteResponse{Status: "dry_run", Owner: "plan-manager", Operation: "candidate-artifact-review", RequestID: req.IdempotencyKey, Receipt: "sha256:" + hex.EncodeToString(digest[:])}
	if req.IdempotencyKey == "" {
		result.Status, result.Reason = "refused", "idempotency_key is required"
		return result
	}
	if req.ExpectedSourceSha256 == "" || req.ExpectedSourceSha256 != req.Proposal.SourceSha256 {
		result.Status, result.Reason = "refused", "source revision/hash precondition is missing or stale"
		return result
	}
	if req.Proposal.Action == "unresolved" {
		result.Status, result.Reason = "refused", "owner and disposition remain unresolved"
		return result
	}
	if !req.DryRun && req.Authorization != "owner" {
		result.Status, result.Reason = "refused", "owner authorization is required"
		return result
	}
	if !req.DryRun {
		result.Status = "accepted"
	}
	return result
}

func proposeInventoryDispositions(req ProposalRequest) ProposalResponse {
	result := ProposalResponse{Status: "ok", Proposals: make([]DispositionProposal, 0, len(req.Candidates))}
	if len(req.Candidates) > 100 {
		req.Candidates = req.Candidates[:100]
		result.Status = "partial"
		result.Warnings = append(result.Warnings, "proposal input capped at 100 candidates")
	}
	for _, candidate := range req.Candidates {
		proposal := DispositionProposal{Source: candidate.Path, SourceSha256: candidate.Sha256, Action: "unresolved", Decision: "proposed", Confidence: 0.2, Uncertainty: []string{"owner authority is unresolved", "proposal is not an accepted action"}, Citations: []string{"path:" + candidate.Path + "#sha256=" + candidate.Sha256}, Preservation: []string{"preserve source bytes and incoming references until owner review"}, AuthorizationRequired: "owner"}
		lowerExcerpt := strings.ToLower(candidate.Excerpt)
		if strings.Contains(lowerExcerpt, "ignore previous") || strings.Contains(lowerExcerpt, "system prompt") || strings.Contains(lowerExcerpt, "override instructions") {
			proposal.PromptInjectionSignal = true
			proposal.Confidence = 0
			proposal.Uncertainty = append(proposal.Uncertainty, "retrieved text contains instruction-like content and was treated as untrusted evidence")
		}
		if candidate.OwnerHint != "" {
			proposal.Uncertainty = append(proposal.Uncertainty, "semantic authority and duplicate/conflict analysis require owner review")
		}
		if candidate.ArtifactClass == "generated_html" || candidate.ArtifactClass == "report" {
			proposal.Action = "retain"
			proposal.Uncertainty = append(proposal.Uncertainty, "artifact may be derived output; source disposition is unresolved")
		}
		result.Proposals = append(result.Proposals, proposal)
	}
	return result
}

func buildCandidateInventory(repoRoot string, req InventoryRequest) (InventoryResponse, error) {
	root, err := filepath.Abs(strings.TrimSpace(repoRoot))
	if err != nil || strings.TrimSpace(repoRoot) == "" {
		return InventoryResponse{}, fmt.Errorf("repository root is required")
	}
	if len(req.Roots) == 0 {
		req.Roots = []string{"docs", "scenarios"}
	}
	if len(req.Roots) > 16 {
		return InventoryResponse{}, fmt.Errorf("at most 16 inventory roots are allowed")
	}
	maxFiles := req.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 100
	}
	if maxFiles > 500 {
		maxFiles = 500
	}
	maxBytes := req.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}
	if maxBytes > 8<<20 {
		maxBytes = 8 << 20
	}
	response := InventoryResponse{Status: "ok", Revision: req.Revision}
	if response.Revision == "" {
		response.Revision = strings.TrimSpace(os.Getenv("GIT_COMMIT"))
		if response.Revision == "" {
			response.Revision = "working-tree"
		}
	}
	seen := map[string]struct{}{}
	for _, rawRoot := range req.Roots {
		relRoot, err := safeInventoryPath(rawRoot)
		if err != nil {
			return InventoryResponse{}, err
		}
		walkRoot := filepath.Join(root, filepath.FromSlash(relRoot))
		walkErr := filepath.WalkDir(walkRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				response.Warnings = append(response.Warnings, fmt.Sprintf("%s: %v", relRoot, walkErr))
				return fs.SkipDir
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if !inventoryIncluded(rel, req.Includes, req.Excludes) {
				return nil
			}
			if _, ok := seen[rel]; ok {
				return nil
			}
			seen[rel] = struct{}{}
			if len(response.Candidates) >= maxFiles {
				response.Truncated = true
				return fs.SkipAll
			}
			info, err := entry.Info()
			if err != nil {
				response.Warnings = append(response.Warnings, rel+": stat unavailable")
				return nil
			}
			candidate := InventoryCandidate{Path: rel, ArtifactClass: inventoryArtifactClass(rel), Bytes: info.Size(), Action: "inspect", ReferenceStatus: "unresolved", EvidenceStanding: "unknown", InspectHandle: "inspect:" + rel, ProposalHandle: "propose:" + rel}
			if info.Size() > maxBytes {
				candidate.PortabilitySignals = []string{"over_output_bound"}
				response.Candidates = append(response.Candidates, candidate)
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				candidate.PortabilitySignals = []string{"content_unavailable"}
				response.Candidates = append(response.Candidates, candidate)
				return nil
			}
			digest := sha256.Sum256(data)
			candidate.Sha256 = "sha256:" + hex.EncodeToString(digest[:])
			candidate.PortabilitySignals = inventoryPortabilitySignals(rel, data)
			response.Candidates = append(response.Candidates, candidate)
			return nil
		})
		if walkErr != nil && walkErr != fs.SkipAll {
			response.Warnings = append(response.Warnings, walkErr.Error())
		}
		if response.Truncated {
			break
		}
	}
	sort.Slice(response.Candidates, func(i, j int) bool { return response.Candidates[i].Path < response.Candidates[j].Path })
	if response.Truncated || len(response.Warnings) > 0 {
		response.Status = "partial"
	}
	return response, nil
}

func safeInventoryPath(raw string) (string, error) {
	p := filepath.ToSlash(strings.TrimSpace(raw))
	if p == "" || filepath.IsAbs(p) || p == "." || strings.HasPrefix(p, "../") || strings.Contains(p, "/../") || strings.ContainsAny(p, "\\:\x00") {
		return "", fmt.Errorf("unsafe inventory root %q", raw)
	}
	return p, nil
}

func inventoryIncluded(path string, includes, excludes []string) bool {
	matchAny := func(patterns []string) bool {
		for _, pattern := range patterns {
			if ok, _ := filepath.Match(filepath.ToSlash(pattern), path); ok {
				return true
			}
			if strings.HasPrefix(path, strings.TrimSuffix(filepath.ToSlash(pattern), "/*")+"/") {
				return true
			}
		}
		return false
	}
	if len(includes) > 0 && !matchAny(includes) {
		return false
	}
	return !matchAny(excludes)
}

func inventoryArtifactClass(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm"):
		return "generated_html"
	case strings.Contains(lower, "/report") || strings.Contains(lower, "report."):
		return "report"
	case strings.Contains(lower, "evidence") || strings.Contains(lower, "artifact"):
		return "evidence_artifact"
	case strings.Contains(lower, "plan") || strings.Contains(lower, "phase"):
		return "plan_phase"
	case strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".mdx"):
		return "ordinary_doc"
	default:
		return "other"
	}
}

func inventoryPortabilitySignals(path string, data []byte) []string {
	text := string(data)
	lower := strings.ToLower(path + "\n" + text)
	var signals []string
	if strings.HasSuffix(strings.ToLower(path), ".html") {
		signals = append(signals, "generated_html")
	}
	if strings.Contains(text, "/home/") || strings.Contains(text, "C:\\") {
		signals = append(signals, "absolute_path")
	}
	if strings.Contains(lower, "localhost:") || strings.Contains(lower, "127.0.0.1:") {
		signals = append(signals, "machine_specific_endpoint")
	}
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
		signals = append(signals, "external_reference")
	}
	return signals
}

func encodeInventory(v any) ([]byte, error) { return json.Marshal(v) }
