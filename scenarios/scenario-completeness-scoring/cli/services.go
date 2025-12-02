package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
	"scenario-completeness-scoring/cli/models"
)

type Services struct {
	Health  *HealthService
	Scoring *ScoringService
	Config  *ConfigService
}

func NewServices(api *cliutil.APIClient) *Services {
	return &Services{
		Health:  &HealthService{api: api},
		Scoring: &ScoringService{api: api},
		Config:  &ConfigService{api: api},
	}
}

type HealthService struct {
	api *cliutil.APIClient
}

func (s *HealthService) Status() ([]byte, models.HealthResponse, error) {
	body, err := s.api.Get("/health", nil)
	if err != nil {
		return nil, models.HealthResponse{}, err
	}
	var parsed models.HealthResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return body, models.HealthResponse{}, fmt.Errorf("parse response: %w", err)
	}
	return body, parsed, nil
}

func (s *HealthService) Collectors() ([]byte, error) {
	return s.api.Get("/api/v1/health/collectors", nil)
}

func (s *HealthService) CircuitBreakerStatus() ([]byte, error) {
	return s.api.Get("/api/v1/health/circuit-breaker", nil)
}

func (s *HealthService) ResetCircuitBreaker() ([]byte, error) {
	return s.api.Request(http.MethodPost, "/api/v1/health/circuit-breaker/reset", nil, map[string]interface{}{})
}

type scoresListResponse struct {
	Scenarios []struct {
		Scenario       string  `json:"scenario"`
		Category       string  `json:"category"`
		Score          float64 `json:"score"`
		Classification string  `json:"classification"`
		Partial        bool    `json:"partial"`
	} `json:"scenarios"`
	Total int `json:"total"`
}

type ScoringService struct {
	api *cliutil.APIClient
}

func (s *ScoringService) ScoresList() (scoresListResponse, []byte, error) {
	body, err := s.api.Get("/api/v1/scores", nil)
	if err != nil {
		return scoresListResponse{}, nil, err
	}
	var resp scoresListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return scoresListResponse{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}

func (s *ScoringService) Score(scenarioName string) (models.ScoreResponse, []byte, error) {
	path := fmt.Sprintf("/api/v1/scores/%s", scenarioName)
	body, err := s.api.Get(path, nil)
	if err != nil {
		return models.ScoreResponse{}, nil, err
	}
	var resp models.ScoreResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return models.ScoreResponse{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}

func (s *ScoringService) Calculate(scenarioName, source string, tags []string) ([]byte, error) {
	payload := map[string]interface{}{}
	if source != "" {
		payload["source"] = source
	}
	if len(tags) > 0 {
		payload["tags"] = tags
	}
	path := fmt.Sprintf("/api/v1/scores/%s/calculate", scenarioName)
	return s.api.Request(http.MethodPost, path, nil, payload)
}

func (s *ScoringService) History(scenarioName string, limit int, source string, tags []string) ([]byte, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if source != "" {
		query.Set("source", source)
	}
	for _, tag := range tags {
		query.Add("tag", tag)
	}
	path := fmt.Sprintf("/api/v1/scores/%s/history", scenarioName)
	return s.api.Get(path, query)
}

func (s *ScoringService) Trends(scenarioName string, limit int, source string, tags []string) ([]byte, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if source != "" {
		query.Set("source", source)
	}
	for _, tag := range tags {
		query.Add("tag", tag)
	}
	path := fmt.Sprintf("/api/v1/scores/%s/trends", scenarioName)
	return s.api.Get(path, query)
}

func (s *ScoringService) WhatIf(scenarioName string, changes map[string]interface{}) ([]byte, error) {
	if changes == nil {
		changes = map[string]interface{}{}
	}
	path := fmt.Sprintf("/api/v1/scores/%s/what-if", scenarioName)
	return s.api.Request(http.MethodPost, path, nil, changes)
}

func (s *ScoringService) Recommend(scenarioName string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/recommendations/%s", scenarioName)
	return s.api.Get(path, nil)
}

type ConfigService struct {
	api *cliutil.APIClient
}

func (s *ConfigService) Get() ([]byte, error) {
	return s.api.Get("/api/v1/config", nil)
}

func (s *ConfigService) Set(payload map[string]interface{}) ([]byte, error) {
	return s.api.Request(http.MethodPut, "/api/v1/config", nil, payload)
}

func (s *ConfigService) Presets() ([]byte, error) {
	return s.api.Get("/api/v1/config/presets", nil)
}

func (s *ConfigService) ApplyPreset(name string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/config/presets/%s/apply", name)
	return s.api.Request(http.MethodPost, path, nil, map[string]interface{}{})
}
