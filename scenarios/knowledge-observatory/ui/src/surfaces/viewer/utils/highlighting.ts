import hljs from "highlight.js";

const languageMap: Record<string, string> = {
  js: "javascript",
  jsx: "javascript",
  ts: "typescript",
  tsx: "typescript",
  json: "json",
  yaml: "yaml",
  yml: "yaml",
  md: "markdown",
  mdx: "markdown",
  go: "go",
  sh: "bash",
  bash: "bash",
  sql: "sql",
  txt: "plaintext",
};

export const escapeHtml = (value: string): string =>
  value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");

export const languageFromPath = (path: string): string | undefined => {
  const parts = path.split("/");
  const name = parts[parts.length - 1] ?? "";
  const ext = name.includes(".") ? name.split(".").pop()?.toLowerCase() : "";
  if (!ext) return undefined;
  return languageMap[ext] ?? undefined;
};

export const highlightCodeToHtml = async (code: string, language?: string): Promise<string> => {
  try {
    if (language && hljs.getLanguage(language)) {
      return hljs.highlight(code, { language }).value;
    }
    return hljs.highlightAuto(code).value;
  } catch {
    return escapeHtml(code);
  }
};

export const highlightCodeBlocks = async (container: HTMLElement) => {
  try {
    const blocks = container.querySelectorAll<HTMLElement>("pre code");
    blocks.forEach((block) => hljs.highlightElement(block));
  } catch {
    // Ignore highlighting failures.
  }
};
