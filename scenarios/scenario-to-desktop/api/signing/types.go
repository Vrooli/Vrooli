// Package signing provides code signing configuration, validation,
// and generation for desktop bundle deployments.
//
// This package handles platform-specific signing requirements for Windows
// (Authenticode), macOS (Developer ID + Notarization), and Linux (GPG).
package signing

// Re-export all types from the types subpackage for convenience.
// This allows consumers to use signing.SigningConfig instead of signing/types.SigningConfig.
import (
	"scenario-to-desktop-api/signing/types"
)

// Type aliases for external consumers
type (
	SigningConfig                = types.SigningConfig
	WindowsSigningConfig         = types.WindowsSigningConfig
	MacOSSigningConfig           = types.MacOSSigningConfig
	LinuxSigningConfig           = types.LinuxSigningConfig
	ValidationResult             = types.ValidationResult
	PlatformValidation           = types.PlatformValidation
	CertificateInfo              = types.CertificateInfo
	ValidationError              = types.ValidationError
	ValidationWarning            = types.ValidationWarning
	ElectronBuilderSigningConfig = types.ElectronBuilderSigningConfig
	ElectronBuilderWinSigning    = types.ElectronBuilderWinSigning
	ElectronBuilderMacSigning    = types.ElectronBuilderMacSigning
	NotarizeConfig               = types.NotarizeConfig
	ToolDetectionResult          = types.ToolDetectionResult
	DiscoveredCertificate        = types.DiscoveredCertificate
	ReadinessResponse            = types.ReadinessResponse
	PlatformStatus               = types.PlatformStatus
	SigningConfigResponse        = types.SigningConfigResponse
)

// Constant re-exports
const (
	PlatformWindows = types.PlatformWindows
	PlatformMacOS   = types.PlatformMacOS
	PlatformLinux   = types.PlatformLinux

	CertSourceFile          = types.CertSourceFile
	CertSourceStore         = types.CertSourceStore
	CertSourceAzureKeyVault = types.CertSourceAzureKeyVault
	CertSourceAWSKMS        = types.CertSourceAWSKMS

	SignAlgorithmSHA256 = types.SignAlgorithmSHA256
	SignAlgorithmSHA384 = types.SignAlgorithmSHA384
	SignAlgorithmSHA512 = types.SignAlgorithmSHA512

	DefaultTimestampServerDigiCert   = types.DefaultTimestampServerDigiCert
	DefaultTimestampServerSectigo    = types.DefaultTimestampServerSectigo
	DefaultTimestampServerGlobalSign = types.DefaultTimestampServerGlobalSign

	SchemaVersion = types.SchemaVersion
)
