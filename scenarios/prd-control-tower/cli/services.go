package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

type Services struct {
	Health *HealthService
	Drafts *DraftService
	AI     *AIService
	PRD    *PRDService
}

func NewServices(api *cliutil.APIClient) *Services {
	return &Services{
		Health: &HealthService{api: api},
		Drafts: &DraftService{api: api},
		AI:     &AIService{api: api},
		PRD:    &PRDService{api: api},
	}
}

type HealthService struct {
	api *cliutil.APIClient
}

func (s *HealthService) Status() ([]byte, HealthResponse, error) {
	body, err := s.api.Get("/health", nil)
	if err != nil {
		return nil, HealthResponse{}, err
	}
	var parsed HealthResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return body, HealthResponse{}, fmt.Errorf("parse response: %w", err)
	}
	return body, parsed, nil
}

type DraftService struct {
	api *cliutil.APIClient
}

func (s *DraftService) List() ([]byte, DraftListResponse, error) {
	body, err := s.api.Get("/api/v1/drafts", url.Values{})
	if err != nil {
		return nil, DraftListResponse{}, err
	}
	var parsed DraftListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return body, DraftListResponse{}, fmt.Errorf("parse response: %w", err)
	}
	return body, parsed, nil
}

func (s *DraftService) Publish(draftID string, req PublishRequest) ([]byte, PublishResponse, error) {
	path := fmt.Sprintf("/api/v1/drafts/%s/publish", draftID)
	body, err := s.api.Request(http.MethodPost, path, nil, req)
	if err != nil {
		return nil, PublishResponse{}, err
	}
	var parsed PublishResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return body, PublishResponse{}, fmt.Errorf("parse response: %w", err)
	}
	return body, parsed, nil
}

type PRDService struct {
	api *cliutil.APIClient
}

func (s *PRDService) Validate(req ValidateRequest) ([]byte, error) {
	return s.api.Request(http.MethodPost, "/api/v1/drafts/validate", nil, req)
}

// ValidateStandards returns detailed PRD validation with violations in standards format.
func (s *PRDService) ValidateStandards(entityType, entityName string, useCache bool) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/quality/%s/%s/standards", url.PathEscape(entityType), url.PathEscape(entityName))
	params := url.Values{}
	if !useCache {
		params.Set("use_cache", "false")
	}
	return s.api.Get(path, params)
}

type AIService struct {
	api *cliutil.APIClient
}

func (s *AIService) GenerateDraft(req AIGenerateDraftRequest) ([]byte, AIGenerateDraftResponse, error) {
	body, err := s.api.Request(http.MethodPost, "/api/v1/drafts/ai/generate", nil, req)
	if err != nil {
		return nil, AIGenerateDraftResponse{}, err
	}
	var parsed AIGenerateDraftResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return body, AIGenerateDraftResponse{}, fmt.Errorf("parse response: %w", err)
	}
	return body, parsed, nil
}
