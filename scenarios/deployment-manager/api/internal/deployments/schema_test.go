package deployments

import "testing"

func TestSchemaIsTheDeploymentDomainSchema(t *testing.T) {
	if len(Schema()) == 0 {
		t.Fatal("deployment schema is empty")
	}
}
