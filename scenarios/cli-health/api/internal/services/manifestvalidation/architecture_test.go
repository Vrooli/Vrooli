package manifestvalidation

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func cmd(name, primitive string) cliapp.ManifestCommand {
	c := cliapp.ManifestCommand{
		Name:       name,
		Binding:    cliapp.ManifestBinding{Kind: "connect-rpc", Service: "Svc", Method: name},
		Governance: cliapp.ManifestGovernance{Effect: "read", RunEligible: true},
	}
	if primitive != "" {
		c.Architecture = &cliapp.ManifestArchitecture{Primitive: primitive}
	}
	return c
}

func manifestWith(commands []cliapp.ManifestCommand, exceptions []cliapp.ManifestException) *cliapp.Manifest {
	return &cliapp.Manifest{
		Name:       "fixture",
		Groups:     []cliapp.ManifestGroup{{Name: "g1", Commands: commands}},
		Exceptions: exceptions,
	}
}

// evidence builds an ArchitectureEvidence from "g1 name" -> primitive pairs.
func evidence(pairs map[string]cliapp.PrimitiveClass) ArchitectureEvidence {
	return ArchitectureEvidence{Primitives: pairs}
}

func TestArchitectureStatic_VerifiedWhenEvidenceMatches(t *testing.T) {
	// Declared primitives that match the observed cli-core primitive evidence
	// reach verified maturity (L4) — no finding.
	m := manifestWith([]cliapp.ManifestCommand{cmd("list", "proto_list"), cmd("create", "proto_mutation")}, nil)
	ev := evidence(map[string]cliapp.PrimitiveClass{
		"g1 list":   cliapp.PrimitiveProtoList,
		"g1 create": cliapp.PrimitiveProtoMutation,
	})
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	if len(got) != 0 {
		t.Fatalf("verified-by-evidence manifest should be clean, got %+v", got)
	}
}

func TestArchitectureStatic_DeclaredOnlyIsUnverifiedDebt(t *testing.T) {
	// A declared primitive with NO observed evidence cannot be verified: it is
	// advisory not-yet-verified debt, never falsely clean. This is the core teeth
	// of the follow-up — declaration alone no longer reaches top maturity.
	m := manifestWith([]cliapp.ManifestCommand{cmd("list", "proto_list")}, nil)
	got := architectureStaticFindings(m, ArchitectureEvidence{}, "cli/manifest.json")
	f := findingByCode(got, CodeArchPrimitiveUnverif)
	if f == nil {
		t.Fatalf("expected primitive_unverified for declared-only command, got %+v", got)
	}
	if f.Severity != SeverityWarning {
		t.Fatalf("primitive_unverified must be a warning (advisory rollout debt), got %s", f.Severity)
	}
}

func TestArchitectureStatic_MismatchIsGatingError(t *testing.T) {
	// Declared primitive contradicts the observed cli-core primitive: a gating
	// error, not advisory debt.
	m := manifestWith([]cliapp.ManifestCommand{cmd("list", "proto_list")}, nil)
	ev := evidence(map[string]cliapp.PrimitiveClass{"g1 list": cliapp.PrimitiveProtoMutation})
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	f := findingByCode(got, CodeArchPrimitiveMismatch)
	if f == nil {
		t.Fatalf("expected primitive_mismatch for contradictory evidence, got %+v", got)
	}
	if f.Severity != SeverityError {
		t.Fatalf("primitive_mismatch must be an error (gating), got %s", f.Severity)
	}
}

func TestArchitectureStatic_PrimitiveUndeclared(t *testing.T) {
	// list is verified by evidence; legacy declares nothing → exactly one
	// undeclared finding.
	m := manifestWith([]cliapp.ManifestCommand{cmd("list", "proto_list"), cmd("legacy", "")}, nil)
	ev := evidence(map[string]cliapp.PrimitiveClass{"g1 list": cliapp.PrimitiveProtoList})
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	if len(got) != 1 || got[0].Code != CodeArchPrimitiveUndecl {
		t.Fatalf("expected one primitive_undeclared, got %+v", got)
	}
	if got[0].Severity != SeverityWarning {
		t.Fatalf("primitive_undeclared must be a warning (honest debt, non-failing), got %s", got[0].Severity)
	}
}

