package recommend

import (
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

// EnvPinFile overrides the model-pin file path (tests, sandboxes).
const EnvPinFile = "WHISPER_MODEL_PIN_FILE"

// ValidModel maps a label to its canonical Model, reporting whether it is one of
// the known Whisper sizes for the independent recommendation read surface.
func ValidModel(label string) (Model, bool) {
	switch Model(strings.TrimSpace(label)) {
	case ModelTiny:
		return ModelTiny, true
	case ModelBase:
		return ModelBase, true
	case ModelSmall:
		return ModelSmall, true
	case ModelMedium:
		return ModelMedium, true
	case ModelLargeV3:
		return ModelLargeV3, true
	default:
		return "", false
	}
}

// PinPath resolves the independent recommendation model-pin file. An explicit
// env override wins (tests/sandboxes); otherwise it lives beside the capacity
// ledger under the runtime-home state dir.
func PinPath(getEnv func(string) string) (string, error) {
	if getEnv == nil {
		getEnv = os.Getenv
	}
	if override := strings.TrimSpace(getEnv(EnvPinFile)); override != "" {
		return override, nil
	}
	home, err := config.HomeDir()
	if err != nil {
		return "", err
	}
	stateDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "whisper-model.pin"), nil
}

// ReadPin returns the pinned model when a valid pin file exists. ok is false
// (with no error) when there is no pin or it is unreadable/invalid — the caller
// then falls back to the host-derived recommendation.
func ReadPin(getEnv func(string) string) (Model, bool) {
	path, err := PinPath(getEnv)
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return ValidModel(string(data))
}
