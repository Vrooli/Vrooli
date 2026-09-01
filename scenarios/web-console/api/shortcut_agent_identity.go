// DOC: docs/concepts/ARCHITECTURE.md#file-map

package main

import (
	"strings"

	"github.com/vrooli/vrooli/packages/capabilityprobe"
)

// Which coding agent does a shortcut launch?
//
// Every surface used to answer this by pattern-matching the command string —
// the launcher grid did it in TypeScript, the settings editor could not do it
// at all, and neither could agree with the other. A guess over command text is
// also defeated by anything an operator writes around the agent: a wrapper
// script, a `cd … &&` prefix, an alias.
//
// So the answer is stored, not guessed: ShortcutEntry.AgentID is part of the
// wire contract. This file is the one place that still derives it, and it runs
// only when a caller supplies no id — an older client, or a profile row
// written before the field existed. Derivation is deliberately conservative:
// it recognises the shapes Web Console itself emits and otherwise returns "",
// which reads as "this is a plain operator command", the safe answer.

// agentAliases maps the tokens that can appear in a launch command to the
// capability id the catalogue uses. The ids come from capabilityprobe so a
// sixth agent cannot be added there and forgotten here; the extra keys are the
// alternate spellings that appear in real commands.
var agentAliases = buildAgentAliases()

func buildAgentAliases() map[string]string {
	aliases := map[string]string{
		// Alternate spellings. The catalogue id for Antigravity is "agy", and
		// Claude Code is installed as "claude" but named "claude-code" by the
		// resource installer.
		"antigravity": "agy",
		"claude-code": "claude",
	}
	for _, def := range capabilityprobe.AITools {
		aliases[def.ID] = def.ID
		if def.Command != "" {
			aliases[strings.ToLower(def.Command)] = def.ID
		}
	}
	return aliases
}

// agentDisplayNames is the catalogue's human name per capability id, used when
// a stored entry has no label of its own.
var agentDisplayNames = buildAgentDisplayNames()

func buildAgentDisplayNames() map[string]string {
	names := make(map[string]string, len(capabilityprobe.AITools))
	for _, def := range capabilityprobe.AITools {
		names[def.ID] = def.Label
	}
	return names
}

// normalizeAgentID maps a caller-supplied agent id onto the closed catalogue
// set. An unrecognised id becomes "" rather than being stored verbatim, so a
// typo cannot mint an agent that no probe will ever report.
func normalizeAgentID(id string) string {
	return agentAliases[strings.ToLower(strings.TrimSpace(id))]
}

// deriveAgentID infers the agent a command launches, or "" for a plain
// operator command.
//
// `--runner <name>` wins over a bare token because that is the governed form:
// `vrooli agent launch --runner claude …` names its agent explicitly, and the
// surrounding words ("vrooli", "agent", "launch") must not be allowed to vote.
func deriveAgentID(command string) string {
	fields := strings.Fields(command)
	for i, field := range fields {
		lower := strings.ToLower(field)
		if value, found := strings.CutPrefix(lower, "--runner="); found {
			if id := normalizeAgentID(value); id != "" {
				return id
			}
		}
		if lower == "--runner" && i+1 < len(fields) {
			if id := normalizeAgentID(fields[i+1]); id != "" {
				return id
			}
		}
		if value, found := strings.CutPrefix(lower, "--agent="); found {
			if id := normalizeAgentID(value); id != "" {
				return id
			}
		}
		if lower == "--agent" && i+1 < len(fields) {
			if id := normalizeAgentID(fields[i+1]); id != "" {
				return id
			}
		}
	}
	for _, field := range fields {
		// A bare invocation may carry a path ("/usr/bin/codex") or a shell
		// prefix ("exec codex"); the base name is what identifies the agent.
		token := strings.ToLower(field)
		if index := strings.LastIndexAny(token, "/\\"); index >= 0 {
			token = token[index+1:]
		}
		token = strings.TrimSuffix(token, ".exe")
		if id := agentAliases[token]; id != "" {
			return id
		}
	}
	return ""
}

// normalizeShortcutEntries fills in the agent id for every entry that lacks
// one, so a stored profile is self-describing from the moment it is read.
// Entries are returned as a new slice: callers hold onto the built-in default
// list, and mutating it in place would edit the defaults for the process.
func normalizeShortcutEntries(entries []ShortcutEntry) []ShortcutEntry {
	if entries == nil {
		return nil
	}
	out := make([]ShortcutEntry, len(entries))
	copy(out, entries)
	for i := range out {
		if id := normalizeAgentID(out[i].AgentID); id != "" {
			out[i].AgentID = id
			continue
		}
		out[i].AgentID = deriveAgentID(out[i].Command)
	}
	return out
}

// agentDisplayName returns the catalogue's name for an agent id, or "" when
// the id names no known agent.
func agentDisplayName(id string) string {
	return agentDisplayNames[normalizeAgentID(id)]
}