func TestArchitectureStatic_LocalCommandDoesNotRequireProtoPrimitive(t *testing.T) {
	m := manifestWith([]cliapp.ManifestCommand{{
		Name:       "status",
		Binding:    cliapp.ManifestBinding{Kind: "local"},
		Governance: cliapp.ManifestGovernance{Effect: "read", RunEligible: true},
	}}, nil)
	if got := architectureStaticFindings(m, ArchitectureEvidence{}, "cli/manifest.json"); len(got) != 0 {
		t.Fatalf("local command must not create proto architecture debt: %+v", got)
	}
}

func TestArchitectureStatic_DeclaredExceptionVerifiedByEvidence(t *testing.T) {
	// A custom command declared in exceptions[] that is NOT a manifest-bound
	// command is a legitimate special case only when static cli-core evidence
	// proves the matching primitive.
	m := manifestWith(
		[]cliapp.ManifestCommand{cmd("list", "proto_list")},
		[]cliapp.ManifestException{{Command: "execute", Class: "durable_run", Reason: "server-owned run"}},
	)
	ev := evidence(map[string]cliapp.PrimitiveClass{
		"g1 list": cliapp.PrimitiveProtoList,
		"execute": cliapp.PrimitiveDurableRun,
	})
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	if len(got) != 0 {
		t.Fatalf("declared exception with matching evidence should be clean, got %+v", got)
	}
}

func TestArchitectureStatic_DeclaredExceptionWithoutEvidenceIsUnverified(t *testing.T) {
	m := manifestWith(
		[]cliapp.ManifestCommand{cmd("list", "proto_list")},
		[]cliapp.ManifestException{{Command: "execute", Class: "durable_run", Reason: "server-owned run"}},
	)
	ev := evidence(map[string]cliapp.PrimitiveClass{"g1 list": cliapp.PrimitiveProtoList})
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	if findingByCode(got, CodeArchPrimitiveUnverif) == nil {
		t.Fatalf("declared exception without matching evidence should be unverified, got %+v", got)
	}
}

func TestArchitectureStatic_DeclaredExceptionMismatchedEvidenceIsGating(t *testing.T) {
	m := manifestWith(
		[]cliapp.ManifestCommand{cmd("list", "proto_list")},
		[]cliapp.ManifestException{{Command: "execute", Class: "durable_run", Reason: "server-owned run"}},
	)
	ev := evidence(map[string]cliapp.PrimitiveClass{
		"g1 list": cliapp.PrimitiveProtoList,
		"execute": cliapp.PrimitiveStreaming,
	})
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	if findingByCode(got, CodeArchPrimitiveMismatch) == nil {
		t.Fatalf("declared exception with wrong evidence should be a mismatch, got %+v", got)
	}
}

func TestArchitectureStatic_ClaimedViolationWhenExceptionNamesBoundCommand(t *testing.T) {
	// exceptions[] claims "g1 list" is a durable_run special case, but it is a
	// normal manifest-bound proto command — a false maturity claim (gating).
	m := manifestWith(
		[]cliapp.ManifestCommand{cmd("list", "proto_list")},
		[]cliapp.ManifestException{{Command: "g1 list", Class: "durable_run", Reason: "bogus"}},
	)
	ev := evidence(map[string]cliapp.PrimitiveClass{"g1 list": cliapp.PrimitiveProtoList})
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	if findingByCode(got, CodeArchClaimedViolation) == nil {
		t.Fatalf("expected claimed_maturity_violation, got %+v", got)
	}
	if findingByCode(got, CodeArchClaimedViolation).Severity != SeverityError {
		t.Fatalf("claimed_maturity_violation must be an error (gating)")
	}
}

func TestArchitectureRuntime_StaleExceptionIsInvalid(t *testing.T) {
	// exceptions[] declares "execute" but the runtime surface does not expose it.
	m := manifestWith(
		[]cliapp.ManifestCommand{cmd("list", "proto_list")},
		[]cliapp.ManifestException{{Command: "execute", Class: "durable_run", Reason: "server-owned run"}},
	)
	obs := RuntimeObservation{Resolved: true, Commands: []RuntimeCommand{{Group: "g1", Name: "list"}}}
	got := architectureRuntimeFindings(m, obs, "cli/manifest.json")
	if findingByCode(got, CodeArchMetadataInvalid) == nil {
		t.Fatalf("expected metadata_invalid for stale exception, got %+v", got)
	}
}

