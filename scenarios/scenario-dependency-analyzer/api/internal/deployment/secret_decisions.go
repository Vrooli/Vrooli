package deployment

import (
	"encoding/json"
	"fmt"
	"os"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type declaredCredentials struct {
	Descriptors []struct {
		LogicalID   string `json:"logical_id"`
		Field       string `json:"field"`
		Label       string `json:"label"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
	} `json:"descriptors"`
}

// ReadSecretRequirements reads the resource owner's credential declarations.
// The analyzer does not infer secret needs from a resource name.
func ReadSecretRequirements(manifestPath, resourceName, resourceType string) ([]types.SecretRequirement, error) {
	raw, err := os.ReadFile(manifestPath) // #nosec G304 -- path is the resource manifest path carried by the DAG node.
	if err != nil {
		return nil, fmt.Errorf("read credential declarations: %w", err)
	}
	var manifest struct {
		Credentials declaredCredentials `json:"credentials"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("parse credential declarations: %w", err)
	}
	result := make([]types.SecretRequirement, 0, len(manifest.Credentials.Descriptors))
	for _, descriptor := range manifest.Credentials.Descriptors {
		if descriptor.LogicalID == "" && descriptor.Field == "" {
			continue
		}
		secretType := descriptor.Label
		if secretType == "" {
			secretType = descriptor.Field
		}
		result = append(result, types.SecretRequirement{
			DependencyName:    resourceName,
			DependencyType:    resourceType,
			SecretType:        secretType,
			RequiredSecrets:   []string{descriptor.LogicalID},
			PlaybookReference: descriptor.Description,
			Priority:          priority(descriptor.Required),
		})
	}
	return result, nil
}

func priority(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}
