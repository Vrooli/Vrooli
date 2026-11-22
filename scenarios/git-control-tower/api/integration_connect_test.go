package main

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/collaboration"
	integrationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/integration"
)

func TestIntegrationConnectGetCapabilitiesPreservesRegistryFields(t *testing.T) {
	registry := NewCapabilityRegistry(
		[]CapabilityDef{{
			ID: "scenario-auditor", Name: "Scenario Auditor", Description: "Validates scenario standards",
			DependencyKind: DependencyScenario, DependencySlug: "scenario-auditor", Features: []string{"standards"},
		}},
		map[string]StatusChecker{"scenario-auditor": &FakeStatusChecker{Status: StatusAvailable}},
		time.Minute,
	)
	server := integrationConnectServer{server: &Server{capabilities: registry}}
	response, err := server.GetCapabilities(context.Background(), connect.NewRequest(&integrationv1.GetCapabilitiesRequest{}))
	if err != nil {
		t.Fatalf("GetCapabilities returned error: %v", err)
	}
	if response.Msg.GetTimestamp() == "" || len(response.Msg.GetCapabilities()) != 1 {
		t.Fatalf("unexpected capability response: %#v", response.Msg)
	}
	capability := response.Msg.GetCapabilities()[0].AsMap()
	if capability["id"] != "scenario-auditor" || capability["status"] != "available" || capability["dependencySlug"] != "scenario-auditor" {
		t.Fatalf("capability registry fields were lost: %#v", capability)
	}
}

func TestIntegrationConnectGetCollaborationStatusPreservesValidationAndUnavailableState(t *testing.T) {
	server := integrationConnectServer{}
	invalid, err := server.GetCollaborationStatus(context.Background(), connect.NewRequest(&integrationv1.GetCollaborationStatusRequest{}))
	if err != nil {
		t.Fatalf("invalid but unconfigured identity should remain a readable status: %v", err)
	}
	if invalid.Msg.GetStatus() != "unconfigured" || invalid.Msg.GetHost() != nil || invalid.Msg.GetReason() == "" || len(invalid.Msg.GetCapabilities()) != 6 {
		t.Fatalf("unexpected unconfigured response: %#v", invalid.Msg)
	}
	valid, err := server.GetCollaborationStatus(context.Background(), connect.NewRequest(&integrationv1.GetCollaborationStatusRequest{
		Kind: "github", InstanceUrl: " https://github.example ", RepositoryId: " org/repo ",
	}))
	if err != nil {
		t.Fatalf("valid identity should return the owner's unavailable state: %v", err)
	}
	if valid.Msg.GetStatus() != "unavailable" || valid.Msg.GetHost().GetInstanceUrl() != "https://github.example" || valid.Msg.GetReason() != "Integration Hub host adapter is not active" {
		t.Fatalf("unexpected unavailable response: %#v", valid.Msg)
	}
	if valid.Msg.GetCapabilities()[0].GetCapability() != string(collaboration.CapabilityReadChange) {
		t.Fatalf("capability ordering changed: %#v", valid.Msg.GetCapabilities())
	}
}
