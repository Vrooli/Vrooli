// Handoff payload rendering.
//
// This module is the send path's only text transformation. It deliberately
// imports nothing: not the capture-rule matcher, not the path helpers, not
// the conversation store. Deleting every capture rule must not change one
// line of behaviour here, and the only way to guarantee that mechanically is
// for this file to have no way to reach them.
//
// [REQ:P0-014d] Handoff Between Sessions In A Group

/** The placeholder a role's incoming prompt may contain, at most once. */
export const PAYLOAD_PLACEHOLDER = "{{payload}}";

/**
 * Substitutes the payload into a role's incoming prompt.
 *
 * The template language is deliberately one placeholder and nothing else. A
 * richer language would need escaping rules, a parser, and error states, and
 * would let a template encode workflow knowledge that belongs to the operator.
 * An empty template means "compose the message by hand".
 *
 * Three cases, and nothing else:
 *   - empty template  → the payload alone.
 *   - no placeholder  → the template, then the payload after a blank line.
 *   - placeholder     → every occurrence replaced.
 *
 * This function never inspects the payload. It does not look at its file
 * extension, its prefix, or its content, because nothing in this console is
 * allowed to know that a payload might be a plan.
 */
export function renderHandoffPrompt(template: string, payload: string): string {
  if (!template.trim()) return payload;
  if (!template.includes(PAYLOAD_PLACEHOLDER)) {
    return payload ? `${template}\n\n${payload}` : template;
  }
  return template.split(PAYLOAD_PLACEHOLDER).join(payload);
}

/**
 * What happened to one target of a handoff.
 *
 * `queued` is a first-class outcome, not a variant of success. A session
 * created moments ago has no mounted terminal, so its text sits in the pending
 * queue until the terminal registers. Collapsing it into `sent` would tell the
 * operator their message arrived when it has not.
 */
export type HandoffDeliveryStatus = "sent" | "queued" | "failed";

/** One target's outcome, named so the composer can report per target. */
export interface HandoffResult {
  /** The target's stable id: a session id for a running target, a role id otherwise. */
  targetId: string;
  /** What the operator sees in the result line. */
  label: string;
  status: HandoffDeliveryStatus;
  /** Present when status is "queued" or "failed"; explains which. */
  reason?: string;
}

/** True when every target reached a terminal. */
export function allDelivered(results: readonly HandoffResult[]): boolean {
  return results.length > 0 && results.every((r) => r.status === "sent");
}

/** True when any target's text is still unsent, queued or failed. */
export function anyUnsent(results: readonly HandoffResult[]): boolean {
  return results.some((r) => r.status !== "sent");
}
