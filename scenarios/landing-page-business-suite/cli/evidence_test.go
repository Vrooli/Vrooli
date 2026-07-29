package main

import (
	"os"
	"strconv"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"landing-page-business-suite/cli/domains"
	"landing-page-business-suite/cli/internal/support"
)

// TestPrimitiveEvidenceArtifactCurrent keeps the generated, static primitive
// evidence in lockstep with the manifest-backed command tree. CLI
// Health reads this artifact to verify that declared renderer primitives are
// actually constructed by cli-core rather than asserted only in JSON metadata.
func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups, err := evidenceGroups()
	if err != nil {
		t.Fatalf("assemble manifest-backed commands: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    appName,
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, updatePrimitiveEvidence(t))
}

// updatePrimitiveEvidence accepts Go boolean syntax so an accidental value
// cannot silently alter generated evidence. Empty keeps normal test runs
// read-only; maintainers opt in with UPDATE_CLI_EVIDENCE=true.
func updatePrimitiveEvidence(t *testing.T) bool {
	t.Helper()
	raw, set := os.LookupEnv("UPDATE_CLI_EVIDENCE")
	if !set || raw == "" {
		return false
	}
	update, err := strconv.ParseBool(raw)
	if err != nil {
		t.Fatalf("UPDATE_CLI_EVIDENCE must be a boolean: %v", err)
	}
	return update
}

func TestDeclaredMeasuresPrimitivesHaveObservedEvidence(t *testing.T) {
	groups, err := evidenceGroups()
	if err != nil {
		t.Fatalf("assemble manifest-backed commands: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    appName,
		ManifestRaw: manifestBytes,
		Groups:      groups,
	})
	if err != nil {
		t.Fatalf("build primitive evidence: %v", err)
	}
	for path, observed := range artifact.ObservedPrimitives() {
		if observed == "" {
			t.Errorf("command %q has no observed primitive evidence", path)
		}
	}
}

func evidenceGroups() ([]cliapp.SubcommandGroup, error) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		return nil, err
	}
	var legacyCommands []cliapp.Command
	for _, group := range domains.CommandGroups(support.Dependencies{}) {
		for _, command := range group.Commands {
			switch command.Name {
			case "admin-stripe-settings", "admin-stripe-settings-update", "admin-stripe-secret", "variant-space", "seo", "admin-variant-seo-update", "landing-config",
				"admin-bundles", "admin-bundle-price-update", "admin-coupon-mappings", "admin-coupons-create", "admin-coupons-delete", "admin-coupons-get", "admin-coupons-list", "admin-coupons-update", "admin-coupons-usage", "admin-plan-coupon-remove", "admin-plan-coupon-set", "admin-stripe-coupons-preview":
				legacyCommands = append(legacyCommands, command)
			}
		}
	}
	return append(groups, cliapp.SubcommandGroup{Name: "legacy", Subcommands: legacyCommands}), nil
}
