package findings_test

import (
	"context"
	"errors"
	"testing"

	"web-search/internal/findings"

	"github.com/stretchr/testify/require"
)

// TestResolveDisputeKeepRoundTrips asserts that resolving a disputed finding
// with "keep" returns it to ACTIVE, clears the dispute note, and writes a
// "resolve" audit row after the "flag" one.
func TestResolveDisputeKeepRoundTrips(t *testing.T) {
	repo, d := newRepo(t)
	ctx := context.Background()
	svc := findings.NewService(repo)

	f, err := svc.Add(ctx, findings.NewFinding{Claim: "the sky is blue", Confidence: 0.9})
	require.NoError(t, err)

	flagged, err := svc.Flag(ctx, f.ID, "a source says it is teal")
	require.NoError(t, err)
	require.Equal(t, findings.StatusDisputed, flagged.Status)
	require.Equal(t, "a source says it is teal", flagged.DisputeNote)

	resolved, err := svc.ResolveDispute(ctx, f.ID, findings.ResolutionKeep, "", "checked: the original holds")
	require.NoError(t, err)
	require.Equal(t, findings.StatusActive, resolved.Status)
	require.Empty(t, resolved.DisputeNote)

	require.Equal(t, []string{findings.MutationCreate, findings.MutationFlag, findings.MutationResolve}, auditRows(t, d, f.ID))
}

// TestResolveDisputeSupersedeRetires asserts the "supersede" resolution retires
// the disputed finding in favor of a replacement (reusing Supersede).
func TestResolveDisputeSupersedeRetires(t *testing.T) {
	repo, d := newRepo(t)
	ctx := context.Background()
	svc := findings.NewService(repo)

	old, err := svc.Add(ctx, findings.NewFinding{Claim: "old claim", Confidence: 0.5})
	require.NoError(t, err)
	replacement, err := svc.Add(ctx, findings.NewFinding{Claim: "new claim", Confidence: 0.9})
	require.NoError(t, err)

	_, err = svc.Flag(ctx, old.ID, "contested")
	require.NoError(t, err)

	resolved, err := svc.ResolveDispute(ctx, old.ID, findings.ResolutionSupersede, replacement.ID, "newer evidence")
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, resolved.Status)
	require.Equal(t, replacement.ID, resolved.SupersededBy)

	require.Equal(t, []string{findings.MutationCreate, findings.MutationFlag, findings.MutationSupersede}, auditRows(t, d, old.ID))
}

// TestResolveDisputeRejectsBadInput covers the guardrails: a non-disputed
// finding cannot be "kept", supersede requires a replacement, and an unknown
// resolution is rejected.
func TestResolveDisputeRejectsBadInput(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	svc := findings.NewService(repo)

	f, err := svc.Add(ctx, findings.NewFinding{Claim: "active claim", Confidence: 0.9})
	require.NoError(t, err)

	// keep on a non-disputed finding fails.
	_, err = svc.ResolveDispute(ctx, f.ID, findings.ResolutionKeep, "", "")
	require.Error(t, err)
	var invalid findings.ErrInvalidFinding
	require.True(t, errors.As(err, &invalid))

	// supersede without a replacement fails.
	_, err = svc.Flag(ctx, f.ID, "contested")
	require.NoError(t, err)
	_, err = svc.ResolveDispute(ctx, f.ID, findings.ResolutionSupersede, "", "no replacement")
	require.Error(t, err)
	require.True(t, errors.As(err, &invalid))

	// unknown resolution fails.
	_, err = svc.ResolveDispute(ctx, f.ID, "merge", "", "")
	require.Error(t, err)
	require.True(t, errors.As(err, &invalid))
}
