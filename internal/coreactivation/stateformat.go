package coreactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/stateformat"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const stateFormatLimit = 1 << 20

// ProbeStateFormat reads an image's versioned compatibility manifest through
// its read-only `version --state-format` entrypoint. Unlike the per-owner
// identity manifest, this manifest states which formats an image writes and
// which it can read, so compatibility is a directional question rather than an
// equality test.
func ProbeStateFormat(ctx context.Context, image string) (stateformat.Manifest, error) {
	var manifest stateformat.Manifest
	if err := ValidateRootExecutableRole(image); err != nil {
		return manifest, err
	}
	data, err := probeNativeMetadata(ctx, image, []string{"version", "--state-format"}, stateFormatLimit)
	if err != nil {
		return manifest, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("invalid state format response")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return manifest, fmt.Errorf("state format response has trailing data")
	}
	if err := manifest.Validate(); err != nil {
		return manifest, err
	}
	return manifest, nil
}

// probeStateFormat is the admission seam over ProbeStateFormat so the decision
// rules can be tested without building two real images.
var probeStateFormat = ProbeStateFormat

// QualifyRollbackImages decides whether the serving image can be kept as the
// rollback target for the candidate.
//
// Build and runtime-home layout metadata are still required of both images,
// because they govern more than persistent state. Persistent-state
// compatibility itself is decided by the versioned manifests when both images
// report one: the question is directional, and the older identity digest must
// not be able to override the answer. Images that predate the manifest fall
// back to the identity rule, which is the single transition that migration
// requires.
func QualifyRollbackImages(ctx context.Context, prior, candidate *cliv1.CliVersion, priorImage, candidateImage string) error {
	if err := qualifyRollbackMetadata(prior, candidate); err != nil {
		return err
	}
	priorFormat, priorErr := probeStateFormat(ctx, priorImage)
	candidateFormat, candidateErr := probeStateFormat(ctx, candidateImage)
	if priorErr != nil || candidateErr != nil {
		return qualifyRollbackIdentity(prior, candidate)
	}
	assessment := stateformat.Check(priorFormat, candidateFormat)
	if assessment.AllowsAutomaticRollback() {
		return nil
	}
	if assessment.Verdict == stateformat.Unknown {
		return fmt.Errorf("%w: %w: the manifests could not be compared: %s", ErrConflict, ErrStateContractChanged, strings.Join(formatChanges(assessment), "; "))
	}
	return fmt.Errorf("%w: %w: %s", ErrConflict, ErrStateContractChanged, strings.Join(formatChanges(assessment), "; "))
}

func formatChanges(assessment stateformat.Assessment) []string {
	if len(assessment.Changes) == 0 {
		return []string{"no compatible store set"}
	}
	changes := make([]string, 0, len(assessment.Changes))
	for _, change := range assessment.Changes {
		if change.Store == "" {
			changes = append(changes, change.Kind+": "+change.Detail)
			continue
		}
		changes = append(changes, change.Store+" ("+change.Kind+"): "+change.Detail)
	}
	return changes
}

// ExplainStateFormatChange names the stores behind a state-format refusal. It
// returns nil when either image reports no versioned manifest, leaving the
// per-owner explanation to speak for the identity fallback.
func ExplainStateFormatChange(ctx context.Context, servingImage, candidateImage string) []string {
	prior, err := ProbeStateFormat(ctx, servingImage)
	if err != nil {
		return nil
	}
	candidate, err := ProbeStateFormat(ctx, candidateImage)
	if err != nil {
		return nil
	}
	assessment := stateformat.Check(prior, candidate)
	if assessment.AllowsAutomaticRollback() {
		return nil
	}
	return formatChanges(assessment)
}
