/**
 * Persists the user's last-selected copy format to localStorage
 * so it is remembered across sessions.
 */

const STORAGE_KEY = 'prompt-manager:copy-format'
const VALID_FORMATS = new Set(['xml', 'markdown', 'json', 'cli'])

export function getSavedFormat(): 'xml' | 'markdown' | 'json' | 'cli' {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored && VALID_FORMATS.has(stored)) {
      return stored as 'xml' | 'markdown' | 'json' | 'cli'
    }
  } catch {
    // localStorage unavailable (e.g. SSR)
  }
  return 'xml'
}

export function saveFormat(format: string): void {
  try {
    localStorage.setItem(STORAGE_KEY, format)
  } catch {
    // localStorage unavailable
  }
}
