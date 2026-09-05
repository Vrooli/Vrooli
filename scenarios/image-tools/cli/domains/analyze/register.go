package analyze

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the analyze subcommand group. `list` is bound from the
// manifest to AnalysisService.ListAnalysisOperations; the per-op commands drive
// the REST multipart edge and are appended directly (documented in the
// manifest's `omitted` array).
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AnalysisService.ListAnalysisOperations": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("analyze: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, h.runCommands()...)
	return group, nil
}

func (h *handlers) runCommands() []cliapp.Command {
	cmd := func(name, op, desc string) cliapp.Command {
		return cliapp.Command{
			Name:        name,
			Description: desc,
			NeedsAPI:    true,
			Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "input", Required: true, Description: "Path to the input image"}}},
			RunCtx:      h.run(op),
		}
	}
	return []cliapp.Command{
		cmd("probe", "probe", "Report structured image info (pure-Go, always available)"),
		cmd("ocr", "ocr", "Extract text from an image (OCR)"),
		cmd("nsfw", "nsfw_classify", "Classify an image for NSFW / unsafe content"),
		cmd("duplicate", "duplicate_detect", "Compute perceptual fingerprints for near-duplicate detection (pure-Go)"),
		cmd("quality", "quality_assessment", "Assess no-reference image quality — sharpness/exposure/contrast (pure-Go)"),
	}
}
