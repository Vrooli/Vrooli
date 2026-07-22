/**
 * @vrooliComponentSource react-component-library:markdown-renderer
 * @vrooliComponentVersion 0.3.2
 * @vrooliComponentAdoption 612450da-7d3d-4888-85a9-e9ecf63254a6
 * @vrooliComponentAppliedAt 2026-07-21T21:01:34Z
 * @vrooliComponentSourceSha256 7b8d128b5a79e8814987887999d9e48df609b75ea7def70dc9356871d98ffe97
 * @vrooliComponentDriftHash 7b8d128b5a79e8814987887999d9e48df609b75ea7def70dc9356871d98ffe97
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
const LANGUAGE_ALIASES: Record<string, string> = {
  js: "javascript",
  ts: "typescript",
  yml: "yaml",
  sh: "bash",
  shell: "bash",
  html: "html",
  text: "text",
};

export function normalizeCodeLanguage(value?: string): string {
  const normalized = (value ?? "text").trim().toLowerCase();
  return (LANGUAGE_ALIASES[normalized] ?? normalized) || "text";
}

export function languageLabel(value?: string): string {
  const language = normalizeCodeLanguage(value);
  return language === "text" ? "Plain text" : language.toUpperCase();
}

/**
 * Preserve prose paths as text nodes. The plugin is intentionally a no-op: it
 * is an opt-in seam for consumers that need path-aware remark processing.
 */
export function remarkProsePaths() {
  return () => undefined;
}