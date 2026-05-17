// Bundle markdown content at build time via Vite's `?raw` glob import.
// Globs are scoped to the scenario root (PRD.md, README.md, ...) and the
// `docs/` tree. Keys are normalised to scenario-root-relative paths matching
// the DOCS registry entries (e.g. "PRD.md", "docs/concepts/ARCHITECTURE.md").

const SCENARIO_ROOT_PREFIX = "../../../../";

const rawRoot = import.meta.glob("../../../../*.md", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

const rawTree = import.meta.glob("../../../../docs/**/*.md", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

function normaliseKey(key: string): string | null {
  if (!key.startsWith(SCENARIO_ROOT_PREFIX)) return null;
  return key.slice(SCENARIO_ROOT_PREFIX.length);
}

const docs: Record<string, string> = {};
for (const source of [rawRoot, rawTree]) {
  for (const [key, value] of Object.entries(source)) {
    const normalised = normaliseKey(key);
    if (!normalised) continue;
    docs[normalised] = value;
  }
}

export function getDocContent(path: string): string | null {
  return docs[path] ?? null;
}

export function listDocPaths(): string[] {
  return Object.keys(docs);
}
