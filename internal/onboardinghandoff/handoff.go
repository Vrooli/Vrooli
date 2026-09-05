// Package onboardinghandoff contains the pure policy that maps an observed
// presentation capability to the surface setup should hand the operator to.
package onboardinghandoff

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostpresentation"
)

type Mode string

const (
	ModeAuto    Mode = "auto"
	ModeBrowser Mode = "browser"
	ModeCLI     Mode = "cli"
	ModeURL     Mode = "url"
	ModeNone    Mode = "none"
)

type Decision struct {
	Action        string
	Reason        string
	Kind          hostpresentation.Kind
	ResumeCommand string
}

type CLINeedsTerminalError struct{}

func (CLINeedsTerminalError) Error() string {
	return "cli onboarding needs a terminal on standard input; use --onboarding=url or run `vrooli-onboarding wizard run --interactive` from a terminal"
}

func ParseMode(value string) (Mode, error) {
	mode := Mode(strings.ToLower(strings.TrimSpace(value)))
	if mode == "" {
		return ModeAuto, nil
	}
	switch mode {
	case ModeAuto, ModeBrowser, ModeCLI, ModeURL, ModeNone:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid onboarding mode %q: accepted values are auto, browser, cli, url, none", value)
	}
}

func Decide(cap hostpresentation.Capability, mode Mode, stdinIsTTY bool) (Decision, error) {
	parsed, err := ParseMode(string(mode))
	if err != nil {
		return Decision{}, err
	}
	decision := Decision{Action: "url", Kind: cap.Kind, ResumeCommand: "vrooli-onboarding wizard run --interactive"}
	if parsed == ModeNone {
		decision.Action = "none"
		decision.ResumeCommand = ""
		decision.Reason = "onboarding disabled by operator"
		return decision, nil
	}
	if parsed == ModeCLI {
		if !stdinIsTTY {
			return Decision{}, CLINeedsTerminalError{}
		}
		decision.Action = "cli"
		decision.Reason = "interactive terminal requested"
		return decision, nil
	}
	if parsed == ModeURL {
		decision.Reason = "URL handoff requested"
		return decision, nil
	}
	if cap.Degraded {
		decision.Reason = "presentation detection degraded; browser opening is disabled"
		return decision, nil
	}
	if parsed == ModeBrowser {
		decision.Action = "browser"
		decision.Reason = "browser handoff requested"
		return decision, nil
	}
	switch cap.Kind {
	case hostpresentation.KindLocalGraphical, hostpresentation.KindWSLGraphical, hostpresentation.KindRemoteDesktop:
		decision.Action = "browser"
		decision.Reason = cap.Reason
	case hostpresentation.KindForwardedGraphical:
		decision.Reason = "forwarded graphical sessions use a URL handoff under auto"
	default:
		decision.Reason = cap.Reason
	}
	return decision, nil
}
