package skills

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"prompt-manager/internal/store"
)

type indexedSkill struct {
	meta     Metadata
	folder   string
	filename string
	filePath string
}

// Read handles POST /skills/read - resolves and returns multiple skills.
func (h *Handlers) Read(w http.ResponseWriter, r *http.Request) {
	var req ReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Identifiers) == 0 {
		http.Error(w, "Identifiers are required", http.StatusBadRequest)
		return
	}

	resolve := strings.ToLower(strings.TrimSpace(req.Resolve))
	if resolve == "" {
		resolve = "auto"
	}
	if !isValidResolveMode(resolve) {
		http.Error(w, "Resolve must be 'auto', 'id', 'file', or 'name'", http.StatusBadRequest)
		return
	}

	output := normalizeReadOutput(req.Output, len(req.Identifiers))
	if output == "" {
		http.Error(w, "Output must be 'skills', 'combined', 'both', or 'auto'", http.StatusBadRequest)
		return
	}

	allowMissing := true
	if req.AllowMissing != nil {
		allowMissing = *req.AllowMissing
	}

	skillStore := h.storeFor(r.Context())
	indexed, err := loadIndexedSkills(skillStore)
	if err != nil {
		http.Error(w, "Failed to load skills", http.StatusInternalServerError)
		return
	}

	resp := ReadResponse{Resolve: resolve, Output: output}
	var responses []Response

	for _, identifier := range req.Identifiers {
		matches := resolveIdentifier(identifier, resolve, indexed)
		switch len(matches) {
		case 0:
			resp.Missing = append(resp.Missing, ReadIssue{
				Identifier: identifier,
				Reason:     "not_found",
			})
		case 1:
			readSkill, err := h.buildReadResponse(skillStore, matches[0], req.Variables)
			if err != nil {
				http.Error(w, "Failed to load skill content", http.StatusInternalServerError)
				return
			}
			responses = append(responses, readSkill)
		default:
			resp.Ambiguous = append(resp.Ambiguous, ReadAmbiguous{
				Identifier: identifier,
				Candidates: buildCandidates(matches),
			})
		}
	}

	// Variant-aware selection requires an explicit experiment ID. Organic reads
	// remain observational and must never be silently assigned treatment.
	if len(responses) > 0 {
		exp, err := h.resolveExperiment(r.Context(), req, responses[0].ID)
		if err != nil {
			http.Error(w, "Experiment error: "+err.Error(), http.StatusBadRequest)
			return
		}
		if exp != nil {
			selectedVariantID, variantContent, selErr := h.selectVariant(r.Context(), exp, req.VariantID)
			switch {
			case selErr != nil && req.ExperimentID != "":
				http.Error(w, "Experiment error: "+selErr.Error(), http.StatusBadRequest)
				return
			case selErr != nil:
				// Blind serving: swallow and serve the original content.
			default:
				if variantContent != nil {
					// Apply variable substitution to variant content
					content := *variantContent
					if len(req.Variables) > 0 {
						content = SubstituteVariables(content, req.Variables)
					}
					responses[0].Content = content
					responses[0].Variables = ExtractVariables(*variantContent)
					responses[0].ContentHash = contentSHA256(*variantContent)
				}
				resp.SelectedVariantID = selectedVariantID
				resp.ExperimentID = exp.ID
				// Record the serve event so the served variant can be attributed
				// to an outcome later (best-effort; never breaks the read).
				_ = h.experimentStore.RecordServe(r.Context(), store.ExperimentServe{
					ExperimentID: exp.ID,
					SkillID:      exp.SkillID,
					VariantID:    selectedVariantID,
					Source:       req.Source,
				})
			}
		}
	}
	// An active workflow run may read additional skills after receiving its
	// assigned prompt. Provenance comes exclusively from verified claims; an
	// unavailable or invalid token is retained as an unattributed observation.
	h.recordExposureReceipts(r, responses)
	// Counted at the read, not at `skill use`, so every consumer — CLI, UI,
	// MCP, other scenarios — is counted exactly once per resolved skill.
	readIDs := make([]string, 0, len(responses))
	for _, response := range responses {
		readIDs = append(readIDs, response.ID)
	}
	h.readRecorder.Record(r, readIDs)

	if !allowMissing && (len(resp.Missing) > 0 || len(resp.Ambiguous) > 0) {
		status := http.StatusNotFound
		if len(resp.Ambiguous) > 0 {
			status = http.StatusConflict
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Handle scope inclusion
	var scopeSkill *Response
	if req.Scope != "" {
		// Explicit scope requested
		scopeMatches := resolveIdentifier(req.Scope, "id", indexed)
		if len(scopeMatches) == 1 {
			s, err := h.buildReadResponse(skillStore, scopeMatches[0], req.Variables)
			if err == nil {
				scopeSkill = &s
			}
		}
	} else if req.WithScope && len(responses) > 0 {
		// Use default scope from first skill with one
		for _, skill := range responses {
			if skill.DefaultScope != "" {
				scopeMatches := resolveIdentifier(skill.DefaultScope, "id", indexed)
				if len(scopeMatches) == 1 {
					s, err := h.buildReadResponse(skillStore, scopeMatches[0], req.Variables)
					if err == nil {
						scopeSkill = &s
					}
				}
				break
			}
		}
	}
	resp.ScopeSkill = scopeSkill

	if outputIncludesSkills(output) {
		resp.Skills = responses
	}

	if outputIncludesCombined(output) {
		// Include scope skill at the beginning of combined output
		toRender := responses
		if scopeSkill != nil {
			toRender = append([]Response{*scopeSkill}, responses...)
		}
		combined, normalizedFormat, err := RenderCombined(toRender, req.Format)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp.Combined = combined
		resp.CombinedHash = contentSHA256(combined)
		resp.SkillCount = len(toRender)
		resp.TotalTokens = (len(combined) + 3) / 4
		resp.Format = normalizedFormat
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) recordExposureReceipts(r *http.Request, responses []Response) {
	if h.experimentStore == nil || len(responses) == 0 {
		return
	}
	exposures, ok := h.experimentStore.(store.ExperimentExposureStore)
	if !ok {
		return
	}
	var identity *VerifiedWorkflowIdentity
	if h.identityVerifier != nil {
		identity, _ = h.identityVerifier.VerifyWorkflowIdentity(r.Context(), r.Header.Get(agentIdentityHeader))
	}
	provenance, runID, experimentID, variantID, executionID, nodeID, attemptID := "unattributed", "", "", "", "", "", ""
	if identity != nil {
		meta := identity.Meta
		runID, experimentID, variantID = identity.RunID, strings.TrimSpace(meta["workflowExperimentId"]), strings.TrimSpace(meta["workflowVariantId"])
		executionID, nodeID, attemptID = strings.TrimSpace(meta["workflowExecutionId"]), strings.TrimSpace(meta["workflowNodeId"]), strings.TrimSpace(meta["workflowAttemptId"])
		provenance = "verified-run"
	}
	readKey := strings.TrimSpace(r.Header.Get("X-Experiment-Read-Receipt-ID"))
	if readKey == "" {
		readKey = time.Now().UTC().Format(time.RFC3339Nano)
	}
	for _, response := range responses {
		keyMaterial := strings.Join([]string{experimentID, runID, response.ID, readKey}, "|")
		keyHash := sha256.Sum256([]byte(keyMaterial))
		_ = exposures.RecordExposure(r.Context(), store.ExperimentExposure{ExperimentID: experimentID, VariantID: variantID, RunID: runID, ExecutionID: executionID, NodeID: nodeID, AttemptID: attemptID, ReadSkillID: response.ID, Provenance: provenance, IdempotencyKey: fmt.Sprintf("exposure/%x", keyHash)})
	}
}

func isValidResolveMode(mode string) bool {
	switch mode {
	case "auto", "id", "file", "name":
		return true
	default:
		return false
	}
}

func loadIndexedSkills(store SkillStore) ([]indexedSkill, error) {
	var indexed []indexedSkill
	for _, folder := range Folders {
		skills, err := store.LoadMetadata(folder)
		if err != nil {
			return nil, err
		}
		for _, skill := range skills {
			filename := skill.File
			prefix := folder + "/"
			filename = strings.TrimPrefix(filename, prefix)
			indexed = append(indexed, indexedSkill{
				meta:     skill,
				folder:   folder,
				filename: filename,
				filePath: filepath.ToSlash(prefix + filename),
			})
		}
	}
	return indexed, nil
}

func resolveIdentifier(identifier, mode string, skills []indexedSkill) []indexedSkill {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil
	}

	switch mode {
	case "id":
		return resolveByID(identifier, skills)
	case "file":
		return resolveByFile(identifier, skills)
	case "name":
		return resolveByName(identifier, skills)
	default:
		if matches := resolveByID(identifier, skills); len(matches) > 0 {
			return matches
		}
		if matches := resolveByFile(identifier, skills); len(matches) > 0 {
			return matches
		}
		return resolveByName(identifier, skills)
	}
}

func resolveByID(identifier string, skills []indexedSkill) []indexedSkill {
	var matches []indexedSkill
	for _, skill := range skills {
		if strings.EqualFold(skill.meta.ID, identifier) {
			matches = append(matches, skill)
		}
	}
	return matches
}

func resolveByName(identifier string, skills []indexedSkill) []indexedSkill {
	var matches []indexedSkill
	for _, skill := range skills {
		if strings.EqualFold(skill.meta.Name, identifier) {
			matches = append(matches, skill)
		}
	}
	return matches
}

func resolveByFile(identifier string, skills []indexedSkill) []indexedSkill {
	normalized := normalizeFileIdentifier(identifier)
	if normalized == "" {
		return nil
	}

	hasPath := strings.Contains(normalized, "/")
	normalizedNoExt := stripMDExt(normalized)

	var matches []indexedSkill
	for _, skill := range skills {
		if hasPath {
			if strings.EqualFold(skill.filePath, normalized) ||
				strings.EqualFold(stripMDExt(skill.filePath), normalizedNoExt) {
				matches = append(matches, skill)
			}
			continue
		}
		if strings.EqualFold(skill.filename, normalized) ||
			strings.EqualFold(stripMDExt(skill.filename), normalizedNoExt) {
			matches = append(matches, skill)
		}
	}
	return matches
}

func normalizeFileIdentifier(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = filepath.ToSlash(s)
	s = strings.TrimPrefix(s, "./")
	s = strings.TrimPrefix(s, "/")
	if idx := strings.LastIndex(s, "/skills/"); idx != -1 {
		s = s[idx+len("/skills/"):]
	}
	s = strings.TrimPrefix(s, "skills/")
	return strings.TrimPrefix(s, "/")
}

func stripMDExt(path string) string {
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		return path[:len(path)-3]
	}
	return path
}

