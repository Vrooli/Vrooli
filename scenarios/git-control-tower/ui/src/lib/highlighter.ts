import { createHighlighter, type Highlighter, type BundledLanguage } from "shiki";

// Map file extensions to language identifiers
const extensionToLanguage: Record<string, BundledLanguage> = {
  // JavaScript/TypeScript
  js: "javascript",
  jsx: "jsx",
  ts: "typescript",
  tsx: "tsx",
  mjs: "javascript",
  cjs: "javascript",

  // Web
  html: "html",
  htm: "html",
  css: "css",
  scss: "scss",
  sass: "sass",
  less: "less",
  vue: "vue",
  svelte: "svelte",

  // Data formats
  json: "json",
  jsonc: "jsonc",
  json5: "json5",
  yaml: "yaml",
  yml: "yaml",
  toml: "toml",
  xml: "xml",

  // Programming languages
  go: "go",
  rs: "rust",
  py: "python",
  rb: "ruby",
  java: "java",
  kt: "kotlin",
  kts: "kotlin",
  scala: "scala",
  c: "c",
  h: "c",
  cpp: "cpp",
  cc: "cpp",
  cxx: "cpp",
  hpp: "cpp",
  cs: "csharp",
  swift: "swift",
  m: "objective-c",
  php: "php",
  pl: "perl",
  r: "r",
  lua: "lua",
  ex: "elixir",
  exs: "elixir",
  erl: "erlang",
  hs: "haskell",
  clj: "clojure",
  lisp: "lisp",
  elm: "elm",
  fs: "fsharp",
  fsx: "fsharp",
  dart: "dart",
  zig: "zig",
  nim: "nim",
  v: "v",
  jl: "julia",
  groovy: "groovy",
  gradle: "groovy",

  // Shell/scripting
  sh: "bash",
  bash: "bash",
  zsh: "bash",
  fish: "fish",
  ps1: "powershell",
  psm1: "powershell",
  bat: "batch",
  cmd: "batch",

  // Markup/docs
  md: "markdown",
  mdx: "mdx",
  rst: "rst",
  tex: "latex",
  latex: "latex",

  // Config files
  dockerfile: "dockerfile",
  containerfile: "dockerfile",
  makefile: "makefile",
  cmake: "cmake",
  nginx: "nginx",
  ini: "ini",
  conf: "ini",
  cfg: "ini",
  properties: "properties",
  env: "dotenv",

  // Database
  sql: "sql",
  prisma: "prisma",
  graphql: "graphql",
  gql: "graphql",

  // Other
  diff: "diff",
  patch: "diff",
  vim: "viml",
  vimrc: "viml",
  proto: "proto",
  tf: "terraform",
  hcl: "hcl",
  nix: "nix",
  astro: "astro",
  wasm: "wasm",
  asm: "asm",
};

// Common languages to bundle (subset for better loading time)
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
  "graphql",
];

// Singleton highlighter instance
let highlighterInstance: Highlighter | null = null;
let highlighterPromise: Promise<Highlighter> | null = null;

/**
 * Get the language identifier from a file path
 */
export function getLanguageFromPath(path: string): BundledLanguage | null {
  if (!path) return null;

  // Handle special filenames
  const filename = path.split("/").pop()?.toLowerCase() || "";

  // Check for specific filenames first
  if (filename === "dockerfile" || filename.startsWith("dockerfile.")) {
    return "dockerfile";
  }
  if (filename === "makefile" || filename === "gnumakefile") {
    return "makefile";
  }
  if (filename === ".env" || filename.startsWith(".env.")) {
    return "dotenv";
  }
  // gitignore and similar files don't have shiki language support
  if (filename === ".gitignore" || filename === ".gitattributes" || filename === ".editorconfig") {
    return null;
  }

  // Get extension
  const ext = filename.split(".").pop()?.toLowerCase();
  if (!ext) return null;

  return extensionToLanguage[ext] || null;
}

/**
 * Initialize the highlighter (lazy, singleton)
 */
export async function getHighlighter(): Promise<Highlighter> {
  if (highlighterInstance) {
    return highlighterInstance;
  }

  if (highlighterPromise) {
    return highlighterPromise;
  }

  highlighterPromise = createHighlighter({
    themes: ["github-dark"],
    langs: bundledLanguages,
  }).then((instance) => {
    highlighterInstance = instance;
    return instance;
  });

  return highlighterPromise;
}

/**
 * Load an additional language on demand
 */
export async function loadLanguage(lang: BundledLanguage): Promise<void> {
  const highlighter = await getHighlighter();
  const loadedLangs = highlighter.getLoadedLanguages();

  if (!loadedLangs.includes(lang)) {
    try {
      await highlighter.loadLanguage(lang);
    } catch {
      // Language not available, will fall back to plaintext
      console.warn(`Could not load language: ${lang}`);
    }
  }
}

/**
 * Token with style information for a piece of code
 */
export interface HighlightToken {
  content: string;
  color?: string;
  fontStyle?: "italic" | "bold" | "underline";
}

/**
 * A highlighted line with its tokens
 */
export interface HighlightedLine {
  lineNumber: number;
  tokens: HighlightToken[];
}

/**
 * Highlight code and return tokens per line
 */
export async function highlightCode(
  code: string,
  language: BundledLanguage | null
): Promise<HighlightedLine[]> {
  const highlighter = await getHighlighter();

  // Use plaintext if no language detected or not supported
  const lang = language || "plaintext";

  // Ensure language is loaded
  if (lang !== "plaintext") {
    const loadedLangs = highlighter.getLoadedLanguages();
    if (!loadedLangs.includes(lang)) {
      try {
        await highlighter.loadLanguage(lang);
      } catch {
        // Fall back to plaintext
        return highlightAsPlaintext(code);
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
    return highlightAsPlaintext(code);
  }
}

/**
 * Simple plaintext fallback
 */
function highlightAsPlaintext(code: string): HighlightedLine[] {
  const lines = code.split("\n");
  return lines.map((line, index) => ({
    lineNumber: index + 1,
    tokens: [{ content: line }],
  }));
}

/**
 * Highlight a single line (useful for streaming/incremental highlighting)
 */
export async function highlightLine(
  line: string,
  language: BundledLanguage | null
): Promise<HighlightToken[]> {
  const result = await highlightCode(line, language);
  return result[0]?.tokens || [{ content: line }];
}
