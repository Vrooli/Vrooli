package deployability

import "strings"

// ToolAcquisitionDeclaration is the manifest subset required to prove that a
// declared macOS tool has an acquisition path. It deliberately accepts
// normalized inputs so the rule remains independent of manifest I/O.
type ToolAcquisitionDeclaration struct {
	Platforms []HostOS
	Brew      string
	Source    string
	Handler   string
	Manual    bool
}

// ValidateMacOSAcquisition returns an error when a tool claims macOS support
// without a package, source, custom handler, or explicit manual path.
func ValidateMacOSAcquisition(declaration ToolAcquisitionDeclaration) error {
	// An omitted platforms field means the tool applies to every platform
	// according to tool.schema.json, so it claims macOS as well.
	claimsMacOS := len(declaration.Platforms) == 0
	for _, platform := range declaration.Platforms {
		if platform == HostOSMacOS {
			claimsMacOS = true
			break
		}
	}
	if !claimsMacOS || declaration.Manual || strings.TrimSpace(declaration.Brew) != "" || strings.TrimSpace(declaration.Source) != "" || strings.TrimSpace(declaration.Handler) != "" {
		return nil
	}
	return errMissingMacOSAcquisitionPath{}
}

type errMissingMacOSAcquisitionPath struct{}

func (errMissingMacOSAcquisitionPath) Error() string {
	return "tool declares macOS support without a brew package, source target, handler, or manual acquisition path"
}
