package graph

import (
	"context"
	"path/filepath"
	"strings"

	"typescript-code-graph/internal/sidecar"
)

// Service orchestrates the Extract flow: validate input → acquire the
// per-path mutex → check sidecar readiness → call sidecar.Extract →
// normalize → hash → return.
//
// All non-trivial work flows through seams (sidecar.SidecarClient,
// PathMutex) so the Service itself is pure orchestration and
// exhaustively testable with sidecar/mocks.FakeSidecarClient.
//
// Substrate rule: this file does NOT import time, os, net/http, or
// os/exec. Timing measurement lives in the handler; filesystem checks
// live in the Node sidecar (it owns the tsconfig discovery).
type Service struct {
	client sidecar.SidecarClient
	mu     *PathMutex
}

// NewService wires the production Service. Both arguments are required;
// the caller (api/main.go) owns construction so the mutex can be shared
// with the future rewrite domain (OT-P0-007).
func NewService(client sidecar.SidecarClient, mu *PathMutex) *Service {
	return &Service{client: client, mu: mu}
}

// Extract validates the input, locks the absolute project path, asks
// the sidecar for the raw graph, normalizes, hashes, and returns the
// ExtractOutput. Errors are typed ExtractError so handlers can map them
// to Connect codes via ErrorToConnectCode.
//
// SidecarRequestID is populated from the sidecar seam (the supervisor-
// minted IPC request UUID) so callers can correlate a graph with the
// exact sidecar request that produced it. ExtractionMs is NOT populated
// here — timing belongs to the transport layer, and the handler measures
// it with its own clock when projecting onto the proto ExtractResponse.
func (s *Service) Extract(ctx context.Context, in ExtractInput) (ExtractOutput, error) {
	path := strings.TrimSpace(in.ProjectPath)
	if path == "" {
		return ExtractOutput{}, ExtractError{
			Kind:    ExtractErrorInvalidInput,
			Message: "project_path is required",
		}
	}
	if !filepath.IsAbs(path) {
		return ExtractOutput{}, ExtractError{
			Kind:    ExtractErrorInvalidInput,
			Path:    path,
			Message: "project_path must be absolute",
		}
	}
	abs := filepath.Clean(path)

	unlock := s.mu.Lock(abs)
	defer unlock()

	if st := s.client.Status(); st != sidecar.StatusReady {
		return ExtractOutput{}, ExtractError{
			Kind:    ExtractErrorSidecarUnavailable,
			Path:    abs,
			Message: "sidecar status " + string(st),
		}
	}

	res, err := s.client.Extract(ctx, abs)
	if err != nil {
		return ExtractOutput{}, fromSidecarError(abs, err)
	}

	g := Normalize(res.Graph)
	warnings := fromSidecarWarnings(res.Warnings)
	hash := GraphHash(g)

	return ExtractOutput{
		Graph:            g,
		Warnings:         warnings,
		GraphHash:        hash,
		SidecarRequestID: res.RequestID,
	}, nil
}