func TestArchitectureRuntime_LiveExceptionIsClean(t *testing.T) {
	m := manifestWith(
		[]cliapp.ManifestCommand{cmd("list", "proto_list")},
		[]cliapp.ManifestException{{Command: "execute", Class: "durable_run", Reason: "server-owned run"}},
	)
	obs := RuntimeObservation{Resolved: true, Commands: []RuntimeCommand{{Group: "", Name: "execute"}, {Group: "g1", Name: "list"}}}
	got := architectureRuntimeFindings(m, obs, "cli/manifest.json")
	if len(got) != 0 {
		t.Fatalf("live declared exception should be clean, got %+v", got)
	}
}

func TestArchitectureRuntime_TopLevelExceptionMayAppearUnderHelpSection(t *testing.T) {
	m := manifestWith(
		[]cliapp.ManifestCommand{cmd("list", "proto_list")},
		[]cliapp.ManifestException{{Command: "execute", Class: "durable_run", Reason: "server-owned run"}},
	)
	obs := RuntimeObservation{Resolved: true, Commands: []RuntimeCommand{{Group: "Suites", Name: "execute"}, {Group: "g1", Name: "list"}}}
	got := architectureRuntimeFindings(m, obs, "cli/manifest.json")
	if len(got) != 0 {
		t.Fatalf("top-level exception exposed under a help display section should be clean, got %+v", got)
	}
}

func TestArchitectureRuntime_GroupedExceptionRequiresFullPath(t *testing.T) {
	m := manifestWith(
		[]cliapp.ManifestCommand{cmd("list", "proto_list")},
		[]cliapp.ManifestException{{Command: "runs wait", Class: "streaming", Reason: "server stream"}},
	)
	obs := RuntimeObservation{Resolved: true, Commands: []RuntimeCommand{{Group: "other", Name: "wait"}, {Group: "g1", Name: "list"}}}
	got := architectureRuntimeFindings(m, obs, "cli/manifest.json")
	if findingByCode(got, CodeArchMetadataInvalid) == nil {
		t.Fatalf("grouped exception must require exact group path, got %+v", got)
	}
}

func TestArchitectureRuntime_SkippedWhenProbeUnresolved(t *testing.T) {
	m := manifestWith([]cliapp.ManifestCommand{cmd("list", "proto_list")}, []cliapp.ManifestException{{Command: "gone", Class: "streaming", Reason: "x"}})
	if got := architectureRuntimeFindings(m, RuntimeObservation{Resolved: false}, "cli/manifest.json"); got != nil {
		t.Fatalf("runtime findings must be skipped when the probe did not resolve, got %+v", got)
	}
}

func TestArchitectureStatic_PerCommandExceptionVerifiedByEvidence(t *testing.T) {
	// A command declaring a per-command exception (special-case shape) is verified
	// when the observed cli-core primitive satisfies that exception class.
	c := cliapp.ManifestCommand{
		Name:         "stream",
		Binding:      cliapp.ManifestBinding{Kind: "connect-rpc", Service: "Svc", Method: "Stream"},
		Governance:   cliapp.ManifestGovernance{Effect: "read", RunEligible: true},
		Architecture: &cliapp.ManifestArchitecture{Exception: &cliapp.ManifestArchitectureExcept{Class: "streaming", Reason: "holds a server stream"}},
	}
	m := manifestWith([]cliapp.ManifestCommand{c}, nil)

	// Matching evidence -> verified, no finding.
	ev := evidence(map[string]cliapp.PrimitiveClass{"g1 stream": cliapp.PrimitiveStreaming})
	if got := architectureStaticFindings(m, ev, "cli/manifest.json"); len(got) != 0 {
		t.Fatalf("streaming exception verified by streaming primitive should be clean, got %+v", got)
	}

	// No evidence -> unverified debt.
	if got := architectureStaticFindings(m, ArchitectureEvidence{}, "cli/manifest.json"); findingByCode(got, CodeArchPrimitiveUnverif) == nil {
		t.Fatalf("declared exception without evidence should be unverified, got %+v", got)
	}

	// Wrong evidence -> mismatch error.
	bad := evidence(map[string]cliapp.PrimitiveClass{"g1 stream": cliapp.PrimitiveUpload})
	if got := architectureStaticFindings(m, bad, "cli/manifest.json"); findingByCode(got, CodeArchPrimitiveMismatch) == nil {
		t.Fatalf("streaming exception satisfied by upload primitive should mismatch, got %+v", got)
	}
}

