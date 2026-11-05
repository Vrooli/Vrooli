package main

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	earningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning"
)

func TestManifestCoversEveryTypedServiceMethod(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, manifestBytes, accessv1.File_token_economy_v1_access_access_proto, "MinterService")
	cliapp.RequireProtoServiceCoverage(t, manifestBytes, accessv1.File_token_economy_v1_access_access_proto, "HolderService")
	cliapp.RequireProtoServiceCoverage(t, manifestBytes, earningv1.File_token_economy_v1_earning_earning_proto, "EarningService")
}

func TestEveryQueryUsesJSONCapableReportPrimitive(t *testing.T) {
	manifest, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			if command.Governance.Effect != "read" {
				continue
			}
			if got := command.Architecture.CommandArchitecture().Primitive; got != cliapp.PrimitiveProtoList {
				t.Errorf("query %s/%s primitive = %q, want proto_list for renderer-separated --json output", group.Name, command.Name, got)
			}
		}
	}
}
