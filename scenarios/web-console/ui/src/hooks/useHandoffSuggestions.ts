import { useCallback, useEffect, useMemo, useState } from "react";

import { listHandoffRules, type HandoffRuleDTO } from "../api/handoffrules";
import { matchRules, suggestionsForEvent, type HandoffSuggestion } from "../lib/captureRules";
import { getSessionConversationEvents, useConversationStore } from "../stores/useConversationStore";

// [REQ:P0-014h] Handoff Capture Rules
//
// This hook reads rules and events and produces suggestions. It cannot send
// anything: the composer it opens is the same one a button opens, reached
// through an ordinary callback.

/**
 * The handoff suggestions currently on offer in one session.
 *
 * The scan is memoized on the EVENT COUNT rather than on the array identity:
 * the conversation store replaces its array on every append, so keying on
 * identity would re-scan on each render for no new information.
 */
export function useHandoffSuggestions(sessionId: string) {
  const [rules, setRules] = useState<HandoffRuleDTO[]>([]);
  const [dismissed, setDismissed] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    let cancelled = false;
    listHandoffRules()
      .then((next) => { if (!cancelled) setRules(next); })
      .catch((error: unknown) => { console.error("Failed to load handoff rules:", error); });
    return () => { cancelled = true; };
  }, []);

  const events = useConversationStore((state) => getSessionConversationEvents(state, sessionId));
  const eventCount = events.length;

  const suggestions = useMemo(
    () => (rules.length === 0 ? [] : matchRules(rules, events)),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- keyed on the count, not the array identity: the store replaces the array on every append.
    [rules, eventCount, sessionId],
  );

  const visible = useMemo(
    () => suggestions.filter((s) => !dismissed.has(`${s.ruleId}:${s.eventId}:${s.payload}`)),
    [dismissed, suggestions],
  );

  /** Dismiss one suggestion for this session's lifetime. */
  const dismiss = useCallback((suggestion: HandoffSuggestion) => {
    setDismissed((prev) => {
      const next = new Set(prev);
      next.add(`${suggestion.ruleId}:${suggestion.eventId}:${suggestion.payload}`);
      return next;
    });
  }, []);

  const forEvent = useCallback(
    (eventId: string) => suggestionsForEvent(visible, eventId),
    [visible],
  );

  return { suggestions: visible, forEvent, dismiss };
}
