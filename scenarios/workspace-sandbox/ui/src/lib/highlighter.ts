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

// Languages preloaded eagerly with the highlighter. Trimmed to the
// always-on set; everything else loads on first demand via `loadLanguage`.
// Cuts ~50–150 KB off the initial highlighter init. See
// docs/perf/2026-05-03-history-fileviewer-resize.md F6.
const bundledLanguages: BundledLanguage[] = [
  "javascript",
  "typescript",
  "tsx",
  "json",
  "markdown",
  "bash",
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

// Result cache. Keyed on (language, content-hash). FNV-1a hash is collision-
// resistant enough for our purposes (small population of distinct files); a
// real collision merely returns a wrong-but-consistent highlight, which the
// next genuine change evicts. Bounded to 64 entries so a session that opens
// many files doesn't grow unboundedly. See F6.
const HIGHLIGHT_CACHE_LIMIT = 64;
const highlightCache = new Map<string, HighlightedLine[]>();

function fnv1a(str: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i);
    h = (h + ((h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24))) >>> 0;
  }
  return h >>> 0;
}

function cacheKey(lang: string, code: string): string {
  return `${lang}|${code.length}|${fnv1a(code).toString(16)}`;
}

function rememberHighlight(key: string, lines: HighlightedLine[]): HighlightedLine[] {
  if (highlightCache.size >= HIGHLIGHT_CACHE_LIMIT) {
    const oldest = highlightCache.keys().next().value;
    if (oldest !== undefined) highlightCache.delete(oldest);
  }
  highlightCache.set(key, lines);
  return lines;
}

/**
 * Highlight code and return tokens per line
 */
export async function highlightCode(
  code: string,
  language: BundledLanguage | null
): Promise<HighlightedLine[]> {
  // Use plaintext if no language detected or not supported
  const lang = language || "plaintext";

  // Cache lookup before paying for highlighter init / parser load — this
  // is the dominant win when re-rendering the same file (view-mode toggle,
  // re-mount on tab switch, etc.).
  const key = cacheKey(lang, code);
  const cached = highlightCache.get(key);
  if (cached) return cached;

  const highlighter = await getHighlighter();

  // Ensure language is loaded
  if (lang !== "plaintext") {
    const loadedLangs = highlighter.getLoadedLanguages();
    if (!loadedLangs.includes(lang)) {
      try {
        await highlighter.loadLanguage(lang);
      } catch {
        // Fall back to plaintext
        return rememberHighlight(key, highlightAsPlaintext(code));
      }
    }
  }

  try {
    const result = highlighter.codeToTokens(code, {
      lang,
      theme: "github-dark",
    });

    const lines: HighlightedLine[] = result.tokens.map((lineTokens, index) => ({
      lineNumber: index + 1,
      tokens: lineTokens.map((token) => ({
        content: token.content,
        color: token.color,
        fontStyle:
          token.fontStyle === 1 ? "italic" : token.fontStyle === 2 ? "bold" : undefined,
      })),
    }));
    return rememberHighlight(key, lines);
  } catch {
    return rememberHighlight(key, highlightAsPlaintext(code));
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
