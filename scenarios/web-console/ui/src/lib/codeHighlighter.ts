import { createHighlighter, type BundledLanguage, type Highlighter } from "shiki";

const extensionToLanguage: Record<string, BundledLanguage> = {
  js: "javascript",
  jsx: "jsx",
  ts: "typescript",
  tsx: "tsx",
  mjs: "javascript",
  cjs: "javascript",
  html: "html",
  htm: "html",
  css: "css",
  scss: "scss",
  sass: "sass",
  less: "less",
  vue: "vue",
  svelte: "svelte",
  json: "json",
  jsonc: "jsonc",
  json5: "json5",
  yaml: "yaml",
  yml: "yaml",
  toml: "toml",
  xml: "xml",
  go: "go",
  rs: "rust",
  py: "python",
  rb: "ruby",
  java: "java",
  kt: "kotlin",
  c: "c",
  h: "c",
  cpp: "cpp",
  cc: "cpp",
  hpp: "cpp",
  cs: "csharp",
  swift: "swift",
  php: "php",
  lua: "lua",
  sh: "bash",
  bash: "bash",
  zsh: "bash",
  ps1: "powershell",
  md: "markdown",
  mdx: "mdx",
  sql: "sql",
  graphql: "graphql",
  gql: "graphql",
  diff: "diff",
  patch: "diff",
  proto: "proto",
  tf: "terraform",
  hcl: "hcl",
  dockerfile: "dockerfile",
  makefile: "makefile",
  ini: "ini",
  conf: "ini",
  cfg: "ini",
  env: "dotenv",
};

const bundledLanguages: BundledLanguage[] = [
  "javascript",
  "typescript",
  "jsx",
  "tsx",
  "html",
  "css",
  "json",
  "yaml",
  "markdown",
  "go",
  "python",
  "rust",
  "bash",
  "sql",
  "dockerfile",
  "diff",
  "proto",
];

let highlighterInstance: Highlighter | null = null;
let highlighterPromise: Promise<Highlighter> | null = null;

export function getLanguageFromPath(path: string): BundledLanguage | null {
  if (!path) return null;
  const filename = path.split("/").pop()?.toLowerCase() || "";
  if (filename === "dockerfile" || filename.startsWith("dockerfile.")) return "dockerfile";
  if (filename === "makefile" || filename === "gnumakefile") return "makefile";
  if (filename === ".env" || filename.startsWith(".env.")) return "dotenv";
  const ext = filename.includes(".") ? filename.split(".").pop() : filename;
  if (!ext) return null;
  return extensionToLanguage[ext] || null;
}

export async function getCodeHighlighter(): Promise<Highlighter> {
  if (highlighterInstance) return highlighterInstance;
  if (highlighterPromise) return highlighterPromise;
  highlighterPromise = createHighlighter({
    themes: ["github-dark"],
    langs: bundledLanguages,
  }).then((instance) => {
    highlighterInstance = instance;
    return instance;
  });
  return highlighterPromise;
}

export interface HighlightToken {
  content: string;
  color?: string;
  fontStyle?: "italic" | "bold";
}

export interface HighlightedLine {
  lineNumber: number;
  tokens: HighlightToken[];
}

export async function highlightCode(
  code: string,
  language: BundledLanguage | null,
): Promise<HighlightedLine[]> {
  const highlighter = await getCodeHighlighter();
  let lang: BundledLanguage | "plaintext" = language || "plaintext";

  if (lang !== "plaintext") {
    const loaded = highlighter.getLoadedLanguages();
    if (!loaded.includes(lang)) {
      try {
        await highlighter.loadLanguage(lang);
      } catch {
        lang = "plaintext";
      }
    }
  }

  try {
    const result = highlighter.codeToTokens(code, {
      lang,
      theme: "github-dark",
    });
    return result.tokens.map((lineTokens, index) => ({
      lineNumber: index + 1,
      tokens: lineTokens.map((token) => ({
        content: token.content,
        color: token.color,
        fontStyle: token.fontStyle === 1 ? "italic" : token.fontStyle === 2 ? "bold" : undefined,
      })),
    }));
  } catch {
    return code.split("\n").map((line, index) => ({
      lineNumber: index + 1,
      tokens: [{ content: line }],
    }));
  }
}
