package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ConfigStore provides in-memory caching of variant and branding configuration
// loaded from JSON files. It serves as the single source of truth for config,
// replacing the previous database-backed variant/branding storage.
type ConfigStore struct {
	mu           sync.RWMutex
	variants     map[string]*VariantSnapshot // slug -> full variant with sections
	branding     *SiteBranding
	variantsDir  string
	brandingPath string
	space        *VariantSpace
}

// NewConfigStore creates a new ConfigStore with paths to the JSON files.
func NewConfigStore(variantsDir, brandingPath string, space *VariantSpace) *ConfigStore {
	if space == nil {
		space = defaultVariantSpace
	}
	return &ConfigStore{
		variants:     make(map[string]*VariantSnapshot),
		variantsDir:  variantsDir,
		brandingPath: brandingPath,
		space:        space,
	}
}

// LoadAll loads all configuration from JSON files into memory.
func (cs *ConfigStore) LoadAll() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if err := cs.loadBrandingLocked(); err != nil {
		return fmt.Errorf("load branding: %w", err)
	}

	if err := cs.loadVariantsLocked(); err != nil {
		return fmt.Errorf("load variants: %w", err)
	}

	return nil
}

func (cs *ConfigStore) loadBrandingLocked() error {
	if cs.brandingPath == "" {
		cs.branding = defaultBranding()
		return nil
	}

	data, err := os.ReadFile(cs.brandingPath)
	if err != nil {
		if os.IsNotExist(err) {
			cs.branding = defaultBranding()
			logStructured("branding_file_not_found", map[string]interface{}{
				"path":     cs.brandingPath,
				"fallback": "default branding",
			})
			return nil
		}
		return err
	}

	var branding SiteBranding
	if err := json.Unmarshal(data, &branding); err != nil {
		return fmt.Errorf("parse branding JSON: %w", err)
	}

	// Set defaults for required fields
	if branding.SiteName == "" {
		branding.SiteName = "My Landing"
	}
	branding.ID = 1 // Singleton pattern
	branding.CreatedAt = time.Now()
	branding.UpdatedAt = time.Now()

	cs.branding = &branding
	logStructured("branding_loaded", map[string]interface{}{
		"path":      cs.brandingPath,
		"site_name": branding.SiteName,
	})
	return nil
}

func (cs *ConfigStore) loadVariantsLocked() error {
	if cs.variantsDir == "" {
		logStructured("variants_dir_not_set", map[string]interface{}{
			"fallback": "empty variants map",
		})
		return nil
	}

	entries, err := os.ReadDir(cs.variantsDir)
	if err != nil {
		if os.IsNotExist(err) {
			logStructured("variants_dir_not_found", map[string]interface{}{
				"path":     cs.variantsDir,
				"fallback": "empty variants map",
			})
			return nil
		}
		return err
	}

	cs.variants = make(map[string]*VariantSnapshot)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if entry.Name() == "fallback.json" {
			continue
		}

		path := filepath.Join(cs.variantsDir, entry.Name())
		if err := cs.loadVariantFileLocked(path); err != nil {
			logStructuredError("variant_load_failed", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			continue
		}
	}

	logStructured("variants_loaded", map[string]interface{}{
		"count": len(cs.variants),
		"dir":   cs.variantsDir,
	})
	return nil
}

