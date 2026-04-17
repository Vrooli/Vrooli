package sshidentity

import (
	"encoding/json"
	"scenario-to-cloud/domain"
)

// FromDeployment extracts canonical SSH identity from deployment state.
func FromDeployment(dep *domain.Deployment) (DeploymentSSHIdentity, error) {
	if dep == nil || !dep.SSHIdentity.Valid || len(dep.SSHIdentity.Data) == 0 {
		return DeploymentSSHIdentity{AuthMode: AuthModeUnknown, VerificationState: VerificationUnknown}, nil
	}
	return Unmarshal(dep.SSHIdentity.Data)
}

// ToNullRawMessage marshals identity into domain.NullRawMessage for persistence.
func ToNullRawMessage(identity DeploymentSSHIdentity) (domain.NullRawMessage, error) {
	data, err := Marshal(identity)
	if err != nil {
		return domain.NullRawMessage{}, err
	}
	return domain.NullRawMessage{Data: json.RawMessage(data), Valid: true}, nil
}
