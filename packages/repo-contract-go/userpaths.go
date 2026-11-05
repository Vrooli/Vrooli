package repocontract

// User runtime-home helpers. These resolve well-known files under the operator
// runtime home ($HOME/.vrooli) from the contract's runtime_home authority —
// there are NO hard-coded filenames here. The caller supplies `home` (resolved
// sudo-aware where relevant); the structural names come from the loaded
// contract. Contract-load failure is a hard error (no fallback).

// VrooliUserRoot returns the runtime-home root ($home/.vrooli) for the given OS
// home dir, using the contract's runtime_home.dir_name.
func VrooliUserRoot(home string) (string, error) {
	contract, err := loadRuntimeHomeContract()
	if err != nil {
		return "", err
	}
	return contract.RuntimeHome(home)
}

// UserPlaintextSecretsPath returns the plaintext user secrets file path.
func UserPlaintextSecretsPath(home string) (string, error) {
	contract, err := loadRuntimeHomeContract()
	if err != nil {
		return "", err
	}
	entry, err := contract.RuntimeHomeEntry(home, "secrets")
	if err != nil {
		return "", err
	}
	return entry.AbsPath, nil
}

// UserEncryptedSecretsPath returns the encrypted user secrets file path.
func UserEncryptedSecretsPath(home string) (string, error) {
	contract, err := loadRuntimeHomeContract()
	if err != nil {
		return "", err
	}
	entry, err := contract.RuntimeHomeEntry(home, "secrets_enc")
	if err != nil {
		return "", err
	}
	return entry.AbsPath, nil
}

// UserScenarioPlaintextSecretsPath returns the per-scenario plaintext secrets
// file path under the runtime home.
func UserScenarioPlaintextSecretsPath(home, scenario string) (string, error) {
	contract, err := loadRuntimeHomeContract()
	if err != nil {
		return "", err
	}
	return contract.ScopedRuntimePath(home, "scenario_secrets", map[string]string{"scenario": scenario})
}

// loadRuntimeHomeContract resolves the active repo root and loads the canonical
// contract that holds the runtime_home authority. No fallback: a missing or
// invalid contract is a hard error so structural names can never silently drift.
func loadRuntimeHomeContract() (*Contract, error) {
	contract, _, err := LoadDefaultFromEnvOrCWD()
	if err != nil {
		return nil, err
	}
	return contract, nil
}