func normalizeReadOutput(output string, identifierCount int) string {
	out := strings.ToLower(strings.TrimSpace(output))
	if out == "" || out == "auto" {
		if identifierCount > 1 {
			return "combined"
		}
		return "skills"
	}
	if out == "skills" || out == "combined" || out == "both" {
		return out
	}
	return ""
}

func outputIncludesSkills(output string) bool {
	return output == "skills" || output == "both"
}

func outputIncludesCombined(output string) bool {
	return output == "combined" || output == "both"
}

func (h *Handlers) buildReadResponse(store SkillStore, skill indexedSkill, variables map[string]string) (Response, error) {
	resp := h.toResponse(skill.meta)
	resp.Folder = skill.folder
	resp.File = skill.filename

	content, err := store.GetContent(skill.folder, skill.filename)
	if err != nil {
		return Response{}, err
	}

	// Extract variables from original content before substitution
	resp.Variables = ExtractVariables(content)
	resp.ContentHash = contentSHA256(content)

	// Apply variable substitution if values provided
	if len(variables) > 0 {
		content = SubstituteVariables(content, variables)
	}
	resp.Content = content

	return resp, nil
}

func contentSHA256(content string) string {
	digest := sha256.Sum256([]byte(content))
	return fmt.Sprintf("sha256:%x", digest[:])
}

