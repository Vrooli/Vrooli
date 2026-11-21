/**
 * Browser-side attribution helper.
 *
 * The full canon is docs/agent-system/RUNTIME_ATTRIBUTION.md. UI clients
 * are always treated as `kind: "operator-direct"` (a human at a browser),
 * so this helper produces a single fixed payload — no env-var inheritance,
 * no per-call attribution construction.
 *
 * Mirrors:
 *  - scenarios/prompt-manager/api/store/models.go (AttributionInfo, KnowledgeKind*)
 *  - scenarios/prompt-manager/cli/internal/attribution/attribution.go (Info, Encode)
 *
 * If the API-side struct changes shape, update this file to match.
 */

export const ATTRIBUTION_HEADER_NAME = 'X-Vrooli-Attribution'

interface AttributionPayload {
  kind: 'operator-direct'
  member_id: null
  team_id: null
  run_id: null
  spawn_origin: 'operator-cli'
  source_skill_id: null
}

const operatorDirectPayload: AttributionPayload = {
  kind: 'operator-direct',
  member_id: null,
  team_id: null,
  run_id: null,
  spawn_origin: 'operator-cli',
  source_skill_id: null,
}

/**
 * Returns the X-Vrooli-Attribution header value for a UI-originated
 * mutating request. The value is a base64-encoded JSON payload claiming
 * `operator-direct` (the canonical UI caller kind).
 *
 * Cached at module scope — the payload is immutable for the lifetime of
 * the page, so we avoid re-encoding on every request.
 */
let cachedHeaderValue: string | null = null

export function operatorDirectAttributionHeaderValue(): string {
  if (cachedHeaderValue === null) {
    cachedHeaderValue = btoa(JSON.stringify(operatorDirectPayload))
  }
  return cachedHeaderValue
}

/**
 * Returns a header object suitable for spreading into a fetch
 * RequestInit.headers. Convenience wrapper.
 */
export function operatorDirectAttributionHeaders(): Record<string, string> {
  return { [ATTRIBUTION_HEADER_NAME]: operatorDirectAttributionHeaderValue() }
}
