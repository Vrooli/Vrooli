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
// the known Whisper sizes. It is the single validator for the capacity-degrade
// verb and the pin reader.
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

// PinPath resolves the operator/broker model-pin file. The pin is how the
// capacity broker's degrade actuation (`capacity-degrade --to <model>`) forces a
// smaller model: it persists the choice so the next managed-service start —
// which harvests `recommend-model --env` — comes up at the pinned size instead
// of the host-derived recommendation. An explicit env override wins (tests/sandboxes);
// otherwise it lives beside the capacity ledger under the runtime-home state dir.
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

// WritePin persists the model pin (creating the state dir if needed).
func WritePin(getEnv func(string) string, model Model) error {
	path, err := PinPath(getEnv)
	if err != nil {
		return err
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(string(model)+"\n"), 0o644)
}

// ClearPin removes the model pin (used by upshift). A missing pin is a no-op.
func ClearPin(getEnv func(string) string) error {
	path, err := PinPath(getEnv)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
