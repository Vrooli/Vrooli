package actions

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"prompt-manager/store"
)

// SemanticActionSearcher finds existing actions semantically similar to a
// free-text query. It is a seam: the production implementation adapts the
// aisearch service (semantic vectors + text fallback), while tests inject a
// fake so create previews are deterministic without Qdrant/Ollama.
type SemanticActionSearcher interface {
	SearchSimilarActions(ctx context.Context, query string, limit int) ([]SemanticActionHit, error)
}

// SemanticActionHit is one semantic-similarity result.
type SemanticActionHit struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

// DraftActionInput is the request payload for a create preview. Either Argv
// (the command to infer a contract from) or Contract (a fully authored
// action.json, e.g. from --file) is provided.
type DraftActionInput struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	ID          string          `json:"id,omitempty"`
	Pack        string          `json:"pack,omitempty"`
	Argv        []string        `json:"argv,omitempty"`
	Inputs      []InputOverride `json:"inputs,omitempty"`
	Contract    *store.Action   `json:"contract,omitempty"`
}

// InputOverride refines an inferred input (type, requiredness, description).
type InputOverride struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Required    *bool  `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

// InferenceNote records one field the preview inferred and how, so the author
// can refine it before applying.
type InferenceNote struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// SimilarMatch is an existing action surfaced as a possible near-duplicate.
type SimilarMatch struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"` // "same-command" | "semantic"
}

// ActionPreview is the read-only result of PreviewCreate. It writes nothing.
type ActionPreview struct {
	Rendered   *store.Action      `json:"rendered"`
	Validation ValidationResponse `json:"validation"`
	Inferred   []InferenceNote    `json:"inferred"`
	Warnings   []string           `json:"warnings"`
	Similar    []SimilarMatch     `json:"similar"`
}

var wholePlaceholderRe = regexp.MustCompile(`^\{\{([a-z][a-zA-Z0-9]*)\}\}$`)

// PreviewCreate renders the contract that `action create --apply` would write,
// validates it, infers owner/inputs/permissions from the command, and surfaces
// similar existing actions — all without persisting anything.
func (s *Service) PreviewCreate(ctx context.Context, draft DraftActionInput) (*ActionPreview, error) {
	var (
		action *store.Action
		notes  []InferenceNote
	)
	if draft.Contract != nil {
		// --file path: the contract is already authored; do not re-infer.
		contract := *draft.Contract
		action = &contract
		if action.Kind == "" {
			action.Kind = store.KindAction
		}
		if action.SchemaVersion == 0 {
			action.SchemaVersion = store.CurrentSchemaVersion
		}
		if action.Status == "" {
			action.Status = store.StatusActive
		}
	} else {
		action, notes = s.InferActionFromCommand(ctx, draft.Argv, draft.Name, draft.Description, draft.ID)
	}

	for _, override := range draft.Inputs {
		applyInputOverride(action, override, &notes)
	}

	validation := s.Validate(ctx, action)
	validation.Action = action
	similar := s.FindSimilarActions(ctx, action)
	warnings := buildPreviewWarnings(action, validation, similar)

	return &ActionPreview{
		Rendered:   action,
		Validation: validation,
		Inferred:   notes,
		Warnings:   warnings,
		Similar:    similar,
	}, nil
}