// resolveExperiment returns the experiment that should serve a variant for the
// given skill read, or nil if none applies. An explicit experimentId is
// validated (must be running and target the skill) and surfaces its error.
func (h *Handlers) resolveExperiment(ctx context.Context, req ReadRequest, skillID string) (*store.Experiment, error) {
	if h.experimentStore == nil || h.variantStore == nil {
		if req.ExperimentID != "" {
			return nil, fmt.Errorf("experiment support not configured")
		}
		return nil, nil
	}

	if req.ExperimentID != "" {
		exp, err := h.experimentStore.Get(ctx, req.ExperimentID)
		if err != nil {
			return nil, fmt.Errorf("experiment not found: %s", req.ExperimentID)
		}
		if exp.Status != store.ExperimentStatusRunning {
			return nil, fmt.Errorf("experiment %s is not running (status: %s)", req.ExperimentID, exp.Status)
		}
		if exp.SkillID != skillID {
			return nil, fmt.Errorf("experiment %s targets skill %s, not %s", req.ExperimentID, exp.SkillID, skillID)
		}
		return exp, nil
	}

	return nil, nil
}

// selectVariant picks a variant from a running experiment via a weighted walk.
// Returns the selected variant ID and its content (nil for control, which serves
// the original SKILL.md).
func (h *Handlers) selectVariant(ctx context.Context, exp *store.Experiment, requestedVariantID string) (string, *string, error) {
	if len(exp.Arms) == 0 {
		return "", nil, fmt.Errorf("experiment %s has no arms", exp.ID)
	}

	selectedArm := exp.Arms[len(exp.Arms)-1] // fallback to last arm
	if requestedVariantID != "" {
		found := false
		for _, arm := range exp.Arms {
			if arm.VariantID == requestedVariantID {
				selectedArm, found = arm, true
				break
			}
		}
		if !found {
			return "", nil, fmt.Errorf("variant %s is not an arm of experiment %s", requestedVariantID, exp.ID)
		}
	} else {
		roll := rand.Float64()
		var cumulative float64
		for _, arm := range exp.Arms {
			cumulative += arm.Weight
			if roll < cumulative {
				selectedArm = arm
				break
			}
		}
	}

	// If control, use original SKILL.md (no content override)
	if selectedArm.VariantID == store.ControlVariantID {
		return store.ControlVariantID, nil, nil
	}

	// Load variant content
	_, content, err := h.variantStore.GetWithContent(ctx, exp.SkillID, selectedArm.VariantID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load variant %s: %w", selectedArm.VariantID, err)
	}

	return selectedArm.VariantID, &content, nil
}

func buildCandidates(skills []indexedSkill) []ReadCandidate {
	candidates := make([]ReadCandidate, 0, len(skills))
	for _, skill := range skills {
		candidates = append(candidates, ReadCandidate{
			ID:     skill.meta.ID,
			Name:   skill.meta.Name,
			File:   skill.filename,
			Folder: skill.folder,
		})
	}
	return candidates
}
