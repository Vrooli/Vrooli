package skills

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/researchhome"
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
	if req.ExperimentID != "" || req.VariantID != "" {
		http.Error(w, "Skill experiments are no longer supported", http.StatusBadRequest)
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
		if researchhome.IsID(identifier) {
			if readSkill, issue := h.readResearchSkill(identifier, req.Variables); issue != nil {
				resp.Missing = append(resp.Missing, *issue)
			} else {
				responses = append(responses, readSkill)
			}
			continue
		}
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

	identity := h.verifiedWorkflowIdentity(r)
	// Counted at the read, not at `skill use`, so every consumer — CLI, UI,
	// MCP, other scenarios — is counted exactly once per resolved skill.
	readIDs := make([]string, 0, len(responses))
	for _, response := range responses {
		readIDs = append(readIDs, response.ID)
	}
	if identity != nil {
		h.readRecorder.RecordWithRunID(r, readIDs, identity.RunID)
	} else {
		h.readRecorder.Record(r, readIDs)
	}

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

func (h *Handlers) verifiedWorkflowIdentity(r *http.Request) *VerifiedWorkflowIdentity {
	if h.identityVerifier == nil {
		return nil
	}
	identity, _ := h.identityVerifier.VerifyWorkflowIdentity(r.Context(), r.Header.Get(agentIdentityHeader))
	return identity
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
	folders, err := loadReaderMetadata(store)
	if err != nil {
		return nil, err
	}
	var indexed []indexedSkill
	for _, folder := range Folders {
		for _, skill := range folders[folder] {
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

func loadReaderMetadata(source SkillStore) (map[string][]Metadata, error) {
	if batch, ok := source.(MetadataBatchStore); ok {
		return batch.LoadMetadataBatch(Folders)
	}
	folders := make(map[string][]Metadata, len(Folders))
	for _, folder := range Folders {
		metadata, err := source.LoadMetadata(folder)
		if err != nil {
			return nil, err
		}
		folders[folder] = metadata
	}
	return folders, nil
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

	content = StripFrontmatter(content)
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