func (cs *ConfigStore) loadVariantFileLocked(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var input VariantSnapshotInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("parse variant JSON: %w", err)
	}

	slug := strings.TrimSpace(input.Variant.Slug)
	if slug == "" {
		return errors.New("variant slug is required")
	}

	if len(input.Variant.Axes) == 0 {
		return fmt.Errorf("variant %s missing axes", slug)
	}

	// Normalize header config
	headerCfg := normalizeLandingHeaderConfig(input.Variant.HeaderConfig, input.Variant.Name)

	// Build sections
	sections := make([]VariantSection, 0, len(input.Sections))
	for idx, sec := range input.Sections {
		order := sec.Order
		if order <= 0 {
			order = idx + 1
		}
		enabled := true
		if sec.Enabled != nil {
			enabled = *sec.Enabled
		}
		sections = append(sections, VariantSection{
			SectionType: sec.SectionType,
			Content:     sec.Content,
			Order:       order,
			Enabled:     enabled,
		})
	}

	// Sort sections by order
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Order < sections[j].Order
	})

	snapshot := &VariantSnapshot{
		Variant: VariantSnapshotMeta{
			Slug:         slug,
			Name:         input.Variant.Name,
			Description:  input.Variant.Description,
			Axes:         input.Variant.Axes,
			HeaderConfig: headerCfg,
			SEOConfig:    input.Variant.SEOConfig,
		},
		Sections: sections,
	}

	cs.variants[slug] = snapshot
	return nil
}

// GetVariant retrieves a variant by slug from the in-memory cache.
func (cs *ConfigStore) GetVariant(slug string) (*VariantSnapshot, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	snapshot, ok := cs.variants[slug]
	if !ok {
		return nil, fmt.Errorf("variant %q not found", slug)
	}
	return snapshot, nil
}

// ListVariants returns all variants from the in-memory cache.
func (cs *ConfigStore) ListVariants() []*VariantSnapshot {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*VariantSnapshot, 0, len(cs.variants))
	for _, snapshot := range cs.variants {
		result = append(result, snapshot)
	}

	// Sort by slug for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Variant.Slug < result[j].Variant.Slug
	})

	return result
}

// SaveVariant writes a variant snapshot to its JSON file and updates the cache.
func (cs *ConfigStore) SaveVariant(slug string, snapshot *VariantSnapshot) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if strings.TrimSpace(slug) == "" {
		return errors.New("variant slug is required")
	}

	if snapshot.Variant.Slug != slug {
		return fmt.Errorf("snapshot slug %q does not match provided slug %q", snapshot.Variant.Slug, slug)
	}

	// Validate axes
	if len(snapshot.Variant.Axes) == 0 {
		return errors.New("axes selection is required")
	}
	if err := cs.space.ValidateSelection(snapshot.Variant.Axes); err != nil {
		return err
	}

	// Build the file structure
	fileData := VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug:         snapshot.Variant.Slug,
			Name:         snapshot.Variant.Name,
			Description:  snapshot.Variant.Description,
			Axes:         snapshot.Variant.Axes,
			HeaderConfig: &snapshot.Variant.HeaderConfig,
			SEOConfig:    snapshot.Variant.SEOConfig,
		},
		Sections: make([]VariantSectionInput, 0, len(snapshot.Sections)),
	}

	for _, sec := range snapshot.Sections {
		enabled := sec.Enabled
		fileData.Sections = append(fileData.Sections, VariantSectionInput{
			SectionType: sec.SectionType,
			Content:     sec.Content,
			Order:       sec.Order,
			Enabled:     &enabled,
		})
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal variant: %w", err)
	}

	// Write to file
	path := filepath.Join(cs.variantsDir, slug+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write variant file: %w", err)
	}

	// Update cache
	cs.variants[slug] = snapshot

	logStructured("variant_saved", map[string]interface{}{
		"slug": slug,
		"path": path,
	})
	return nil
}

// DeleteVariant removes a variant from the cache and deletes its JSON file.
func (cs *ConfigStore) DeleteVariant(slug string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, ok := cs.variants[slug]; !ok {
		return fmt.Errorf("variant %q not found", slug)
	}

	path := filepath.Join(cs.variantsDir, slug+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete variant file: %w", err)
	}

	delete(cs.variants, slug)

	logStructured("variant_deleted", map[string]interface{}{
		"slug": slug,
	})
	return nil
}

// GetBranding retrieves the site branding from the in-memory cache.
func (cs *ConfigStore) GetBranding() *SiteBranding {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.branding == nil {
		return defaultBranding()
	}
	return cs.branding
}

