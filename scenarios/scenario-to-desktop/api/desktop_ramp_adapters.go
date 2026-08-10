package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// desktopRampBuilder adapts the scenario's existing artifact index to the
// spine Builder seam. The build pipeline remains the owner of electron-builder
// invocation; validation receives only an immutable artifact description.
type desktopRampBuilder struct{ finder validationArtifactFinder }

func (b desktopRampBuilder) Build(_ context.Context, request deliveryramp.BuildRequest) (deliveryramp.Artifact, error) {
	if b.finder == nil {
		return deliveryramp.Artifact{}, fmt.Errorf("desktop artifact builder is unavailable")
	}
	scenario := strings.TrimSpace(request.SourceRef)
	if scenario == "" {
		return deliveryramp.Artifact{}, fmt.Errorf("desktop build source reference is required")
	}
	path, err := b.finder.FindArtifact(scenario)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("resolve built desktop artifact: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("stat built desktop artifact: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("hash built desktop artifact: %w", err)
	}
	digest := sha256.Sum256(data)
	return deliveryramp.Artifact{ImmutableRef: "artifact:" + hex.EncodeToString(digest[:]), LocalPath: path, Kind: "desktop-installer", Checksum: "sha256:" + hex.EncodeToString(digest[:]), SizeBytes: info.Size(), CreatedAt: info.ModTime().UTC()}, nil
}

// desktopRampDistributor only states legitimate destinations. It does not
// upload or publish; those side effects remain owned by the deployment
// pipeline and its explicit distribution handlers.
type desktopRampDistributor struct{}

func (desktopRampDistributor) Distribute(_ context.Context, request deliveryramp.DistributionRequest) (deliveryramp.DistributionResult, error) {
	if strings.TrimSpace(request.Artifact.ImmutableRef) == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "desktop artifact has no immutable identity"}, nil
	}
	return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionPass, Targets: []deliveryramp.DistributionTarget{
		{ID: "local-installer", Kind: "local", Available: true},
		{ID: "verified-sideload", Kind: "verified-sideload", Available: true},
	}}, nil
}

var (
	_ deliveryramp.Builder     = desktopRampBuilder{}
	_ deliveryramp.Distributor = desktopRampDistributor{}
)
