package coreactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/statecontract"
)

const stateManifestLimit = 1 << 20

// ProbeStateManifest reads an image's per-owner persistent-state contract
// through its read-only `version --state-manifest` entrypoint. The manifest
// must be internally consistent; its caller binds it to the image's reported
// contract digest.
func ProbeStateManifest(ctx context.Context, image string) (statecontract.Manifest, error) {
	var manifest statecontract.Manifest
	if err := ValidateRootExecutableRole(image); err != nil {
		return manifest, err
	}
	data, err := probeNativeMetadata(ctx, image, []string{"version", "--state-manifest"}, stateManifestLimit)
	if err != nil {
		return manifest, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("invalid state manifest response")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return manifest, fmt.Errorf("state manifest response has trailing data")
	}
	if err := statecontract.Verify(manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

// ExplainStateContractChange names what differs between two root images'
// persistent-state contracts, owner by owner. When a manifest is unavailable
// it says why instead, so a refusal is never reduced to a bare digest pair.
func ExplainStateContractChange(ctx context.Context, servingImage, candidateImage string) []string {
	candidate, err := ProbeStateManifest(ctx, candidateImage)
	if err != nil {
		return []string{"the candidate does not report a per-owner state manifest: " + err.Error()}
	}
	serving, err := ProbeStateManifest(ctx, servingImage)
	if err != nil {
		return []string{fmt.Sprintf("the serving image predates per-owner state manifests (contract format v%d), so the two contracts cannot be compared owner by owner; the first candidate built with this format always differs from it", statecontract.ManifestVersion-1)}
	}
	changes := statecontract.Diff(serving, candidate)
	if len(changes) == 0 {
		return []string{"the per-owner manifests are equal; the reported contract digests differ"}
	}
	return changes
}

// ExplainRollbackRefusal appends the owner-level explanation to a
// state-contract refusal, or the image's own reason to an unqualified one.
// Other refusals are returned unchanged.
func ExplainRollbackRefusal(ctx context.Context, err error, servingImage, candidateImage string) error {
	if errors.Is(err, ErrUnqualifiedStateContract) {
		image := candidateImage
		if strings.Contains(err.Error(), "the serving image") {
			image = servingImage
		}
		if _, probeErr := ProbeStateManifest(ctx, image); probeErr != nil {
			return fmt.Errorf("%w (%s)", err, probeErr.Error())
		}
		return err
	}
	if !errors.Is(err, ErrStateContractChanged) {
		return err
	}
	return fmt.Errorf("%w:\n  - %s", err, strings.Join(ExplainStateContractChange(ctx, servingImage, candidateImage), "\n  - "))
}
