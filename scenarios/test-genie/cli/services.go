package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

type Services struct {
	Health   *HealthService
	Suite    *SuiteService
	RunTests *RunTestsService
}

func NewServices(api *cliutil.APIClient) *Services {
	return &Services{
		Health:   &HealthService{api: api},
		Suite:    &SuiteService{api: api},
		RunTests: &RunTestsService{api: api},
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
	var resp HealthResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, HealthResponse{}, fmt.Errorf("parse health response: %w", err)
	}
	return body, resp, nil
}

type SuiteService struct {
	api *cliutil.APIClient
}

func (s *SuiteService) Generate(req GenerateRequest) (GenerateResponse, []byte, error) {
	body, err := s.api.Request(http.MethodPost, "/api/v1/suite-requests", nil, req)
	if err != nil {
		return GenerateResponse{}, nil, err
	}
	var resp GenerateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return GenerateResponse{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}

func (s *SuiteService) Execute(req ExecuteRequest) (ExecuteResponse, []byte, error) {
	body, err := s.api.Request(http.MethodPost, "/api/v1/executions", nil, req)
	if err != nil {
		return ExecuteResponse{}, nil, err
	}
	var resp ExecuteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ExecuteResponse{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}

type RunTestsService struct {
	api *cliutil.APIClient
}

func (s *RunTestsService) Run(scenario string, req RunTestsRequest) (RunTestsResponse, []byte, error) {
	path := fmt.Sprintf("/api/v1/scenarios/%s/run-tests", url.PathEscape(scenario))
	body, err := s.api.Request(http.MethodPost, path, nil, req)
	if err != nil {
		return RunTestsResponse{}, nil, err
	}
	var resp RunTestsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return RunTestsResponse{}, body, fmt.Errorf("parse response: %w", err)
	}
	return resp, body, nil
}
