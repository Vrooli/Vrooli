package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"audio-tools/cli/domains"
)

func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "audio-tools",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}

func TestModelCommandsHaveVerifiedPrimitiveEvidence(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    "audio-tools",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	})
	if err != nil {
		t.Fatalf("build evidence: %v", err)
	}
	observed := artifact.ObservedPrimitives()
	for _, path := range []string{"model list", "model resolve"} {
		got, ok := observed[path]
		if !ok {
			t.Errorf("command %q has no observed primitive evidence", path)
			continue
		}
		if got != cliapp.PrimitiveProtoList {
			t.Errorf("command %q observed primitive = %q, want %q", path, got, cliapp.PrimitiveProtoList)
		}
	}
}

func TestSTTCommandsHaveExpectedPrimitiveEvidence(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario: "audio-tools", ManifestRaw: manifestBytes, Groups: groups,
	})
	if err != nil {
		t.Fatalf("build evidence: %v", err)
	}
	observed := artifact.ObservedPrimitives()
	for _, path := range []string{"stt transcribe", "stt formats", "stt engines", "stt engine-impact", "stt stream-config", "stt stream-config-set", "stt speaker-status", "stt speaker-config", "stt speaker-enroll", "stt speaker-clips", "stt speaker-delete-clip"} {
		if got := observed[path]; got != cliapp.PrimitiveAction {
			t.Errorf("command %q observed primitive = %q, want action", path, got)
		}
	}
	if got := observed["stt transcribe-stream"]; got != cliapp.PrimitiveStreaming {
		t.Errorf("streaming exception evidence = %q, want streaming lifecycle evidence", got)
	}
}
