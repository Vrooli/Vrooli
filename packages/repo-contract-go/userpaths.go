package repocontract

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	vrooliUserDirName          = ".vrooli"
	plaintextSecretsFilename   = "secrets.json"
	encryptedSecretsFilename   = "secrets.enc.json"
	userScenarioSecretsDirName = "scenarios"
)

func VrooliUserRoot(home string) (string, error) {
	home = filepath.Clean(strings.TrimSpace(home))
	if home == "" {
		return "", &Error{Kind: ErrInvalidInput, Message: "user home dir is required"}
	}
	return filepath.Join(home, vrooliUserDirName), nil
}

func VrooliUserRootFromOS() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return VrooliUserRoot(home)
}

func UserPlaintextSecretsPath(home string) (string, error) {
	root, err := VrooliUserRoot(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, plaintextSecretsFilename), nil
}

func UserEncryptedSecretsPath(home string) (string, error) {
	root, err := VrooliUserRoot(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, encryptedSecretsFilename), nil
}

func UserScenarioPlaintextSecretsPath(home, scenario string) (string, error) {
	root, err := VrooliUserRoot(home)
	if err != nil {
		return "", err
	}
	scenario, err = cleanIdentifier(scenario)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, userScenarioSecretsDirName, scenario, plaintextSecretsFilename), nil
}
