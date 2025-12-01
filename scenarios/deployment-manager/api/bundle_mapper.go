package main

import (
	"encoding/json"
	"fmt"
)

// secretsManagerBundleSecret represents bundle_secrets from secrets-manager.
type secretsManagerBundleSecret struct {
	ID          string                 `json:"id"`
	Class       string                 `json:"class"`
	Required    bool                   `json:"required"`
	Description string                 `json:"description"`
	Format      string                 `json:"format"`
	Target      secretTarget           `json:"target"`
	Prompt      *secretPrompt          `json:"prompt,omitempty"`
	Generator   map[string]interface{} `json:"generator,omitempty"`
}

type secretTarget struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type secretPrompt struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// applyBundleSecrets copies bundle secret plans into the bundle manifest and re-validates it.
func applyBundleSecrets(manifest *desktopBundleManifest, secrets []secretsManagerBundleSecret) error {
	if manifest == nil {
		return fmt.Errorf("manifest is nil")
	}

	converted := make([]manifestSecret, 0, len(secrets))
	for _, s := range secrets {
		if s.Target.Type != "env" && s.Target.Type != "file" {
			return fmt.Errorf("secret %s has unsupported target type %s", s.ID, s.Target.Type)
		}
		var prompt *manifestSecretPrompt
		if s.Prompt != nil {
			prompt = &manifestSecretPrompt{
				Label:       s.Prompt.Label,
				Description: s.Prompt.Description,
			}
		}
		converted = append(converted, manifestSecret{
			ID:          s.ID,
			Class:       s.Class,
			Description: s.Description,
			Format:      s.Format,
			Required:    &s.Required,
			Prompt:      prompt,
			Generator:   s.Generator,
			Target: manifestSecretTarget{
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
	if err := validateDesktopBundleManifestBytes(payload); err != nil {
		return fmt.Errorf("manifest failed validation after merging secrets: %w", err)
	}
	return nil
}
