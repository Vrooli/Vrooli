import { create } from 'zustand';

// ============================================================================
// Types
// ============================================================================

export type ModifierKey = 'ctrl' | 'meta' | 'alt' | 'shift';

export type ShortcutContext =
  | 'global'           // Works everywhere
  | 'dashboard'        // Dashboard view
  | 'project-detail'   // Project detail view
  | 'workflow-builder' // Workflow canvas
  | 'settings'         // Settings page
  | 'modal';           // When a modal is open (usually just Escape)

export interface ShortcutDefinition {
  id: string;
  key: string;
  modifiers?: ModifierKey[];
  description: string;
  category: string;
  contexts: ShortcutContext[];
  // Optional: for chord sequences like "g then h"
  chord?: string;
  // Whether this shortcut is currently enabled
  enabled?: boolean;
}

export interface RegisteredShortcut extends ShortcutDefinition {
  action: () => void;
}

// ============================================================================
// Platform Detection
// ============================================================================

const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.platform);

export const getModifierKey = (): string => (isMac ? '⌘' : 'Ctrl');
export const getAltKey = (): string => (isMac ? '⌥' : 'Alt');
export const getShiftKey = (): string => '⇧';

// ============================================================================
// Default Shortcut Definitions
// ============================================================================

/**
 * All available shortcuts in the application.
 * Note: Actions are registered separately by components.
 * This is the canonical list of what shortcuts exist.
 */
export const DEFAULT_SHORTCUTS: ShortcutDefinition[] = [
  // -------------------------------------------------------------------------
  // Global shortcuts (work everywhere)
  // -------------------------------------------------------------------------
  {
    id: 'show-shortcuts',
    key: '?',
    modifiers: ['shift'],
    description: 'Show keyboard shortcuts',
    category: 'Help',
    contexts: ['global'],
  },
  {
    id: 'close-modal',
    key: 'Escape',
    description: 'Close modal or go back',
    category: 'Navigation',
    contexts: ['global'],
  },
  {
    id: 'global-search',
    key: 'k',
    modifiers: ['meta'],
    description: 'Global search',
    category: 'Navigation',
    contexts: ['global'],
  },
  {
    id: 'open-settings',
    key: ',',
    modifiers: ['meta'],
    description: 'Open settings',
    category: 'Navigation',
    contexts: ['global'],
  },

  // -------------------------------------------------------------------------
  // Dashboard shortcuts
  // -------------------------------------------------------------------------
  {
    id: 'new-project',
    key: 'n',
    modifiers: ['meta', 'shift'],
    description: 'Create new project',
    category: 'Actions',
    contexts: ['dashboard'],
  },
  {
    id: 'go-home',
    key: 'g',
    chord: 'h',
    description: 'Go to dashboard (g then h)',
    category: 'Navigation',
    contexts: ['dashboard', 'project-detail', 'workflow-builder', 'settings'],
  },
  {
    id: 'open-tutorial',
    key: 't',
    modifiers: ['meta', 'shift'],
    description: 'Open guided tour',
    category: 'Help',
    contexts: ['dashboard', 'project-detail', 'workflow-builder'],
  },

  // -------------------------------------------------------------------------
  // Project Detail shortcuts
  // -------------------------------------------------------------------------
  {
    id: 'new-workflow',
    key: 'n',
    modifiers: ['meta', 'shift'],
    description: 'Create new workflow',
    category: 'Actions',
    contexts: ['project-detail'],
  },

  // -------------------------------------------------------------------------
  // Workflow Builder shortcuts
  // -------------------------------------------------------------------------
  {
    id: 'save-workflow',
    key: 's',
    modifiers: ['meta'],
    description: 'Save workflow',
    category: 'Workflow',
    contexts: ['workflow-builder'],
  },
  {
    id: 'execute-workflow',
    key: 'Enter',
    modifiers: ['meta'],
    description: 'Execute workflow',
    category: 'Workflow',
    contexts: ['workflow-builder'],
  },
  {
    id: 'undo',
    key: 'z',
    modifiers: ['meta'],
    description: 'Undo',
    category: 'Editing',
    contexts: ['workflow-builder'],
  },
  {
    id: 'redo',
    key: 'z',
    modifiers: ['meta', 'shift'],
    description: 'Redo',
    category: 'Editing',
    contexts: ['workflow-builder'],
  },
  {
    id: 'delete-selected',
    key: 'Backspace',
    description: 'Delete selected nodes',
    category: 'Editing',
    contexts: ['workflow-builder'],
  },
  {
    id: 'delete-selected-alt',
    key: 'Delete',
    description: 'Delete selected nodes',
    category: 'Editing',
    contexts: ['workflow-builder'],
    enabled: true,
  },
  {
    id: 'select-all',
    key: 'a',
    modifiers: ['meta'],
    description: 'Select all nodes',
    category: 'Editing',
    contexts: ['workflow-builder'],
  },
  {
    id: 'duplicate-selected',
    key: 'd',
    modifiers: ['meta'],
    description: 'Duplicate selected nodes',
    category: 'Editing',
    contexts: ['workflow-builder'],
  },
  {
    id: 'focus-node-search',
    key: 'k',
    modifiers: ['meta'],
    description: 'Focus node palette search',
    category: 'Navigation',
    contexts: ['workflow-builder'],
  },
  {
    id: 'zoom-in',
    key: '=',
    modifiers: ['meta'],
    description: 'Zoom in',
    category: 'Canvas',
    contexts: ['workflow-builder'],
  },
  {
    id: 'zoom-out',
    key: '-',
    modifiers: ['meta'],
    description: 'Zoom out',
    category: 'Canvas',
    contexts: ['workflow-builder'],
  },
  {
    id: 'fit-view',
    key: '0',
    modifiers: ['meta'],
    description: 'Fit view',
    category: 'Canvas',
    contexts: ['workflow-builder'],
  },
  {
    id: 'toggle-lock',
    key: 'l',
    modifiers: ['meta', 'shift'],
    description: 'Toggle canvas lock',
    category: 'Canvas',
    contexts: ['workflow-builder'],
  },
];

