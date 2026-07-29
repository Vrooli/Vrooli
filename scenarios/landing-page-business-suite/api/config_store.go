package main

import "landing-page-business-suite-api/internal/experimentation"

type (
	ConfigStore              = experimentation.ConfigStore
	ConfigStoreReader        = experimentation.ConfigStoreReader
	ConfigStoreWriter        = experimentation.ConfigStoreWriter
	ConfigStorer             = experimentation.ConfigStorer
	ConfigStoreOptions       = experimentation.ConfigStoreOptions
	VariantSnapshot          = experimentation.VariantSnapshot
	VariantSnapshotMeta      = experimentation.VariantSnapshotMeta
	VariantSnapshotMetaInput = experimentation.VariantSnapshotMetaInput
	VariantSnapshotInput     = experimentation.VariantSnapshotInput
	VariantSection           = experimentation.VariantSection
	VariantSectionInput      = experimentation.VariantSectionInput
	SiteBranding             = experimentation.SiteBranding
	BrandingUpdateRequest    = experimentation.BrandingUpdateRequest
	LandingHeaderConfig      = experimentation.LandingHeaderConfig
	HeaderBrandingConfig     = experimentation.HeaderBrandingConfig
	HeaderNavConfig          = experimentation.HeaderNavConfig
	HeaderNavLink            = experimentation.HeaderNavLink
	HeaderVisibilityConfig   = experimentation.HeaderVisibilityConfig
	HeaderCTAGroup           = experimentation.HeaderCTAGroup
	HeaderCTAConfig          = experimentation.HeaderCTAConfig
	HeaderBehaviorConfig     = experimentation.HeaderBehaviorConfig
)

func NewConfigStore(variantsDir, brandingPath string, space *VariantSpace) *ConfigStore {
	return experimentation.NewConfigStoreWithOptions(ConfigStoreOptions{
		VariantsDir:  variantsDir,
		BrandingPath: brandingPath,
		Space:        space,
		Log:          logStructured,
	})
}

func NewConfigStoreWithOptions(opts ConfigStoreOptions) *ConfigStore {
	if opts.Log == nil {
		opts.Log = logStructured
	}
	return experimentation.NewConfigStoreWithOptions(opts)
}

func defaultLandingHeaderConfig(variantName string) LandingHeaderConfig {
	return experimentation.DefaultLandingHeaderConfig(variantName)
}

func normalizeLandingHeaderConfig(input *LandingHeaderConfig, variantName string) LandingHeaderConfig {
	return experimentation.NormalizeLandingHeaderConfig(input, variantName)
}