func TestArchitectureParseFindings_MalformedMetadataSurfacesArchCode(t *testing.T) {
	// A manifest whose architecture block is malformed (exception without a
	// reason) fails ParseManifest generically; architectureParseFindings recovers
	// the specific arch.metadata_invalid so the command_architecture capability is
	// impacted precisely.
	raw := []byte(`{
		"name": "fixture",
		"groups": [{"name": "g1", "commands": [{
			"name": "list",
			"binding": {"kind": "connect-rpc", "service": "Svc", "method": "List"},
			"governance": {"effect": "read"},
			"architecture": {"exception": {"class": "streaming"}}
		}]}]
	}`)
	// ParseManifest must reject it (proves the arch block is genuinely malformed).
	if _, err := cliapp.ParseManifest(raw); err == nil {
		t.Fatal("expected ParseManifest to reject an exception without a reason")
	}
	got := architectureParseFindings(raw, "cli/manifest.json")
	if findingByCode(got, CodeArchMetadataInvalid) == nil {
		t.Fatalf("expected metadata_invalid from architectureParseFindings, got %+v", got)
	}
}

func TestArchitectureParseFindings_CleanArchitectureYieldsNothing(t *testing.T) {
	// A parse failure unrelated to architecture must not fabricate arch findings.
	raw := []byte(`{
		"name": "fixture",
		"groups": [{"name": "g1", "commands": [{
			"name": "list",
			"binding": {"kind": "connect-rpc", "service": "Svc", "method": "List"},
			"governance": {"effect": "read"},
			"architecture": {"primitive": "proto_list"}
		}]}]
	}`)
	if got := architectureParseFindings(raw, "cli/manifest.json"); len(got) != 0 {
		t.Fatalf("well-formed architecture must yield no parse findings, got %+v", got)
	}
}

func TestArchitectureStatic_StaleArtifactIgnoresEvidenceAndWarns(t *testing.T) {
	// A stale artifact must NOT verify declared primitives (its evidence describes
	// an older surface), so the command falls back to unverified debt AND an
	// explicit arch.evidence_stale warning is emitted.
	m := manifestWith([]cliapp.ManifestCommand{cmd("list", "proto_list")}, nil)
	ev := ArchitectureEvidence{
		Primitives:   map[string]cliapp.PrimitiveClass{"g1 list": cliapp.PrimitiveProtoList},
		Status:       EvidenceArtifactStale,
		ArtifactPath: "cli/primitive-evidence.json",
	}
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	stale := findingByCode(got, CodeArchEvidenceStale)
	if stale == nil || stale.Severity != SeverityWarning {
		t.Fatalf("expected arch.evidence_stale warning, got %+v", got)
	}
	if findingByCode(got, CodeArchPrimitiveUnverif) == nil {
		t.Fatalf("stale evidence must not verify declared primitives; expected unverified, got %+v", got)
	}
}

func TestArchitectureStatic_MalformedArtifactIsGatingErrorAndIgnored(t *testing.T) {
	// A malformed artifact is a gating error and its evidence is ignored.
	m := manifestWith([]cliapp.ManifestCommand{cmd("list", "proto_list")}, nil)
	ev := ArchitectureEvidence{
		Primitives:   map[string]cliapp.PrimitiveClass{"g1 list": cliapp.PrimitiveProtoList},
		Status:       EvidenceArtifactMalformed,
		ArtifactPath: "cli/primitive-evidence.json",
		Detail:       "schema \"x/v9\" is not \"cli-primitive-evidence/v1\"",
	}
	got := architectureStaticFindings(m, ev, "cli/manifest.json")
	mal := findingByCode(got, CodeArchEvidenceMalformed)
	if mal == nil || mal.Severity != SeverityError {
		t.Fatalf("expected arch.evidence_malformed error, got %+v", got)
	}
	if findingByCode(got, CodeArchPrimitiveUnverif) == nil {
		t.Fatalf("malformed evidence must not verify declared primitives; expected unverified, got %+v", got)
	}
}

func TestExceptionCommandPaths_SilencesUndeclared(t *testing.T) {
	// A runtime command declared as an exception must not be flagged as an
	// undeclared command by the surface reconciler.
	m := manifestWith([]cliapp.ManifestCommand{cmd("list", "proto_list")}, []cliapp.ManifestException{{Command: "execute", Class: "durable_run", Reason: "run"}})
	obs := RuntimeObservation{Resolved: true, Commands: []RuntimeCommand{{Group: "", Name: "execute"}, {Group: "g1", Name: "list"}}}
	got := commandSurfaceFindings(obs, m, "cli/manifest.json")
	if findingByCode(got, CodeCLICommandUndeclared) != nil {
		t.Fatalf("declared exception command must not be flagged undeclared, got %+v", got)
	}
}
