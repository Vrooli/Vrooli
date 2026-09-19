package main

import (
	"testing"

	"scenario-to-desktop-api/scenario"
)

func TestDesktopScenarioStatusToProtoPreservesDesktopReadinessFields(t *testing.T) {
	t.Parallel()

	actual := desktopScenarioStatusToProto(scenario.ScenarioDesktopStatus{
		Name:                  "notes",
		DisplayName:           "Notes",
		ServiceDisplay:        "Notes Service",
		ServiceDesc:           "A desktop notes application",
		ServiceIconPath:       "/icons/notes.png",
		HasDesktop:            true,
		DesktopPath:           "/scenarios/notes/desktop",
		Version:               "1.2.3",
		Platforms:             []string{"linux", "darwin"},
		Built:                 true,
		DistPath:              "/dist",
		LastModified:          "2026-07-26T12:00:00Z",
		PackageSize:           42,
		ConnectionConfig:      &scenario.DesktopConnectionConfig{Mode: "managed", Endpoint: "http://127.0.0.1:9000"},
		BuildArtifacts:        []scenario.DesktopBuildArtifact{{Platform: "linux", FileName: "notes.AppImage", SizeBytes: 42, ModifiedAt: "2026-07-26T12:00:00Z", AbsolutePath: "/dist/notes.AppImage", RelativePath: "notes.AppImage"}},
		ArtifactsSource:       "dist",
		ArtifactsPath:         "/dist",
		ArtifactsExpectedPath: "/expected-dist",
		RecordID:              "record-1",
		RecordOutputPath:      "/output",
		RecordLocationMode:    "managed",
		RecordUpdatedAt:       "2026-07-26T12:00:00Z",
	})

	if actual.GetName() != "notes" || actual.GetDisplayName() != "Notes" || !actual.GetHasDesktop() || !actual.GetBuilt() {
		t.Fatalf("core status fields were not preserved: %#v", actual)
	}
	if actual.GetConnectionConfig().GetMode() != "managed" || actual.GetConnectionConfig().GetEndpoint() != "http://127.0.0.1:9000" {
		t.Fatalf("connection config was not preserved: %#v", actual.GetConnectionConfig())
	}
	if len(actual.GetBuildArtifacts()) != 1 || actual.GetBuildArtifacts()[0].GetRelativePath() != "notes.AppImage" {
		t.Fatalf("build artifacts were not preserved: %#v", actual.GetBuildArtifacts())
	}
	if actual.GetRecordId() != "record-1" || actual.GetRecordLocationMode() != "managed" || actual.GetArtifactsExpectedPath() != "/expected-dist" {
		t.Fatalf("record and artifact provenance fields were not preserved: %#v", actual)
	}
}

func TestProbeEndpointResultToProtoPreservesOptionalDiagnostics(t *testing.T) {
	t.Parallel()

	statusCode := 503
	actual := probeEndpointResultToProto(probeEndpointResult{
		Status:     "error",
		StatusCode: &statusCode,
		Message:    "server returned 503",
	})

	if actual.GetStatus() != "error" || actual.GetStatusCode() != 503 || actual.GetMessage() != "server returned 503" {
		t.Fatalf("probe result was not preserved: %#v", actual)
	}
	if probeEndpointResultToProto(probeEndpointResult{Status: "skipped"}).StatusCode != nil {
		t.Fatal("absent status code must remain absent")
	}
}