// InferActionFromCommand builds a best-effort action contract from a command
// argv, returning inference notes for every field it filled in. Inference is
// advisory: callers preview the result and refine via flags before applying.
func (s *Service) InferActionFromCommand(ctx context.Context, argv []string, name, description, id string) (*store.Action, []InferenceNote) {
	notes := []InferenceNote{}
	action := &store.Action{
		BaseEntity:  store.BaseEntity{Kind: store.KindAction, SchemaVersion: store.CurrentSchemaVersion},
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Status:      store.StatusActive,
		Command:     store.ActionCommand{Argv: append([]string{}, argv...)},
		Inputs:      map[string]store.ActionInput{},
		Validation:  &store.ActionValidation{Mode: "contract"},
		Timestamps:  store.NewTimestamps(),
	}

	// Owner + permissions inference from argv[0] via the controlled-command resolver.
	var resolution CommandResolution
	if s.resolver != nil && len(argv) > 0 {
		if resolved, err := s.resolver.ResolveCommand(ctx, argv); err == nil {
			resolution = resolved
		}
	}
	switch resolution.Certainty {
	case CertaintyCommand, CertaintyOperation:
		action.Owner = store.ActionOwner{Type: resolution.Owner.Type, ID: resolution.Owner.ID}
		action.Permissions = permissionsFromResolution(resolution)
		notes = append(notes, InferenceNote{Field: "owner", Message: fmt.Sprintf("inferred %s:%s from the command", resolution.Owner.Type, resolution.Owner.ID)})
		if len(resolution.Permissions) > 0 {
			notes = append(notes, InferenceNote{Field: "permissions", Message: "inferred from the command's governance: " + strings.Join(resolution.Permissions, ", ")})
		}
	case CertaintyOwnerOnly:
		action.Owner = store.ActionOwner{Type: resolution.Owner.Type, ID: resolution.Owner.ID}
		notes = append(notes, InferenceNote{Field: "owner", Message: fmt.Sprintf("inferred %s:%s, but its cli/manifest.json does not declare this command yet — the action will run as 'unvalidated' with no declared permissions; edit it once the owning scenario adopts a manifest", resolution.Owner.Type, resolution.Owner.ID)})
	default:
		notes = append(notes, InferenceNote{Field: "owner", Message: "could not resolve a Vrooli-controlled owner; the command must be owned by vrooli, prompt-manager, a scenario, or a resource"})
	}

	// Inputs from {{placeholder}} tokens.
	for _, token := range argv {
		match := wholePlaceholderRe.FindStringSubmatch(token)
		if len(match) != 2 {
			continue
		}
		placeholder := match[1]
		if _, exists := action.Inputs[placeholder]; exists {
			continue
		}
		action.Inputs[placeholder] = store.ActionInput{Type: "string", Required: true}
		notes = append(notes, InferenceNote{Field: "inputs." + placeholder, Message: fmt.Sprintf("inferred as required string from placeholder; refine with --input %s:<type>[:optional]", placeholder)})
	}
	if len(action.Inputs) == 0 {
		action.Inputs = nil
	}

	// ID derivation.
	action.ID = strings.TrimSpace(id)
	if action.ID == "" {
		if derived := deriveActionID(resolution, argv); derived != "" {
			action.ID = derived
			notes = append(notes, InferenceNote{Field: "id", Message: "derived as " + derived + "; override with --id"})
		}
	}

	return action, notes
}

