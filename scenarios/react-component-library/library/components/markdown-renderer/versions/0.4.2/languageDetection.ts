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
