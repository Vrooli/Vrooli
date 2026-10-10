// finite_installation_association_test.go verifies owner association fails before publication.
package heartbeat

import (
	"github.com/vrooli/api-core/effortauthority"
	"testing"
)

func TestFiniteInstallationIncompleteGraphLeavesAuthorityUnpublished(t *testing.T) {
	tasks := &AgentManagerClient{}
	runtime := &FiniteLeaderRuntime{}
	prepared := &PreparedFiniteInstallation{config: effortauthority.Installation{Enabled: true}, tasks: tasks, efforts: &FiniteEffortAuthority{}}
	if err := prepared.Apply(runtime); err == nil {
		t.Fatal("incomplete graph admitted")
	}
	if runtime.Efforts != nil || tasks.efforts != nil {
		t.Fatal("refusal published finite authority")
	}
}
func TestFiniteInstallationDisabledLeavesGraphUntouched(t *testing.T) {
	authority := &FiniteEffortAuthority{}
	runtime := &FiniteLeaderRuntime{Efforts: authority}
	prepared := &PreparedFiniteInstallation{}
	if err := prepared.Apply(runtime); err != nil {
		t.Fatal(err)
	}
	if runtime.Efforts != authority {
		t.Fatal("disabled installation changed owner")
	}
}
