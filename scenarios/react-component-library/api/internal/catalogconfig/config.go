package catalogconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// DeclaredMaturityFloor reads the catalog's single adoption policy value.
func DeclaredMaturityFloor(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read catalog maturity floor: %w", err)
	}
	var config struct {
		Floor string `json:"adoptionMaturityFloor"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return "", fmt.Errorf("parse catalog maturity floor: %w", err)
	}
	if strings.TrimSpace(config.Floor) == "" {
		return "", fmt.Errorf("catalog config must declare adoptionMaturityFloor")
	}
	return strings.TrimSpace(config.Floor), nil
}
