package deployment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	deploymentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment"
)

func TestDeploymentManifestCoversDeploymentService(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, manifest, deploymentv1.File_decentralized_deploy_v1_deployment_deployment_proto, "DeploymentService")
}
