/**
 * Documentation navigation helpers for health checks
 */

/**
 * Get the documentation path for a given check ID.
 * Maps check IDs to their corresponding documentation files.
 */
export function getDocsPathForCheck(checkId: string): string {
  // Infrastructure checks have individual docs
  const infraChecks = ["infra-network", "infra-dns", "infra-docker", "infra-cloudflared", "infra-rdp"];
  if (infraChecks.includes(checkId)) {
    return `reference/checks/${checkId}.md`;
  }

  // Resource checks (resource-postgres, resource-redis, etc.) → generic resource doc
  if (checkId.startsWith("resource-")) {
    return "reference/checks/resource-check.md";
  }

  // Scenario checks (scenario-foo, scenario-bar, etc.) → generic scenario doc
  if (checkId.startsWith("scenario-")) {
    return "reference/checks/scenario-check.md";
  }

  // Fallback to check catalog for unknown checks
  return "reference/check-catalog.md";
}

/**
 * Navigate to the docs page with a specific document selected.
 */
export function navigateToDocs(path: string): void {
  window.location.hash = `docs?path=${encodeURIComponent(path)}`;
}

/**
 * Navigate to the documentation for a specific check ID.
 */
export function navigateToCheckDocs(checkId: string): void {
  navigateToDocs(getDocsPathForCheck(checkId));
}