// FindSimilarActions surfaces existing actions that look like near-duplicates of
// the candidate: structural matches (same executable + first subcommand) and,
// when the semantic seam is wired, semantic matches by name/description.
func (s *Service) FindSimilarActions(ctx context.Context, candidate *store.Action) []SimilarMatch {
	matches := map[string]SimilarMatch{}

	// Structural: same executable + first subcommand token.
	if s.store != nil {
		if existing, err := s.store.List(ctx); err == nil {
			candExec, candSub := commandSignature(candidate.Command.Argv)
			for _, action := range existing {
				if action.ID == candidate.ID {
					continue
				}
				exec, sub := commandSignature(action.Command.Argv)
				if exec == "" || exec != candExec || sub != candSub {
					continue
				}
				matches[action.ID] = SimilarMatch{ID: action.ID, Name: action.Name, Score: 1.0, Reason: "same-command"}
			}
		}
	}

	// Semantic: name + description similarity via the injected searcher.
	if s.semanticSearcher != nil {
		query := strings.TrimSpace(candidate.Name + " " + candidate.Description)
		if query != "" {
			if hits, err := s.semanticSearcher.SearchSimilarActions(ctx, query, 5); err == nil {
				for _, hit := range hits {
					if hit.ID == "" || hit.ID == candidate.ID {
						continue
					}
					if existing, ok := matches[hit.ID]; ok {
						if hit.Score > existing.Score {
							existing.Score = hit.Score
							matches[hit.ID] = existing
						}
						continue
					}
					matches[hit.ID] = SimilarMatch{ID: hit.ID, Name: hit.Name, Score: hit.Score, Reason: "semantic"}
				}
			}
		}
	}

	out := make([]SimilarMatch, 0, len(matches))
	for _, match := range matches {
		out = append(out, match)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func applyInputOverride(action *store.Action, override InputOverride, notes *[]InferenceNote) {
	name := strings.TrimSpace(override.Name)
	if name == "" {
		return
	}
	if action.Inputs == nil {
		action.Inputs = map[string]store.ActionInput{}
	}
	input, existed := action.Inputs[name]
	if override.Type != "" {
		input.Type = override.Type
	} else if input.Type == "" {
		input.Type = "string"
	}
	if override.Required != nil {
		input.Required = *override.Required
	} else if !existed {
		input.Required = true
	}
	if override.Description != "" {
		input.Description = override.Description
	}
	action.Inputs[name] = input
	if !existed {
		*notes = append(*notes, InferenceNote{Field: "inputs." + name, Message: "declared via --input but not referenced by any {{placeholder}} in the command; add the placeholder or this input will be unused"})
	}
}

func buildPreviewWarnings(action *store.Action, validation ValidationResponse, similar []SimilarMatch) []string {
	warnings := []string{}
	if strings.TrimSpace(action.Name) == "" {
		warnings = append(warnings, "name is empty — set --name before applying")
	}
	if !validation.Valid {
		warnings = append(warnings, "contract is invalid — fix the failed checks below before --apply")
	}
	if validation.Unvalidated {
		warnings = append(warnings, "owner has not declared cli/manifest.json governance — the action will run as 'unvalidated'")
	}
	if len(similar) > 0 {
		ids := make([]string, 0, len(similar))
		for _, match := range similar {
			ids = append(ids, match.ID)
		}
		warnings = append(warnings, "similar action(s) already exist: "+strings.Join(ids, ", ")+" — consider `prompt-manager action update <id>` instead of creating a near-duplicate")
	}
	return warnings
}

// commandSignature returns the executable and the first non-flag, non-placeholder
// subcommand token, e.g. ("browser-automation-studio", "capture").
func commandSignature(argv []string) (string, string) {
	if len(argv) == 0 {
		return "", ""
	}
	exec := argv[0]
	for _, token := range argv[1:] {
		if strings.HasPrefix(token, "-") || wholePlaceholderRe.MatchString(token) {
			continue
		}
		return exec, token
	}
	return exec, ""
}

func deriveActionID(resolution CommandResolution, argv []string) string {
	base := resolution.Owner.ID
	if base == "" && len(argv) > 0 {
		base = argv[0]
	}
	_, verb := commandSignature(argv)
	id := base
	if verb != "" {
		id = base + "." + verb
	}
	return sanitizeActionID(id)
}

// sanitizeActionID coerces a derived id toward the action ID grammar
// (^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$): lowercase, underscores/spaces → '-',
// invalid characters dropped, leading separators trimmed.
func sanitizeActionID(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-':
			b.WriteRune(r)
		case r == '_' || r == ' ':
			b.WriteRune('-')
		}
	}
	cleaned := strings.Trim(b.String(), ".-")
	if cleaned == "" || !store.IsValidActionID(cleaned) {
		return ""
	}
	return cleaned
}

func permissionsFromResolution(resolution CommandResolution) store.ActionPermissions {
	permissions := store.ActionPermissions{}
	for _, permission := range resolution.Permissions {
		switch permission {
		case "filesystem:read":
			permissions.FilesystemRead = true
		case "filesystem:write":
			permissions.FilesystemWrite = true
		case "network:localhost":
			permissions.LocalhostNetwork = true
		case "network:external":
			permissions.ExternalNetwork = true
		case "api:read":
			permissions.APIRead = true
		case "api:write":
			permissions.APIWrite = true
		case "process:start":
			permissions.ProcessStart = true
		case "process:stop":
			permissions.ProcessStop = true
		case "host:configure":
			permissions.HostConfigure = true
		case "secret:read":
			permissions.SecretRead = true
		case "secret:write":
			permissions.SecretWrite = true
		}
	}
	if resolution.Effect == EffectDestructive {
		permissions.Destructive = true
	}
	return permissions
}
