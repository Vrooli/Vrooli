package main

import (
	"context"
	"testing"
	"time"

	"git-control-tower/internal/policygate"
	"github.com/vrooli/repo-contract-go/repocontracttest"

	"github.com/vrooli/api-core/apihttptest"
	"github.com/vrooli/cli-core/cliutil"
)

// --- Shared Test Utilities ---
// This file contains test helpers shared across all test files.
// Use these helpers to avoid duplication and ensure consistent test setup.

// WriteTestFile creates a file in the given directory with the specified content.
func WriteTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	repocontracttest.WriteFile(t, path, contents)
}

// authorizedHumanContext models the verified principal attached by the
// deployment authentication owner. Direct writer-service tests must carry the
// same authority as production callers; a bare context is intentionally not
// enough to reach a repository writer.
func authorizedHumanContext() context.Context {
	ctx := policygate.WithPrincipal(context.Background(), policygate.Principal{
		Kind:     cliutil.CallerKindHuman,
		Subject:  "test-operator",
		Verified: true,
	})
	consumedAt := time.Now().UTC()
	return policygate.WithIntent(ctx, policygate.HumanIntent{
		ID: "test-consumed-intent", PrincipalID: "test-operator",
		ExpiresAt: consumedAt.Add(time.Minute), ConsumedAt: &consumedAt, Consumed: true,
	})
}

func authorizedHumanContextWithoutIntent() context.Context {
	return policygate.WithPrincipal(context.Background(), policygate.Principal{
		Kind:     cliutil.CallerKindHuman,
		Subject:  "test-operator",
		Verified: true,
	})
}

// AssertContains checks if the expected value exists in the slice.
func AssertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	apihttptest.ContainsString(t, values, expected)
}