// ============================================================================
// Store Interface
// ============================================================================

interface KeyboardShortcutsState {
  // Current active context(s)
  activeContexts: ShortcutContext[];

  // All shortcut definitions
  shortcuts: ShortcutDefinition[];

  // Registered actions (keyed by shortcut id)
  actions: Map<string, () => void>;

  // Chord state for sequences like "g then h"
  pendingChordKey: string | null;
  chordTimeout: ReturnType<typeof setTimeout> | null;

  // Whether shortcuts are globally enabled (disabled during text input, etc.)
  enabled: boolean;

  // Actions
  setActiveContexts: (contexts: ShortcutContext[]) => void;
  addContext: (context: ShortcutContext) => void;
  removeContext: (context: ShortcutContext) => void;
  registerAction: (shortcutId: string, action: () => void) => void;
  unregisterAction: (shortcutId: string) => void;
  setEnabled: (enabled: boolean) => void;

  // Chord handling
  setPendingChord: (key: string | null) => void;
  clearChordTimeout: () => void;

  // Getters
  getActiveShortcuts: () => ShortcutDefinition[];
  getShortcutsByCategory: () => Record<string, ShortcutDefinition[]>;
  executeShortcut: (shortcutId: string) => boolean;
}

// ============================================================================
// Store Implementation
// ============================================================================