// SaveBranding writes branding configuration to its JSON file and updates the cache.
func (cs *ConfigStore) SaveBranding(branding *SiteBranding) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if branding.SiteName == "" {
		return errors.New("site_name is required")
	}

	// Build file structure (exclude internal fields like ID, CreatedAt, UpdatedAt)
	fileData := map[string]interface{}{
		"site_name": branding.SiteName,
	}

	if branding.Tagline != nil {
		fileData["tagline"] = *branding.Tagline
	}
	if branding.LogoURL != nil {
		fileData["logo_url"] = *branding.LogoURL
	}
	if branding.LogoIconURL != nil {
		fileData["logo_icon_url"] = *branding.LogoIconURL
	}
	if branding.FaviconURL != nil {
		fileData["favicon_url"] = *branding.FaviconURL
	}
	if branding.AppleTouchIconURL != nil {
		fileData["apple_touch_icon_url"] = *branding.AppleTouchIconURL
	}
	if branding.DefaultTitle != nil {
		fileData["default_title"] = *branding.DefaultTitle
	}
	if branding.DefaultDescription != nil {
		fileData["default_description"] = *branding.DefaultDescription
	}
	if branding.DefaultOGImageURL != nil {
		fileData["default_og_image_url"] = *branding.DefaultOGImageURL
	}
	if branding.ThemePrimaryColor != nil {
		fileData["theme_primary_color"] = *branding.ThemePrimaryColor
	}
	if branding.ThemeBackgroundColor != nil {
		fileData["theme_background_color"] = *branding.ThemeBackgroundColor
	}
	if branding.CanonicalBaseURL != nil {
		fileData["canonical_base_url"] = *branding.CanonicalBaseURL
	}
	if branding.GoogleSiteVerification != nil {
		fileData["google_site_verification"] = *branding.GoogleSiteVerification
	}
	if branding.RobotsTxt != nil {
		fileData["robots_txt"] = *branding.RobotsTxt
	}
	if branding.SupportChatURL != nil {
		fileData["support_chat_url"] = *branding.SupportChatURL
	}
	if branding.SupportEmail != nil {
		fileData["support_email"] = *branding.SupportEmail
	}
	if branding.SMTPHost != nil {
		fileData["smtp_host"] = *branding.SMTPHost
	}
	if branding.SMTPPort != nil {
		fileData["smtp_port"] = *branding.SMTPPort
	}
	if branding.SMTPUsername != nil {
		fileData["smtp_username"] = *branding.SMTPUsername
	}
	if branding.SMTPPassword != nil {
		fileData["smtp_password"] = *branding.SMTPPassword
	}
	if branding.SMTPFrom != nil {
		fileData["smtp_from"] = *branding.SMTPFrom
	}
	if branding.ComingSoonEnabled != nil {
		fileData["coming_soon_enabled"] = *branding.ComingSoonEnabled
	}
	if branding.ComingSoonMessage != nil {
		fileData["coming_soon_message"] = *branding.ComingSoonMessage
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal branding: %w", err)
	}

	if err := os.WriteFile(cs.brandingPath, data, 0o644); err != nil {
		return fmt.Errorf("write branding file: %w", err)
	}

	// Update cache
	branding.ID = 1
	branding.UpdatedAt = time.Now()
	cs.branding = branding

	logStructured("branding_saved", map[string]interface{}{
		"path":      cs.brandingPath,
		"site_name": branding.SiteName,
	})
	return nil
}

