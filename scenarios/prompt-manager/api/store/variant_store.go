package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FileVariantStore implements VariantStore using the file system.
// Variants are stored alongside their parent skill:
//
//	store/skills/packs/{pack}/{skill-id}/variants/{variant-id}/variant.json + VARIANT.md
type FileVariantStore struct {
	skillStore *FileSkillStore
}

// NewFileVariantStore creates a new file-based variant store.
func NewFileVariantStore(skillStore *FileSkillStore) *FileVariantStore {
	return &FileVariantStore{skillStore: skillStore}
}

// variantsDir returns the variants directory for a skill within a specific pack.
func (s *FileVariantStore) variantsDir(pack, skillID string) string {
	return filepath.Join(s.skillStore.packsDir(), pack, skillID, "variants")
}

// variantDir returns the directory for a specific variant.
func (s *FileVariantStore) variantDir(pack, skillID, variantID string) string {
	return filepath.Join(s.variantsDir(pack, skillID), variantID)
}

// resolveSkillPack finds the pack for a skill, returning the skill and its pack name.
func (s *FileVariantStore) resolveSkillPack(ctx context.Context, skillID string) (*Skill, error) {
	skill, err := s.skillStore.Get(ctx, skillID)
	if err != nil {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}
	return skill, nil
}

// List returns all variants for a skill.
func (s *FileVariantStore) List(ctx context.Context, skillID string) ([]Variant, error) {
	skill, err := s.resolveSkillPack(ctx, skillID)
	if err != nil {
		return nil, err
	}

	varDir := s.variantsDir(skill.Pack, skillID)
	dirs, err := ListDirectories(varDir)
	if err != nil {
		return nil, nil // No variants directory = no variants
	}

	var variants []Variant
	for _, vid := range dirs {
		v, err := s.loadVariant(skill.Pack, skillID, vid)
		if err != nil {
			continue // Skip malformed variants
		}
		variants = append(variants, *v)
	}

	return variants, nil
}

// Get retrieves a variant by skill ID and variant ID.
func (s *FileVariantStore) Get(ctx context.Context, skillID, variantID string) (*Variant, error) {
	skill, err := s.resolveSkillPack(ctx, skillID)
	if err != nil {
		return nil, err
	}

	v, err := s.loadVariant(skill.Pack, skillID, variantID)
	if err != nil {
		return nil, fmt.Errorf("variant not found: %s/%s", skillID, variantID)
	}
	return v, nil
}

// GetWithContent retrieves a variant with its markdown content.
func (s *FileVariantStore) GetWithContent(ctx context.Context, skillID, variantID string) (*Variant, string, error) {
	skill, err := s.resolveSkillPack(ctx, skillID)
	if err != nil {
		return nil, "", err
	}

	v, err := s.loadVariant(skill.Pack, skillID, variantID)
	if err != nil {
		return nil, "", fmt.Errorf("variant not found: %s/%s", skillID, variantID)
	}

	contentPath := filepath.Join(s.variantDir(skill.Pack, skillID, variantID), v.Entry)
	content, err := ReadContent(contentPath)
	if err != nil {
		return v, "", fmt.Errorf("reading variant content: %w", err)
	}

	return v, content, nil
}

// Create creates a new variant for a skill.
func (s *FileVariantStore) Create(ctx context.Context, skillID string, variant *Variant, content string) error {
	skill, err := s.resolveSkillPack(ctx, skillID)
	if err != nil {
		return err
	}

	if variant.ID == ControlVariantID {
		return fmt.Errorf("cannot use reserved variant ID %q", ControlVariantID)
	}

	vDir := s.variantDir(skill.Pack, skillID, variant.ID)
	if FileExists(filepath.Join(vDir, "variant.json")) {
		return fmt.Errorf("variant already exists: %s/%s", skillID, variant.ID)
	}

	variant.Kind = KindVariant
	variant.SchemaVersion = CurrentSchemaVersion
	variant.SkillID = skillID
	variant.Entry = "VARIANT.md"
	variant.Timestamps = NewTimestamps()

	if err := os.MkdirAll(vDir, 0o755); err != nil {
		return fmt.Errorf("creating variant directory: %w", err)
	}

	if err := SaveJSON(filepath.Join(vDir, "variant.json"), variant); err != nil {
		return fmt.Errorf("writing variant.json: %w", err)
	}

	if err := WriteContent(filepath.Join(vDir, "VARIANT.md"), content); err != nil {
		return fmt.Errorf("writing variant content: %w", err)
	}

	return nil
}

// Update updates an existing variant.
func (s *FileVariantStore) Update(ctx context.Context, skillID, variantID string, updates *Variant, content *string) error {
	skill, err := s.resolveSkillPack(ctx, skillID)
	if err != nil {
		return err
	}

	v, err := s.loadVariant(skill.Pack, skillID, variantID)
	if err != nil {
		return fmt.Errorf("variant not found: %s/%s", skillID, variantID)
	}

	if updates.Name != "" {
		v.Name = updates.Name
	}
	if updates.Description != "" {
		v.Description = updates.Description
	}
	v.UpdateTimestamp()

	vDir := s.variantDir(skill.Pack, skillID, variantID)

	if err := SaveJSON(filepath.Join(vDir, "variant.json"), v); err != nil {
		return fmt.Errorf("writing variant.json: %w", err)
	}

	if content != nil {
		if err := WriteContent(filepath.Join(vDir, v.Entry), *content); err != nil {
			return fmt.Errorf("writing variant content: %w", err)
		}
	}

	return nil
}

// Delete removes a variant.
func (s *FileVariantStore) Delete(ctx context.Context, skillID, variantID string) error {
	skill, err := s.resolveSkillPack(ctx, skillID)
	if err != nil {
		return err
	}

	vDir := s.variantDir(skill.Pack, skillID, variantID)
	if !FileExists(filepath.Join(vDir, "variant.json")) {
		return fmt.Errorf("variant not found: %s/%s", skillID, variantID)
	}

	return DeleteDirectory(vDir)
}

// loadVariant loads a variant from its JSON file.
func (s *FileVariantStore) loadVariant(pack, skillID, variantID string) (*Variant, error) {
	path := filepath.Join(s.variantDir(pack, skillID, variantID), "variant.json")
	return LoadJSON[Variant](path)
}
