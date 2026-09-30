package main

import (
	"fmt"
	"net/http"
	"strings"

	"git-control-tower/internal/advisory"
)

type advisoryDraftRequest struct {
	Kind     string                  `json:"kind"`
	Subject  advisory.ChangeSubject  `json:"subject"`
	Evidence advisory.EvidenceBundle `json:"evidence"`
}

type advisoryDraftResponse struct {
	Status        string            `json:"status"`
	Kind          string            `json:"kind"`
	SubjectDigest string            `json:"subjectDigest"`
	Title         string            `json:"title"`
	Body          string            `json:"body"`
	Unknowns      []string          `json:"unknowns,omitempty"`
	Coverage      advisory.Coverage `json:"coverage"`
	EvidenceRefs  []string          `json:"evidenceRefs"`
}

func (s *Server) handleAdvisoryDraft(w http.ResponseWriter, r *http.Request) {
	var req advisoryDraftRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	response, err := buildAdvisoryDraft(req)
	if err != nil {
		NewResponse(w).Error(http.StatusUnprocessableEntity, err.Error())
		return
	}
	NewResponse(w).OK(response)
}

func buildAdvisoryDraft(req advisoryDraftRequest) (advisoryDraftResponse, error) {
	if err := req.Subject.Validate(); err != nil {
		return advisoryDraftResponse{}, fmt.Errorf("subject: %w", err)
	}
	digest, err := req.Subject.Digest()
	if err != nil {
		return advisoryDraftResponse{}, err
	}
	if err := req.Evidence.Validate(); err != nil {
		return advisoryDraftResponse{}, fmt.Errorf("evidence: %w", err)
	}
	if req.Evidence.SubjectDigest != digest {
		return advisoryDraftResponse{}, fmt.Errorf("evidence subject digest does not match subject")
	}
	switch req.Kind {
	case "summary", "commit-draft", "pr-draft", "release-draft":
	default:
		return advisoryDraftResponse{}, fmt.Errorf("unsupported advisory kind %q", req.Kind)
	}
	claims := make([]string, 0, len(req.Evidence.Claims))
	refs := make([]string, 0)
	seenRefs := map[string]bool{}
	for _, claim := range req.Evidence.Claims {
		claims = append(claims, "- "+claim.Text)
		for _, ref := range claim.EvidenceRefs {
			if !seenRefs[ref] {
				refs = append(refs, ref)
				seenRefs[ref] = true
			}
		}
	}
	title := "Evidence-backed change summary"
	if len(claims) > 0 {
		title = strings.TrimPrefix(strings.TrimSpace(claims[0]), "- ")
	}
	body := strings.Join(claims, "\n")
	if body == "" {
		body = "No grounded claims were supplied."
	}
	if len(req.Evidence.Unknowns) > 0 {
		body += "\n\nUnknowns:\n- " + strings.Join(req.Evidence.Unknowns, "\n- ")
	}
	return advisoryDraftResponse{Status: req.Evidence.Status, Kind: req.Kind, SubjectDigest: digest, Title: title, Body: body, Unknowns: req.Evidence.Unknowns, Coverage: req.Evidence.Coverage, EvidenceRefs: refs}, nil
}