// UpdateBranding applies partial updates to branding and saves to JSON.
func (cs *ConfigStore) UpdateBranding(req *BrandingUpdateRequest) (*SiteBranding, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	current := cs.branding
	if current == nil {
		current = defaultBranding()
	}

	// Apply updates
	if req.SiteName != nil {
		current.SiteName = *req.SiteName
	}
	if req.Tagline != nil {
		current.Tagline = req.Tagline
	}
	if req.LogoURL != nil {
		current.LogoURL = req.LogoURL
	}
	if req.LogoIconURL != nil {
		current.LogoIconURL = req.LogoIconURL
	}
	if req.FaviconURL != nil {
		current.FaviconURL = req.FaviconURL
	}
	if req.AppleTouchIconURL != nil {
		current.AppleTouchIconURL = req.AppleTouchIconURL
	}
	if req.DefaultTitle != nil {
		current.DefaultTitle = req.DefaultTitle
	}
	if req.DefaultDescription != nil {
		current.DefaultDescription = req.DefaultDescription
	}
	if req.DefaultOGImageURL != nil {
		current.DefaultOGImageURL = req.DefaultOGImageURL
	}
	if req.ThemePrimaryColor != nil {
		current.ThemePrimaryColor = req.ThemePrimaryColor
	}
	if req.ThemeBackgroundColor != nil {
		current.ThemeBackgroundColor = req.ThemeBackgroundColor
	}
	if req.CanonicalBaseURL != nil {
		current.CanonicalBaseURL = req.CanonicalBaseURL
	}
	if req.GoogleSiteVerification != nil {
		current.GoogleSiteVerification = req.GoogleSiteVerification
	}
	if req.RobotsTxt != nil {
		current.RobotsTxt = req.RobotsTxt
	}
	if req.SupportChatURL != nil {
		current.SupportChatURL = req.SupportChatURL
	}
	if req.SupportEmail != nil {
		current.SupportEmail = req.SupportEmail
	}
	if req.SMTPHost != nil {
		current.SMTPHost = req.SMTPHost
	}
	if req.SMTPPort != nil {
		current.SMTPPort = req.SMTPPort
	}
	if req.SMTPUsername != nil {
		current.SMTPUsername = req.SMTPUsername
	}
	if req.SMTPPassword != nil {
		current.SMTPPassword = req.SMTPPassword
	}
	if req.SMTPFrom != nil {
		current.SMTPFrom = req.SMTPFrom
	}
	if req.ComingSoonEnabled != nil {
		current.ComingSoonEnabled = req.ComingSoonEnabled
	}
	if req.ComingSoonMessage != nil {
		current.ComingSoonMessage = req.ComingSoonMessage
	}

	cs.branding = current
	cs.mu.Unlock()

	// Save to file (releases lock first to avoid deadlock)
	if err := cs.SaveBranding(current); err != nil {
		cs.mu.Lock()
		return nil, err
	}

	cs.mu.Lock()
	return cs.branding, nil
}

// ClearBrandingField sets a specific branding field to nil and saves.
func (cs *ConfigStore) ClearBrandingField(field string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.branding == nil {
		return nil
	}

	switch field {
	case "tagline":
		cs.branding.Tagline = nil
	case "logo_url":
		cs.branding.LogoURL = nil
	case "logo_icon_url":
		cs.branding.LogoIconURL = nil
	case "favicon_url":
		cs.branding.FaviconURL = nil
	case "apple_touch_icon_url":
		cs.branding.AppleTouchIconURL = nil
	case "default_title":
		cs.branding.DefaultTitle = nil
	case "default_description":
		cs.branding.DefaultDescription = nil
	case "default_og_image_url":
		cs.branding.DefaultOGImageURL = nil
	case "theme_primary_color":
		cs.branding.ThemePrimaryColor = nil
	case "theme_background_color":
		cs.branding.ThemeBackgroundColor = nil
	case "canonical_base_url":
		cs.branding.CanonicalBaseURL = nil
	case "google_site_verification":
		cs.branding.GoogleSiteVerification = nil
	case "robots_txt":
		cs.branding.RobotsTxt = nil
	case "support_chat_url":
		cs.branding.SupportChatURL = nil
	case "support_email":
		cs.branding.SupportEmail = nil
	case "smtp_host":
		cs.branding.SMTPHost = nil
	case "smtp_port":
		cs.branding.SMTPPort = nil
	case "smtp_username":
		cs.branding.SMTPUsername = nil
	case "smtp_password":
		cs.branding.SMTPPassword = nil
	case "smtp_from":
		cs.branding.SMTPFrom = nil
	case "coming_soon_enabled":
		cs.branding.ComingSoonEnabled = nil
	case "coming_soon_message":
		cs.branding.ComingSoonMessage = nil
	default:
		return nil // Unknown field, ignore
	}

	// Save to file
	return cs.saveBrandingLocked()
}

