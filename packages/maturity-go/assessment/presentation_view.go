package assessment

import (
	"fmt"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// PhasePresentationView is a deliberately shallow rendering adapter for direct
// operator clients. It consumes the provider-owned projection as-is: in
// particular, it does not sort, group, prioritize, or derive maturity data.
// Native provider detail remains separately available on the response.
type PhasePresentationView struct {
	Summary string
	Lines   []string
}

// PresentationView turns a v1 PhasePresentation into concise terminal text.
// An absent or unknown version is explicitly degraded rather than reconstructed
// from the assessment, because only the provider owns the phase story.
func PresentationView(p *commonv1.PhasePresentation) PhasePresentationView {
	if p == nil {
		return PhasePresentationView{
			Summary: "presentation: unavailable (provider returned no PhasePresentation)",
			Lines:   []string{"Raw assessment and provider-native detail remain available."},
		}
	}
	if p.GetContractVersion() != PhasePresentationContractVersion {
		return PhasePresentationView{
			Summary: fmt.Sprintf("presentation: historical or unsupported contract %q", p.GetContractVersion()),
			Lines:   []string{"Raw assessment and provider-native detail remain available; no phase story was synthesized."},
		}
	}

	level := firstNonBlank(p.GetCurrentLevelLabel(), p.GetCurrentLevel(), "unknown")
	view := PhasePresentationView{Summary: fmt.Sprintf("%s/%s: %s", p.GetProvider(), p.GetPhase(), level)}
	for _, capability := range p.GetCapabilities() {
		if capability == nil {
			continue
		}
		label := firstNonBlank(capability.GetLabel(), capability.GetId(), "capability")
		current := firstNonBlank(capability.GetCurrentLevelLabel(), capability.GetCurrentLevel(), "unknown")
		line := fmt.Sprintf("%s: %s", label, current)
		if next := strings.TrimSpace(capability.GetNextUnlock()); next != "" {
			line += fmt.Sprintf(" — next: %s", next)
		}
		view.Lines = append(view.Lines, line)
	}
	if action := strings.TrimSpace(p.GetNextAction()); action != "" {
		view.Lines = append(view.Lines, "Next action: "+action)
	}
	if northStar := strings.TrimSpace(p.GetNorthStar()); northStar != "" {
		view.Lines = append(view.Lines, "North star: "+northStar)
	}
	if topics := p.GetDocumentationTopics(); len(topics) > 0 {
		view.Lines = append(view.Lines, "Documentation: "+strings.Join(topics, ", "))
	}
	if len(view.Lines) == 0 {
		view.Lines = append(view.Lines, "No capability detail was supplied by the provider.")
	}
	return view
}
