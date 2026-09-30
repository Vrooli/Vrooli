package signing

import (
	"scenario-to-desktop-api/signing/generation"
	"scenario-to-desktop-api/signing/platforms"
	"scenario-to-desktop-api/signing/validation"
)

// Handler owns signing domain collaborators for the generated Connect service.
// Signing has no hand-authored HTTP control-plane routes.
type Handler struct {
	repo          Repository
	validator     Validator
	prereqChecker PrerequisiteChecker
	detector      CertificateDiscoverer
	generator     ConfigGenerator
}

func NewHandler() *Handler {
	return &Handler{
		repo: NewFileRepository(), validator: validation.NewValidator(),
		prereqChecker: validation.NewPrerequisiteChecker(), detector: platforms.NewMultiPlatformDetector(),
		generator: generation.NewGenerator(nil),
	}
}
