// When the console OFFERS a handoff.
//
// A rule produces a SUGGESTION and nothing else. This module has no way to
// send anything: it imports no transport, no session manager, and nothing
// from lib/handoff or hooks/useHandoff. That separation is the safety
// property that makes operator-authored patterns shippable — a wrong rule
// costs a dismissed chip, never a message delivered to an agent — and the
// only way to guarantee it mechanically is for the two sides to share no code.
//
// [REQ:P0-014h] Handoff Capture Rules

import type { ConversationEvent } from "../api/conversation";
import type { HandoffRuleDTO } from "../api/handoffrules";
import { matchProseFilePaths } from "./fileReferences";
import { isWindowsPath } from "./paths";

/**
 * How far back a scan looks.
 *
 * Bounded so cost does not grow with transcript length: a session running for
 * a day would otherwise re-scan thousands of events on every render.
 */
export const SCAN_EVENT_LIMIT = 50;

/**
 * The longest event text a rule is run against.
 *
 * An operator-authored regular expression can backtrack catastrophically, and
 * the input length is the one factor this module controls. A pathological
 * pattern on a bounded input stays bounded.
 */
export const MAX_SCANNED_TEXT_LENGTH = 20_000;

/** One offer, ready to open the composer. */
export interface HandoffSuggestion {
  /** The rule that fired, so a wrong suggestion names the rule to edit. */
  ruleId: string;
  ruleName: string;
  /** The event the suggestion attaches to. */
  eventId: string;
  /** The text a handoff would carry. Never classified by this module. */
  payload: string;
}

/**
 * Compile a glob into a regular expression.
 *
 * Supports the two forms an operator actually writes: `*` for one segment and
 * `**` for any depth. Everything else is escaped, so a pattern containing
 * regular-expression syntax matches literally rather than surprising the
 * author who wrote it as a path.
 */
function globToRegExp(glob: string, windows: boolean): RegExp {
  const sep = windows ? "\\\\" : "/";
  const notSep = windows ? "[^\\\\]" : "[^/]";
  let out = "";
  for (let i = 0; i < glob.length; i++) {
    const ch = glob[i];
    if (ch === undefined) continue;
    if (ch === "*") {
      if (glob[i + 1] === "*") {
        // `**/` crosses any number of segments, including none.
        if (glob[i + 2] === "/" || glob[i + 2] === "\\") {
          out += `(?:.*${sep})?`;
          i += 2;
        } else {
          out += ".*";
          i += 1;
        }
      } else {
        out += `${notSep}*`;
      }
      continue;
    }
    if (ch === "?") {
      out += notSep;
      continue;
    }
    if (ch === "/" || ch === "\\") {
      out += sep;
      continue;
    }
    out += ch.replace(/[.+^${}()|[\]\\]/g, "\\$&");
  }
  // Anchored at the end but not the start, so `*.md` matches a full path the
  // way an operator expects rather than only a bare filename.
  return new RegExp(`(?:^|${sep})${out}$`, "i");
}

/** Does this path match the glob, in the path's own separator flavour? */
export function matchesGlob(glob: string, path: string): boolean {
  if (!glob) return false;
  try {
    return globToRegExp(glob, isWindowsPath(path)).test(path);
  } catch {
    // A pattern that will not compile matches nothing rather than throwing
    // into a render.
    return false;
  }
}

/** Compile a rule's regular expression, or null when it will not compile. */
function safeRegExp(pattern: string): RegExp | null {
  try {
    return new RegExp(pattern);
  } catch {
    return null;
  }
}

/**
 * Every suggestion the enabled rules produce for a bounded window of events.
 *
 * Returns an empty list for an empty rule list, which is the state an operator
 * who wants no suggestions is entitled to — and in which every other handoff
 * entry point keeps working.
 */
export function matchRules(
  rules: readonly HandoffRuleDTO[],
  events: readonly ConversationEvent[],
): HandoffSuggestion[] {
  const enabled = rules.filter((rule) => rule.enabled && rule.pattern);
  if (enabled.length === 0) return [];

  const window = events.slice(-SCAN_EVENT_LIMIT);
  const suggestions: HandoffSuggestion[] = [];
  const seen = new Set<string>();

  for (const event of window) {
    const text = event.text.length > MAX_SCANNED_TEXT_LENGTH
      ? event.text.slice(0, MAX_SCANNED_TEXT_LENGTH)
      : event.text;
    if (!text) continue;

    // Paths are extracted once per event, not once per rule: the extractor is
    // the expensive half and every file_path rule wants the same list.
    let paths: string[] | null = null;

    for (const rule of enabled) {
      if (rule.source === "file_path") {
        paths ??= matchProseFilePaths(text).map((match) => match.path);
        for (const path of paths) {
          if (!matchesGlob(rule.pattern, path)) continue;
          const key = `${rule.id}:${event.id}:${path}`;
          if (seen.has(key)) continue;
          seen.add(key);
          suggestions.push({ ruleId: rule.id, ruleName: rule.name, eventId: event.id, payload: path });
        }
        continue;
      }

      const expression = safeRegExp(rule.pattern);
      if (!expression) continue;
      const match = expression.exec(text);
      if (!match) continue;
      // The first capture group is the payload when the author supplied one;
      // otherwise the whole match is.
      const payload = match[1] ?? match[0];
      if (!payload) continue;
      const key = `${rule.id}:${event.id}:${payload}`;
      if (seen.has(key)) continue;
      seen.add(key);
      suggestions.push({ ruleId: rule.id, ruleName: rule.name, eventId: event.id, payload });
    }
  }

  return suggestions;
}

/** The suggestions attached to one event, for rendering beneath it. */
export function suggestionsForEvent(
  suggestions: readonly HandoffSuggestion[],
  eventId: string,
): HandoffSuggestion[] {
  return suggestions.filter((suggestion) => suggestion.eventId === eventId);
}
