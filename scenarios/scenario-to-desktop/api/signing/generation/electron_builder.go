package generation

import (
	"scenario-to-desktop-api/signing/types"
)

// generateWindowsConfig creates electron-builder Windows signing configuration.
func generateWindowsConfig(config *types.WindowsSigningConfig, opts *Options) *types.ElectronBuilderWinSigning {
	if config == nil {
		return nil
	}

	win := &types.ElectronBuilderWinSigning{
		SignAndEditExecutable: true,
		SignDlls:              true,
	}

	// Configure certificate source
	switch config.CertificateSource {
	case types.CertSourceFile:
		win.CertificateFile = config.CertificateFile
		// Use environment variable reference for password
		if config.CertificatePasswordEnv != "" {
			win.CertificatePassword = "${" + config.CertificatePasswordEnv + "}"
		}
	case types.CertSourceStore:
		win.CertificateSha1 = config.CertificateThumbprint
		// Note: azure_keyvault and aws_kms require custom signtool configuration
		// which electron-builder doesn't directly support via simple config
	}

	// Configure timestamp server
	if config.TimestampServer != "" {
		win.Rfc3161TimeStampServer = config.TimestampServer
	} else {
		win.Rfc3161TimeStampServer = types.DefaultTimestampServerDigiCert
	}

	// Configure signing algorithms
	if config.DualSign {
		// Dual signing for Windows 7 compatibility: SHA-1 + SHA-256
		win.SigningHashAlgorithms = []string{"sha1", "sha256"}
	} else {
		// Single algorithm based on config or default to SHA-256
		algo := config.SignAlgorithm
		if algo == "" {
			algo = types.SignAlgorithmSHA256
		}
		win.SigningHashAlgorithms = []string{algo}
	}

	return win
}

// generateMacOSConfig creates electron-builder macOS signing configuration.
func generateMacOSConfig(config *types.MacOSSigningConfig, opts *Options) *types.ElectronBuilderMacSigning {
	if config == nil {
		return nil
	}

	mac := &types.ElectronBuilderMacSigning{
		Identity:         config.Identity,
		HardenedRuntime:  config.HardenedRuntime,
		GatekeeperAssess: true, // Always assess gatekeeper for proper signing verification
	}

	// Configure entitlements - use provided file or default generated path
	if config.EntitlementsFile != "" {
		mac.Entitlements = config.EntitlementsFile
		mac.EntitlementsInherit = config.EntitlementsFile
	} else if opts != nil && opts.EntitlementsPath != "" {
		entitlementsPath := opts.OutputDir + "/" + opts.EntitlementsPath
		mac.Entitlements = entitlementsPath
		mac.EntitlementsInherit = entitlementsPath
	}

	// Configure provisioning profile if provided
	if config.ProvisioningProfile != "" {
		mac.ProvisioningProfile = config.ProvisioningProfile
	}

	// Configure notarization
	if config.Notarize {
		// Use simple notarize config with team ID
		// electron-builder will look for credentials in environment
		if config.TeamID != "" {
			mac.Notarize = &types.NotarizeConfig{
				TeamID: config.TeamID,
			}
		} else {
			// If no team ID, just enable notarization
			// electron-builder will try to extract from identity
			mac.Notarize = true
		}
	}

	return mac
}

// GenerateElectronBuilderJSON generates the complete electron-builder config JSON
// including both signing and afterSign script reference.
// This is a convenience method that combines signing config with afterSign hook.
func GenerateElectronBuilderJSON(config *types.SigningConfig, opts *Options) (map[string]interface{}, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}

	if opts == nil {
		opts = DefaultOptions()
	}

	result := make(map[string]interface{})

	// Add Windows signing config
	if config.Windows != nil {
		winConfig := generateWindowsConfig(config.Windows, opts)
		if winConfig != nil {
			result["win"] = winConfig
		}
	}

	// Add macOS signing config
	if config.MacOS != nil {
		macConfig := generateMacOSConfig(config.MacOS, opts)
		if macConfig != nil {
			result["mac"] = macConfig
		}

		// Add afterSign hook for notarization
		if config.MacOS.Notarize {
			result["afterSign"] = opts.NotarizeScriptPath
		}
	}

	return result, nil
}
