/**
 * Scenario Utilities
 *
 * Extract scenario names from acceptance glob patterns.
 * Mirrors pathutil.ScenariosFromGlobs on the backend.
 */

/**
 * Extract unique scenario names from acceptance glob patterns.
 * Globs starting with "scenarios/<name>/..." yield the scenario name;
 * all other patterns are skipped.
 */
export function scenariosFromGlobs(globs: string[] | undefined): string[] {
  if (!globs) return [];
  const seen = new Set<string>();
  const result: string[] = [];
  for (const g of globs) {
    if (!g.startsWith("scenarios/")) continue;
    const rest = g.slice("scenarios/".length);
    const slash = rest.indexOf("/");
    const name = slash === -1 ? rest : rest.slice(0, slash);
    if (name && !seen.has(name)) {
      seen.add(name);
      result.push(name);
    }
  }
  return result;
}
