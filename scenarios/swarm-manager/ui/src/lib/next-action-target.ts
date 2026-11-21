/**
 * Next-action target parsing
 *
 * The server encodes an action's subject in a single `target` string, using a
 * `prefix:value` shape (`milestone_review:release-1`). Every surface that acts
 * on a next action has to decode it, and when two surfaces disagree on the
 * prefix list one of them silently stops acting — the button renders, the
 * click resolves to nothing, and there is no error to see.
 *
 * The prefixes are declared once here.
 *
 * Server side: `api/internal/goals/next_action.go`.
 */

/** Prefixes whose value names a milestone within the action's goal. */
export const MILESTONE_TARGET_PREFIXES = [
  "milestone_review:",
  "milestone_criteria:",
] as const;

/**
 * Returns the value carried by `target` under `prefix`, or "" when the target
 * is absent, differently prefixed, or has an empty value.
 */
export function actionTargetSuffix(target: string | undefined, prefix: string): string {
  if (!target || !target.startsWith(prefix)) return "";
  return target.slice(prefix.length).trim();
}

/**
 * Milestone named by a milestone-scoped action target, or "" for any other
 * action.
 */
export function milestoneTargetOf(target: string | undefined): string {
  for (const prefix of MILESTONE_TARGET_PREFIXES) {
    const value = actionTargetSuffix(target, prefix);
    if (value) return value;
  }
  return "";
}
