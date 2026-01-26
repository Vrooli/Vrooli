package skills

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
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

	allowMissing := true
	if req.AllowMissing != nil {
		allowMissing = *req.AllowMissing
	}

	indexed, err := loadIndexedSkills(h.store)
	if err != nil {
		http.Error(w, "Failed to load skills", http.StatusInternalServerError)
		return
	}

	resp := ReadResponse{Resolve: resolve}

	for _, identifier := range req.Identifiers {
		matches := resolveIdentifier(identifier, resolve, indexed)
		switch len(matches) {
		case 0:
			resp.Missing = append(resp.Missing, ReadIssue{
				Identifier: identifier,
				Reason:     "not_found",
			})
		case 1:
			readSkill, err := h.buildReadResponse(matches[0])
			if err != nil {
				http.Error(w, "Failed to load skill content", http.StatusInternalServerError)
				return
			}
			resp.Skills = append(resp.Skills, readSkill)
		default:
			resp.Ambiguous = append(resp.Ambiguous, ReadAmbiguous{
				Identifier: identifier,
				Candidates: buildCandidates(matches),
			})
		}
	}

	if !allowMissing && (len(resp.Missing) > 0 || len(resp.Ambiguous) > 0) {
		status := http.StatusNotFound
		if len(resp.Ambiguous) > 0 {
			status = http.StatusConflict
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
			if strings.HasPrefix(filename, prefix) {
				filename = strings.TrimPrefix(filename, prefix)
			}
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
	if strings.HasPrefix(s, "skills/") {
		s = strings.TrimPrefix(s, "skills/")
	}
	return strings.TrimPrefix(s, "/")
}

func stripMDExt(path string) string {
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		return path[:len(path)-3]
	}
	return path
}

func (h *Handlers) buildReadResponse(skill indexedSkill) (Response, error) {
	resp := h.toResponse(skill.meta)
	resp.Folder = skill.folder
	resp.File = skill.filename

	content, err := h.store.GetContent(skill.folder, skill.filename)
	if err != nil {
		return Response{}, err
	}
	resp.Content = content

	return resp, nil
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
