const LANGUAGE_MAP: Record<string, string> = {
  ts: 'typescript',
  tsx: 'tsx',
  js: 'javascript',
  jsx: 'jsx',
  py: 'python',
  sh: 'bash',
  shell: 'bash',
  yml: 'yaml',
  md: 'markdown',
};

const COMMON_LANGUAGES = [
  'typescript',
  'javascript',
  'python',
  'go',
  'json',
  'bash',
  'sql',
  'html',
  'css',
  'yaml',
  'markdown',
  'jsx',
  'tsx',
  'rust',
  'java',
  'c',
  'cpp',
  'ruby',
  'php',
  'swift',
  'kotlin',
] as const;

export function normalizeLanguage(language: string): string {
  const normalized = language.trim().toLowerCase();
  return LANGUAGE_MAP[normalized] ?? normalized;
}

export function detectLanguage(code: string): string {
  const trimmed = code.trim();
  if (!trimmed) return 'text';

  if (/^\s*\{[\s\S]*\}\s*$/.test(trimmed)) return 'json';
  if (/^\s*(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER)\s+/im.test(trimmed)) return 'sql';
  if (/^\s*#!?\/bin\/(bash|sh)/.test(trimmed)) return 'bash';
  if (/^\s*(package\s+main|func\s+\w+\()/m.test(trimmed)) return 'go';
  if (/^\s*(import\s+React|export\s+default|const\s+\w+\s*=\s*\()/m.test(trimmed)) return 'javascript';
  if (/^\s*(def\s+\w+\(|class\s+\w+\(|import\s+\w+)/m.test(trimmed)) return 'python';
  if (/^\s*([\w-]+:|---)/m.test(trimmed)) return 'yaml';

  return 'text';
}

export function isSupportedLanguage(language: string): boolean {
  return COMMON_LANGUAGES.includes(language as (typeof COMMON_LANGUAGES)[number]);
}
