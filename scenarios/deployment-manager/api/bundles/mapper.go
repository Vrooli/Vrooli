package bundles

import (
	"encoding/json"
	"fmt"

	"deployment-manager/secrets"
	"deployment-manager/shared"
)

// ApplyBundleSecrets copies bundle secret plans into the bundle manifest and re-validates it.
func ApplyBundleSecrets(manifest *Manifest, bundleSecrets []secrets.BundleSecret) error {
	if manifest == nil {
		return fmt.Errorf("manifest is nil")
	}

	converted := make([]ManifestSecret, 0, len(bundleSecrets))
	for _, s := range bundleSecrets {
		// Skip infrastructure secrets - they should NEVER be bundled for security
		if !shared.IsBundleSafeSecretClass(s.Class) {
			continue
		}
		if s.Target.Type != "env" && s.Target.Type != "file" {
			return fmt.Errorf("secret %s has unsupported target type %s", s.ID, s.Target.Type)
		}
		var prompt *SecretPrompt
		if s.Prompt != nil {
			prompt = &SecretPrompt{
				Label:       s.Prompt.Label,
				Description: s.Prompt.Description,
			}
		}
		// Create a local copy of Required to avoid loop variable pointer aliasing (Go < 1.22)
		required := s.Required
		converted = append(converted, ManifestSecret{
			ID:          s.ID,
			Class:       s.Class,
			Description: s.Description,
			Format:      s.Format,
			Required:    &required,
			Prompt:      prompt,
			Generator:   s.Generator,
			Target: SecretTarget{
				Type: s.Target.Type,
				Name: s.Target.Name,
			},
		})
	}

	manifest.Secrets = converted

	// Validate the updated manifest using existing lightweight validation.
	payload, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := ValidateManifestBytes(payload); err != nil {
		return fmt.Errorf("manifest failed validation after merging secrets: %w", err)
	}
	return nil
}
