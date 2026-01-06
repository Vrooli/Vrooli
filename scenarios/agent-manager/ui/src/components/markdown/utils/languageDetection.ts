interface LanguagePattern {
  language: string;
  patterns: RegExp[];
  weight: number;
}

const LANGUAGE_PATTERNS: LanguagePattern[] = [
  {
    language: "json",
    patterns: [/^\s*\{[\s\S]*"[^"]+"\s*:/, /^\s*\[[\s\S]*\{/],
    weight: 10,
  },
  {
    language: "typescript",
    patterns: [
      /:\s*(string|number|boolean|any|void|never|unknown)\b/,
      /interface\s+\w+/,
      /type\s+\w+\s*=/,
      /<[A-Z]\w*>/,
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
      /:\s*$\n\s+/m,
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
      /^#!/,
      /\$\{?\w+\}?/,
      /\becho\s+/,
      /\bif\s+\[\[?\s+/,
      /\bfor\s+\w+\s+in\b/,
      /\|\s*\w+/,
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
      /\.\w+\s*\{[^}]*\}/,
      /#\w+\s*\{[^}]*\}/,
      /\b(color|background|margin|padding|font-size)\s*:/,
      /@(media|keyframes|import)\b/,
    ],
    weight: 8,
  },
  {
    language: "yaml",
    patterns: [/^\w+:\s*$/m, /^\s+-\s+\w+/m, /^\s+\w+:\s+.+$/m],
    weight: 6,
  },
  {
    language: "markdown",
    patterns: [/^#{1,6}\s+.+$/m, /\[.+\]\(.+\)/, /\*\*.+\*\*/, /^-\s+.+$/m],
    weight: 4,
  },
];

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

  let bestLanguage = "text";
  let bestScore = 0.5;

  for (const [language, score] of Object.entries(scores)) {
    if (score > bestScore) {
      bestScore = score;
      bestLanguage = language;
    }
  }

  return bestLanguage;
}

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
