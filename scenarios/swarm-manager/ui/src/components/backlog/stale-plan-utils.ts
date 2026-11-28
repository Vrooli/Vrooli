export interface MissingPath {
  glob: string;
  resolved?: string;
  reason?: string;
}

/**
 * extractMissingPaths attempts to read the `details.missingPaths` payload
 * from a `plan_stale` ApiError. Returns an empty array if the shape is
 * missing or invalid.
 */
export function extractMissingPaths(details: unknown): MissingPath[] {
  if (!details || typeof details !== "object") return [];
  const d = details as Record<string, unknown>;
  const raw = d.missingPaths;
  if (!Array.isArray(raw)) return [];
  return raw
    .map((entry): MissingPath | null => {
      if (!entry || typeof entry !== "object") return null;
      const e = entry as Record<string, unknown>;
      const glob =
        typeof e.glob === "string"
          ? e.glob
          : typeof e.Glob === "string"
            ? e.Glob
            : "";
      if (!glob) return null;
      const resolved =
        typeof e.resolved === "string"
          ? e.resolved
          : typeof e.Resolved === "string"
            ? e.Resolved
            : typeof e.resolved_rel === "string"
              ? e.resolved_rel
              : "";
      const reason =
        typeof e.reason === "string"
          ? e.reason
          : typeof e.Reason === "string"
            ? e.Reason
            : "";
      return { glob, resolved, reason };
    })
    .filter((p): p is MissingPath => p !== null);
}
