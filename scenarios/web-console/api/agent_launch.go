package main

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	"web-console/handlers/modelselection"
	sessionsH "web-console/handlers/sessions"
)

// launcherFormCommand is the attributed launch, which takes runner arguments as
// repeated --arg tokens. A bare agent command takes them directly.
var (
	launcherFormCommand = regexp.MustCompile(`^vrooli\s+agent\s+launch(\s|$)`)
	launcherRunnerArg   = regexp.MustCompile(`--runner[=\s]+(\S+)`)
	shellSyntax         = regexp.MustCompile("[;&|`$()<>\r\n]")
	// A command names its model as -m x, -mx, --model x, --model=x, or a -c model=x
	// config override, each of which may be wrapped in the launcher's --arg= form.
	namedModelArg = regexp.MustCompile(`(^|\s)(--arg=)?(-m|--model(\s|=|$)|(-c|--config)(=|\s+(--arg=)?)"?model=)`)
)

// composableLaunchCommand says whether settings can be added to a command: a plain
// start of this harness that does not already name its own model or limit. Anything
// else is the operator's own command, which runs exactly as written.
func composableLaunchCommand(descriptor harness.Descriptor, command string) bool {
	text := strings.TrimSpace(command)
	if text == "" || shellSyntax.MatchString(text) || namedModelArg.MatchString(text) {
		return false
	}
	if limit := descriptor.Features.ContextLimit; limit != nil && strings.Contains(text, limit.Key) {
		return false
	}
	if launcherFormCommand.MatchString(text) {
		match := launcherRunnerArg.FindStringSubmatch(text)
		if match == nil {
			return false
		}
		runner, known := harness.Active().Lookup(match[1])
		return known && runner.ID == descriptor.ID
	}
	first := strings.Fields(text)[0]
	return first == descriptor.Binary || first == descriptor.ID
}

// composeAgentLaunch applies one launch's settings to its command. The client sends
// intent, and this resolves it against the same catalog the choices came from, so a
// model the catalog no longer offers, a limit that model cannot take, or a metered
// choice from a stale list is refused here rather than only hidden in a browser.
func (s *Server) composeAgentLaunch(ctx context.Context, intent sessionsH.AgentLaunch, command, targetID string) (string, error) {
	descriptor, ok := harness.Active().ByID(strings.TrimSpace(intent.AgentID))
	if !ok {
		return "", fmt.Errorf("%w: unknown agent %q", sessionsH.ErrInvalidArgument, intent.AgentID)
	}
	if intent.Model == "" && intent.Provider == "" && intent.ContextLimitTokens == 0 {
		return command, nil
	}
	if !composableLaunchCommand(descriptor, command) {
		return "", fmt.Errorf("%w: this shortcut runs its own command, so launch settings cannot apply", sessionsH.ErrInvalidArgument)
	}
	policy := s.GetRoleModelChoices(ctx, descriptor.ID, targetID, command)
	composed := strings.TrimSpace(command)
	var chosen *modelselection.Choice
	if intent.Model != "" {
		chosen = resolveLaunchChoice(policy.Choices, intent.Model, intent.Provider)
		if chosen == nil {
			return "", fmt.Errorf("%w: model %q is not offered for %s", sessionsH.ErrInvalidArgument, intent.Model, descriptor.DisplayName)
		}
		if chosen.Metered && policy.CatalogStale {
			return "", fmt.Errorf("%w: %s bills per use and this model list is out of date; refresh it first", sessionsH.ErrFailedPrecondition, chosen.DisplayName)
		}
		if policy.ModelFlag == "" {
			return "", fmt.Errorf("%w: %s has no verified model argument", sessionsH.ErrFailedPrecondition, descriptor.DisplayName)
		}
		composed = appendLaunchArgs(composed, policy.ModelFlag, chosen.LaunchModel)
	}
	if intent.ContextLimitTokens != 0 {
		if err := checkContextLimit(descriptor, policy, chosen, intent.ContextLimitTokens); err != nil {
			return "", err
		}
		composed = applyContextLimit(descriptor, composed, intent.ContextLimitTokens)
	}
	return composed, nil
}

// resolveLaunchChoice finds the offered model. With no provider named, as a usage
// mode's full-power launch sends, one that does not bill per use wins, so nobody
// starts paying because a model is also reachable through a metered source.
func resolveLaunchChoice(choices []modelselection.Choice, model, provider string) *modelselection.Choice {
	var found *modelselection.Choice
	for index := range choices {
		candidate := &choices[index]
		if candidate.Model != model || (provider != "" && candidate.Source != provider) {
			continue
		}
		if !candidate.Metered {
			return candidate
		}
		if found == nil {
			found = candidate
		}
	}
	return found
}

func appendLaunchArgs(command string, args ...string) string {
	parts := []string{command}
	launcher := launcherFormCommand.MatchString(command)
	for _, arg := range args {
		if launcher {
			arg = "--arg=" + arg
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}

func checkContextLimit(descriptor harness.Descriptor, policy modelselection.Result, chosen *modelselection.Choice, tokens int64) error {
	if descriptor.Features.ContextLimit == nil {
		return fmt.Errorf("%w: %s does not take a compaction limit", sessionsH.ErrInvalidArgument, descriptor.DisplayName)
	}
	// With no explicit model, a usage mode may pick which one runs, so any model's
	// offered limit stands. An explicit model answers for itself.
	offered := policy.Choices
	if chosen != nil {
		offered = []modelselection.Choice{*chosen}
	}
	for _, choice := range offered {
		if slices.Contains(choice.ContextLimitTokens, tokens) {
			return nil
		}
	}
	return fmt.Errorf("%w: a limit of %d is not offered here", sessionsH.ErrInvalidArgument, tokens)
}

func applyContextLimit(descriptor harness.Descriptor, command string, tokens int64) string {
	limit := descriptor.Features.ContextLimit
	value := strconv.FormatInt(tokens, 10)
	if limit.Kind == harness.ContextLimitEnv {
		return limit.Key + "=" + value + " " + command
	}
	if launcherFormCommand.MatchString(command) {
		return appendLaunchArgs(command, "-c", limit.Key+"="+value)
	}
	binary, rest, _ := strings.Cut(command, " ")
	return strings.TrimSpace(strings.Join([]string{binary, "-c", limit.Key + "=" + value, rest}, " "))
}
