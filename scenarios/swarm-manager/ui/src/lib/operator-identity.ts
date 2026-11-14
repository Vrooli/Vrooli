/**
 * Remembers who is deciding.
 *
 * Review decisions require an operator identity and a rationale. The identity
 * was a free-text field with no default, retyped for every decision in a queue
 * that routinely holds hundreds — which is both friction and an attribution
 * hazard, because a typo'd actor silently splits one person's audit trail
 * across several names.
 *
 * Deliberately localStorage rather than a proto field: this is a client-side
 * convenience default, not a claim about who the operator is. The server still
 * records whatever was actually submitted.
 */
const STORAGE_KEY = "swarm-manager:operator-identity:v1";

export function readOperatorIdentity(): string {
  if (typeof window === "undefined") return "";
  try {
    return window.localStorage.getItem(STORAGE_KEY)?.trim() ?? "";
  } catch {
    // Private-mode or blocked storage must not break the decision surface.
    return "";
  }
}

export function rememberOperatorIdentity(value: string): void {
  if (typeof window === "undefined") return;
  const trimmed = value.trim();
  try {
    if (trimmed) window.localStorage.setItem(STORAGE_KEY, trimmed);
    else window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Non-fatal: the operator simply retypes next time.
  }
}
