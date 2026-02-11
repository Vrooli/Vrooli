import type { GroupingRule } from "../components/FileList";

/**
 * Normalize a prefix string to ensure it ends with "/".
 * Extracted from FileList for reuse.
 */
export function normalizePrefix(prefix: string): string {
  const trimmed = prefix.trim();
  if (!trimmed) return "";
  if (trimmed === "/") return "/";
  return trimmed.endsWith("/") ? trimmed : `${trimmed}/`;
}

/**
 * Resolve which group a file path belongs to based on grouping rules.
 * Returns null if no rule matches.
 */
export function resolveGroupForFile(
  path: string,
  rules: GroupingRule[]
): { groupDir: string; groupLabel: string } | null {
  for (const rule of rules) {
    const prefixes = rule.prefixes ?? (rule.prefix ? [rule.prefix] : []);
    for (const rawPrefix of prefixes) {
      const prefix = normalizePrefix(rawPrefix);
      if (!prefix || !path.startsWith(prefix)) continue;

      if (rule.mode === "segment") {
        // Extract the next path segment after the prefix
        const rest = path.slice(prefix.length);
        const slashIdx = rest.indexOf("/");
        const segment = slashIdx >= 0 ? rest.slice(0, slashIdx) : rest;
        if (!segment) continue;
        return {
          groupDir: `${prefix}${segment}/`,
          groupLabel: segment,
        };
      }

      // prefix mode
      return {
        groupDir: prefix,
        groupLabel: rule.label,
      };
    }
  }
  return null;
}
