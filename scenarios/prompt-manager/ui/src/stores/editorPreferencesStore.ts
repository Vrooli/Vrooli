import { create } from 'zustand'

const EDITOR_PREFERENCES_STORAGE_KEY = 'pm.editorPreferences.v1'

interface EditorPreferencesSnapshot {
  showCodeLineNumbers: boolean
}

function loadEditorPreferences(): EditorPreferencesSnapshot {
  const defaults: EditorPreferencesSnapshot = {
    showCodeLineNumbers: true,
  }

  if (typeof window === 'undefined') return defaults

  try {
    const raw = localStorage.getItem(EDITOR_PREFERENCES_STORAGE_KEY)
    if (!raw) return defaults
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed === null) return defaults
    const record = parsed as Record<string, unknown>

    return {
      showCodeLineNumbers: typeof record.showCodeLineNumbers === 'boolean'
        ? record.showCodeLineNumbers
        : defaults.showCodeLineNumbers,
    }
  } catch {
    return defaults
  }
}

function saveEditorPreferences(settings: EditorPreferencesSnapshot): void {
  if (typeof window === 'undefined') return
  try {
    localStorage.setItem(EDITOR_PREFERENCES_STORAGE_KEY, JSON.stringify(settings))
  } catch {
    // Ignore quota and storage access failures.
  }
}

interface EditorPreferencesStore extends EditorPreferencesSnapshot {
  setShowCodeLineNumbers: (value: boolean) => void
}

export const useEditorPreferencesStore = create<EditorPreferencesStore>((set) => ({
  ...loadEditorPreferences(),
  setShowCodeLineNumbers: (value) => {
    set(() => {
      const next: EditorPreferencesSnapshot = { showCodeLineNumbers: value }
      saveEditorPreferences(next)
      return next
    })
  },
}))

