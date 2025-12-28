/**
 * Language detection utility for code blocks without explicit language hints.
 * Uses pattern matching to identify common programming languages.
 */

interface LanguagePattern {
  language: string;
  patterns: RegExp[];
  weight: number;
}

const LANGUAGE_PATTERNS: LanguagePattern[] = [
  {
    language: "json",
    patterns: [
      /^\s*\{[\s\S]*"[^"]+"\s*:/,      // Object with quoted keys
      /^\s*\[[\s\S]*\{/,               // Array of objects
    ],
    weight: 10,
  },
  {
    language: "typescript",
    patterns: [
      /:\s*(string|number|boolean|any|void|never|unknown)\b/,
      /interface\s+\w+/,
      /type\s+\w+\s*=/,
      /<[A-Z]\w*>/,                     // Generic types
      /:\s*React\./,
      /import\s+.*\s+from\s+['"].*['"]/,
    ],
    weight: 8,
  },
  {
    language: "javascript",
    patterns: [
      /\bconst\s+\w+\s*=/,
      /\blet\s+\w+\s*=/,
      /\bfunction\s+\w+\s*\(/,
      /=>\s*\{/,
      /\bconsole\.(log|error|warn)\(/,
      /\brequire\s*\(/,
      /\bexport\s+(default\s+)?/,
    ],
    weight: 6,
  },
  {
    language: "python",
    patterns: [
      /\bdef\s+\w+\s*\(/,
      /\bclass\s+\w+.*:/,
      /\bimport\s+\w+/,
      /\bfrom\s+\w+\s+import/,
      /\bif\s+__name__\s*==\s*['"]__main__['"]/,
      /\bself\./,
      /:\s*$\n\s+/m,                    // Colon followed by indented block
    ],
    weight: 8,
  },
  {
    language: "go",
    patterns: [
      /\bfunc\s+(\(\w+\s+\*?\w+\)\s+)?\w+\s*\(/,
      /\bpackage\s+\w+/,
      /\btype\s+\w+\s+struct\s*\{/,
      /\berr\s*!=\s*nil/,
      /:=\s*/,
      /\bfmt\.(Print|Sprint|Fprint)/,
    ],
    weight: 8,
  },
  {
    language: "sql",
    patterns: [
      /\bSELECT\s+.*\s+FROM\b/i,
      /\bINSERT\s+INTO\b/i,
      /\bUPDATE\s+.*\s+SET\b/i,
      /\bCREATE\s+(TABLE|INDEX|VIEW)\b/i,
      /\bALTER\s+TABLE\b/i,
      /\bJOIN\s+\w+\s+ON\b/i,
    ],
    weight: 10,
  },
  {
    language: "bash",
    patterns: [
      /^#!/,                            // Shebang
      /\$\{?\w+\}?/,                    // Variable expansion
      /\becho\s+/,
      /\bif\s+\[\[?\s+/,
      /\bfor\s+\w+\s+in\b/,
      /\|\s*\w+/,                       // Piping
    ],
    weight: 6,
  },
  {
    language: "html",
    patterns: [
      /<(!DOCTYPE|html|head|body|div|span|p|a|img)\b/i,
      /<\/\w+>/,
      /class\s*=\s*["']/,
    ],
    weight: 8,
  },
  {
    language: "css",
    patterns: [
      /\.\w+\s*\{[^}]*\}/,             // Class selectors
      /#\w+\s*\{[^}]*\}/,              // ID selectors
      /\b(color|background|margin|padding|font-size)\s*:/,
      /@(media|keyframes|import)\b/,
    ],
    weight: 8,
  },
  {
    language: "yaml",
    patterns: [
      /^\w+:\s*$/m,                     // Key only on line
      /^\s+-\s+\w+/m,                   // List items
      /^\s+\w+:\s+.+$/m,               // Nested key-value
    ],
    weight: 6,
  },
  {
    language: "markdown",
    patterns: [
      /^#{1,6}\s+.+$/m,                 // Headers
      /\[.+\]\(.+\)/,                   // Links
      /\*\*.+\*\*/,                     // Bold
      /^-\s+.+$/m,                      // List items
    ],
    weight: 4,
  },
];

/**
 * Detects the programming language of a code snippet.
 * Returns "text" if no language can be confidently identified.
 */
export function detectLanguage(code: string): string {
  if (!code || code.trim().length === 0) {
    return "text";
  }

  const scores: Record<string, number> = {};

  for (const { language, patterns, weight } of LANGUAGE_PATTERNS) {
    let matchCount = 0;
    for (const pattern of patterns) {
      if (pattern.test(code)) {
        matchCount++;
      }
    }
    if (matchCount > 0) {
      scores[language] = (matchCount / patterns.length) * weight;
    }
  }

  // Find the language with the highest score
  let bestLanguage = "text";
  let bestScore = 0.5; // Minimum threshold to avoid false positives

  for (const [language, score] of Object.entries(scores)) {
    if (score > bestScore) {
      bestScore = score;
      bestLanguage = language;
    }
  }

  return bestLanguage;
}

/**
 * Maps common language aliases to shiki-supported language names.
 */
export function normalizeLanguage(lang: string): string {
  const aliases: Record<string, string> = {
    js: "javascript",
    ts: "typescript",
    jsx: "jsx",
    tsx: "tsx",
    py: "python",
    rb: "ruby",
    sh: "bash",
    shell: "bash",
    zsh: "bash",
    yml: "yaml",
    md: "markdown",
    dockerfile: "docker",
    tf: "terraform",
    rs: "rust",
    kt: "kotlin",
    cs: "csharp",
    "c++": "cpp",
    "c#": "csharp",
  };

  const normalized = lang.toLowerCase().trim();
  return aliases[normalized] || normalized;
}
