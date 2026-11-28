package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// createDeploymentForConnect reuses the REST handler as the single canonical
// validation and persistence path. The request context is preserved so scoped
// identity and request metadata remain available to the handler.
func (s *Server) createDeploymentForConnect(ctx context.Context, req domain.CreateDeploymentRequest) (*domain.Deployment, bool, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, false, apierrors.Internal("Failed to encode deployment request", err)
	}
	httpReq := httptest.NewRequest("POST", "/api/v1/deployments", bytes.NewReader(body)).WithContext(ctx)
	recorder := httptest.NewRecorder()
	s.handleCreateDeployment(recorder, httpReq)
	if recorder.Code < 200 || recorder.Code >= 300 {
		var envelope apierrors.Envelope
		if json.Unmarshal(recorder.Body.Bytes(), &envelope) == nil && envelope.Error.Code != "" {
			return nil, false, apierrors.New(envelope.Error.Code, envelope.Error.Message)
		}
		return nil, false, apierrors.New(apierrors.CodeManifestInvalid, "Deployment manifest was rejected").WithDetail("status", recorder.Code)
	}
	var response struct {
		Deployment *domain.Deployment `json:"deployment"`
		Created    bool               `json:"created"`
		Updated    bool               `json:"updated"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || response.Deployment == nil {
		return nil, false, apierrors.Internal("Failed to decode deployment response", err)
	}
	return response.Deployment, response.Created, nil
}
