// Keep Markdown as per-document dynamic imports. The route loads only the
// selected file instead of embedding the entire documentation tree into one
// 1 MB+ viewer chunk.

const SCENARIO_ROOT_PREFIX = "../../../../";

const rawRoot = import.meta.glob<string>("../../../../*.md", {
  query: "?raw",
  import: "default",
});

const rawTree = import.meta.glob<string>("../../../../docs/**/*.md", {
  query: "?raw",
  import: "default",
});

function normaliseKey(key: string): string | null {
  if (!key.startsWith(SCENARIO_ROOT_PREFIX)) return null;
  return key.slice(SCENARIO_ROOT_PREFIX.length);
}

const docs: Record<string, () => Promise<string>> = {};
for (const source of [rawRoot, rawTree]) {
  for (const [key, load] of Object.entries(source)) {
    const normalised = normaliseKey(key);
    if (!normalised) continue;
    docs[normalised] = load;
  }
}

export async function getDocContent(path: string): Promise<string | null> {
  const load = docs[path];
  return load ? load() : null;
}

export function listDocPaths(): string[] {
  return Object.keys(docs);
}
