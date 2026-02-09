package main

import "testing"

func TestRemoteProfileSessionMetadataBuildParse(t *testing.T) {
	agent := buildRemoteProfileSessionUserAgent(RemoteProfileSessionMetadata{
		ConnectorID: "connector-123",
		ProfileTag:  "prod",
		Origin:      "local-host",
	})
	meta, ok := parseRemoteProfileSessionUserAgent(agent)
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
	if _, ok := parseRemoteProfileSessionUserAgent("Mozilla/5.0"); ok {
		t.Fatalf("expected parse to fail for non-connector user agent")
	}
}