func (cs *ConfigStore) saveBrandingLocked() error {
	if cs.branding == nil {
		return nil
	}

	fileData := map[string]interface{}{
		"site_name": cs.branding.SiteName,
	}

	if cs.branding.Tagline != nil {
		fileData["tagline"] = *cs.branding.Tagline
	}
	if cs.branding.LogoURL != nil {
		fileData["logo_url"] = *cs.branding.LogoURL
	}
	if cs.branding.LogoIconURL != nil {
		fileData["logo_icon_url"] = *cs.branding.LogoIconURL
	}
	if cs.branding.FaviconURL != nil {
		fileData["favicon_url"] = *cs.branding.FaviconURL
	}
	if cs.branding.AppleTouchIconURL != nil {
		fileData["apple_touch_icon_url"] = *cs.branding.AppleTouchIconURL
	}
	if cs.branding.DefaultTitle != nil {
		fileData["default_title"] = *cs.branding.DefaultTitle
	}
	if cs.branding.DefaultDescription != nil {
		fileData["default_description"] = *cs.branding.DefaultDescription
	}
	if cs.branding.DefaultOGImageURL != nil {
		fileData["default_og_image_url"] = *cs.branding.DefaultOGImageURL
	}
	if cs.branding.ThemePrimaryColor != nil {
		fileData["theme_primary_color"] = *cs.branding.ThemePrimaryColor
	}
	if cs.branding.ThemeBackgroundColor != nil {
		fileData["theme_background_color"] = *cs.branding.ThemeBackgroundColor
	}
	if cs.branding.CanonicalBaseURL != nil {
		fileData["canonical_base_url"] = *cs.branding.CanonicalBaseURL
	}
	if cs.branding.GoogleSiteVerification != nil {
		fileData["google_site_verification"] = *cs.branding.GoogleSiteVerification
	}
	if cs.branding.RobotsTxt != nil {
		fileData["robots_txt"] = *cs.branding.RobotsTxt
	}
	if cs.branding.SupportChatURL != nil {
		fileData["support_chat_url"] = *cs.branding.SupportChatURL
	}
	if cs.branding.SupportEmail != nil {
		fileData["support_email"] = *cs.branding.SupportEmail
	}
	if cs.branding.SMTPHost != nil {
		fileData["smtp_host"] = *cs.branding.SMTPHost
	}
	if cs.branding.SMTPPort != nil {
		fileData["smtp_port"] = *cs.branding.SMTPPort
	}
	if cs.branding.SMTPUsername != nil {
		fileData["smtp_username"] = *cs.branding.SMTPUsername
	}
	if cs.branding.SMTPPassword != nil {
		fileData["smtp_password"] = *cs.branding.SMTPPassword
	}
	if cs.branding.SMTPFrom != nil {
		fileData["smtp_from"] = *cs.branding.SMTPFrom
	}
	if cs.branding.ComingSoonEnabled != nil {
		fileData["coming_soon_enabled"] = *cs.branding.ComingSoonEnabled
	}
	if cs.branding.ComingSoonMessage != nil {
		fileData["coming_soon_message"] = *cs.branding.ComingSoonMessage
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal branding: %w", err)
	}

	if err := os.WriteFile(cs.brandingPath, data, 0o644); err != nil {
		return fmt.Errorf("write branding file: %w", err)
	}

	cs.branding.UpdatedAt = time.Now()
	return nil
}

// GetVariantsDir returns the path to the variants directory.
func (cs *ConfigStore) GetVariantsDir() string {
	return cs.variantsDir
}

// GetBrandingPath returns the path to the branding JSON file.
func (cs *ConfigStore) GetBrandingPath() string {
	return cs.brandingPath
}

// VariantCount returns the number of loaded variants.
func (cs *ConfigStore) VariantCount() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.variants)
}

func defaultBranding() *SiteBranding {
	return &SiteBranding{
		ID:        1,
		SiteName:  "My Landing",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
