package main

import (
	"testing"

	"landing-page-business-suite-api/internal/administration"
)

func TestRemoteProfileSessionMetadataBuildParse(t *testing.T) {
	agent := administration.BuildRemoteProfileSessionUserAgent(administration.RemoteProfileSessionMetadata{
		ConnectorID: "connector-123",
		ProfileTag:  "prod",
		Origin:      "local-host",
	})
	meta, ok := administration.ParseRemoteProfileSessionUserAgent(agent)
	if !ok {
		t.Fatalf("expected metadata to parse")
	}
	if meta.ConnectorID != "connector-123" {
		t.Fatalf("unexpected connector id %q", meta.ConnectorID)
	}
	if meta.ProfileTag != "prod" {
		t.Fatalf("unexpected profile tag %q", meta.ProfileTag)
	}
	if meta.Origin != "local-host" {
		t.Fatalf("unexpected origin %q", meta.Origin)
	}
}

func TestRemoteProfileSessionMetadataParseRejectsNonConnectorAgent(t *testing.T) {
	if _, ok := administration.ParseRemoteProfileSessionUserAgent("Mozilla/5.0"); ok {
		t.Fatalf("expected parse to fail for non-connector user agent")
	}
}