export const useKeyboardShortcutsStore = create<KeyboardShortcutsState>((set, get) => ({
  activeContexts: ['global'],
  shortcuts: DEFAULT_SHORTCUTS,
  actions: new Map(),
  pendingChordKey: null,
  chordTimeout: null,
  enabled: true,

  setActiveContexts: (contexts) => {
    // Always include 'global' context
    const uniqueContexts = Array.from(new Set(['global' as ShortcutContext, ...contexts]));
    set({ activeContexts: uniqueContexts });
  },

  addContext: (context) => {
    const { activeContexts } = get();
    if (!activeContexts.includes(context)) {
      set({ activeContexts: [...activeContexts, context] });
    }
  },

  removeContext: (context) => {
    if (context === 'global') return; // Can't remove global
    const { activeContexts } = get();
    set({ activeContexts: activeContexts.filter((c) => c !== context) });
  },

  registerAction: (shortcutId, action) => {
    const { actions } = get();
    const newActions = new Map(actions);
    newActions.set(shortcutId, action);
    set({ actions: newActions });
  },

  unregisterAction: (shortcutId) => {
    const { actions } = get();
    const newActions = new Map(actions);
    newActions.delete(shortcutId);
    set({ actions: newActions });
  },

  setEnabled: (enabled) => set({ enabled }),

  setPendingChord: (key) => {
    const { chordTimeout } = get();
    if (chordTimeout) {
      clearTimeout(chordTimeout);
    }

    if (key) {
      // Set a timeout to clear the chord if not completed
      const timeout = setTimeout(() => {
        set({ pendingChordKey: null, chordTimeout: null });
      }, 1000); // 1 second to complete chord
      set({ pendingChordKey: key, chordTimeout: timeout });
    } else {
      set({ pendingChordKey: null, chordTimeout: null });
    }
  },

  clearChordTimeout: () => {
    const { chordTimeout } = get();
    if (chordTimeout) {
      clearTimeout(chordTimeout);
    }
    set({ pendingChordKey: null, chordTimeout: null });
  },

  getActiveShortcuts: () => {
    const { shortcuts, activeContexts } = get();
    return shortcuts.filter((shortcut) => {
      if (shortcut.enabled === false) return false;
      return shortcut.contexts.some((ctx) => activeContexts.includes(ctx));
    });
  },

  getShortcutsByCategory: () => {
    const activeShortcuts = get().getActiveShortcuts();
    const grouped: Record<string, ShortcutDefinition[]> = {};

    // Filter out duplicate shortcuts (like delete-selected and delete-selected-alt)
    const seen = new Set<string>();
    const deduplicated = activeShortcuts.filter((shortcut) => {
      // Use description as dedup key to avoid showing both Backspace and Delete
      if (seen.has(shortcut.description)) return false;
      seen.add(shortcut.description);
      return true;
    });

    for (const shortcut of deduplicated) {
      const category = shortcut.category;
      if (!grouped[category]) {
        grouped[category] = [];
      }
      grouped[category].push(shortcut);
    }

    return grouped;
  },

  executeShortcut: (shortcutId) => {
    const { actions } = get();
    const action = actions.get(shortcutId);
    if (action) {
      action();
      return true;
    }
    return false;
  },
}));

// ============================================================================
// Formatting Utilities
// ============================================================================

/**
 * Formats a shortcut for display (e.g., "⌘ + Shift + S")
 */
export function formatShortcut(shortcut: ShortcutDefinition): string {
  const parts: string[] = [];

  if (shortcut.modifiers?.includes('ctrl') || shortcut.modifiers?.includes('meta')) {
    parts.push(getModifierKey());
  }
  if (shortcut.modifiers?.includes('alt')) {
    parts.push(getAltKey());
  }
  if (shortcut.modifiers?.includes('shift')) {
    parts.push('Shift');
  }

  // Format the key
  let displayKey = shortcut.key;
  if (shortcut.key === 'Escape') displayKey = 'Esc';
  else if (shortcut.key === 'Enter') displayKey = '↵';
  else if (shortcut.key === 'Backspace') displayKey = '⌫';
  else if (shortcut.key === 'Delete') displayKey = 'Del';
  else if (shortcut.key === 'ArrowUp') displayKey = '↑';
  else if (shortcut.key === 'ArrowDown') displayKey = '↓';
  else if (shortcut.key === 'ArrowLeft') displayKey = '←';
  else if (shortcut.key === 'ArrowRight') displayKey = '→';
  else if (shortcut.key.length === 1) displayKey = shortcut.key.toUpperCase();

  parts.push(displayKey);

  // Handle chord sequences
  if (shortcut.chord) {
    return `${parts[0]} then ${shortcut.chord.toUpperCase()}`;
  }

  return parts.join(' + ');
}

/**
 * Legacy format function for backward compatibility
 */
export function formatShortcutLegacy(
  key: string,
  modifiers?: ModifierKey[]
): string {
  const parts: string[] = [];

  if (modifiers?.includes('ctrl') || modifiers?.includes('meta')) {
    parts.push(getModifierKey());
  }
  if (modifiers?.includes('alt')) {
    parts.push(getAltKey());
  }
  if (modifiers?.includes('shift')) {
    parts.push('Shift');
  }

  let displayKey = key.toUpperCase();
  if (key === 'escape') displayKey = 'Esc';
  if (key === 'enter') displayKey = '↵';
  if (key === '/') displayKey = '/';
  if (key === '?') displayKey = '?';

  parts.push(displayKey);

  return parts.join(' + ');
}

export default useKeyboardShortcutsStore;
