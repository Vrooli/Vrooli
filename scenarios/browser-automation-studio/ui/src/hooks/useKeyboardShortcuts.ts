import { useEffect, useCallback, useRef } from 'react';
import {
  useKeyboardShortcutsStore,
  type ShortcutContext,
  type ShortcutDefinition,
  getModifierKey,
  formatShortcutLegacy,
} from '@stores/keyboardShortcutsStore';

// Re-export for backward compatibility
export { getModifierKey, formatShortcutLegacy };
export type { ShortcutContext, ShortcutDefinition as KeyboardShortcut };

// ============================================================================
// Hook: useKeyboardShortcutHandler
// ============================================================================

/**
 * Main keyboard shortcut handler hook.
 * Should be called ONCE at the app root level.
 * All other components should use useRegisterShortcut to add their actions.
 */
export function useKeyboardShortcutHandler() {
  const store = useKeyboardShortcutsStore();
  const storeRef = useRef(store);
  storeRef.current = store;

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    const {
      enabled,
      shortcuts,
      activeContexts,
      actions,
      pendingChordKey,
      setPendingChord,
      clearChordTimeout,
    } = storeRef.current;

    if (!enabled) return;

    // Don't trigger shortcuts when typing in inputs, textareas, or contenteditable
    // Exception: Escape should always work
    const target = event.target as HTMLElement;
    const isInputField =
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable;

    if (isInputField && event.key !== 'Escape') {
      return;
    }

    const key = event.key;
    const keyLower = key.toLowerCase();
    const hasCtrl = event.ctrlKey;
    const hasMeta = event.metaKey;
    const hasAlt = event.altKey;
    const hasShift = event.shiftKey;
    const hasCtrlOrMeta = hasCtrl || hasMeta;

    // Handle chord sequences (e.g., "g then h")
    if (pendingChordKey) {
      // Look for shortcuts that match the chord
      for (const shortcut of shortcuts) {
        if (shortcut.enabled === false) continue;
        if (!shortcut.chord) continue;
        if (!shortcut.contexts.some((ctx) => activeContexts.includes(ctx))) continue;

        // Check if this shortcut's first key matches pending and second key matches current
        if (
          shortcut.key.toLowerCase() === pendingChordKey.toLowerCase() &&
          shortcut.chord.toLowerCase() === keyLower
        ) {
          const action = actions.get(shortcut.id);
          if (action) {
            event.preventDefault();
            event.stopPropagation();
            clearChordTimeout();
            action();
            return;
          }
        }
      }
      // Clear chord if no match
      clearChordTimeout();
    }

    // Find matching shortcut
    for (const shortcut of shortcuts) {
      if (shortcut.enabled === false) continue;
      if (!shortcut.contexts.some((ctx) => activeContexts.includes(ctx))) continue;

      const modifiers = shortcut.modifiers || [];
      const needsCtrl = modifiers.includes('ctrl');
      const needsMeta = modifiers.includes('meta');
      const needsAlt = modifiers.includes('alt');
      const needsShift = modifiers.includes('shift');
      const needsCtrlOrMeta = needsCtrl || needsMeta;

      // Check key match
      let keyMatches = false;

      // Handle special case: ? key (Shift + /)
      if (shortcut.key === '?' && key === '?' && hasShift) {
        keyMatches = true;
      }
      // Handle single character keys (case-insensitive)
      else if (shortcut.key.length === 1) {
        keyMatches = keyLower === shortcut.key.toLowerCase();
      }
      // Handle special keys (Escape, Enter, Backspace, Delete, etc.)
      else {
        keyMatches = key === shortcut.key || keyLower === shortcut.key.toLowerCase();
      }

      if (!keyMatches) continue;

      // Check modifiers match
      // Note: For ? key, shift is implicit
      const shiftMatches = shortcut.key === '?' ? true : hasShift === needsShift;
      const modifiersMatch =
        hasCtrlOrMeta === needsCtrlOrMeta &&
        hasAlt === needsAlt &&
        shiftMatches;

      if (!modifiersMatch) continue;

      // Check for chord sequence start
      if (shortcut.chord && !modifiers.length) {
        // This is the start of a chord sequence
        event.preventDefault();
        setPendingChord(shortcut.key);
        return;
      }

      // Execute the action
      const action = actions.get(shortcut.id);
      if (action) {
        event.preventDefault();
        event.stopPropagation();
        action();
        return;
      }
    }
  }, []);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown, { capture: true });
    return () => document.removeEventListener('keydown', handleKeyDown, { capture: true });
  }, [handleKeyDown]);
}

// ============================================================================
// Hook: useRegisterShortcut
// ============================================================================

interface RegisterShortcutOptions {
  /** The shortcut ID from the store's shortcut definitions */
  shortcutId: string;
  /** The action to execute when the shortcut is triggered */
  action: () => void;
  /** Whether the shortcut is currently enabled (default: true) */
  enabled?: boolean;
}

/**
 * Register an action for a specific shortcut.
 * The shortcut definition must exist in the store.
 */
export function useRegisterShortcut({
  shortcutId,
  action,
  enabled = true,
}: RegisterShortcutOptions) {
  const registerAction = useKeyboardShortcutsStore((state) => state.registerAction);
  const unregisterAction = useKeyboardShortcutsStore((state) => state.unregisterAction);

  useEffect(() => {
    if (enabled) {
      registerAction(shortcutId, action);
    }
    return () => {
      unregisterAction(shortcutId);
    };
  }, [shortcutId, action, enabled, registerAction, unregisterAction]);
}

// ============================================================================
// Hook: useRegisterShortcuts
// ============================================================================

type ShortcutActionMap = Record<string, () => void>;

/**
 * Register multiple shortcut actions at once.
 * Useful for components that handle many shortcuts.
 */
export function useRegisterShortcuts(
  actionMap: ShortcutActionMap,
  enabled = true
) {
  const registerAction = useKeyboardShortcutsStore((state) => state.registerAction);
  const unregisterAction = useKeyboardShortcutsStore((state) => state.unregisterAction);

  useEffect(() => {
    if (!enabled) return;

    const ids = Object.keys(actionMap);
    for (const id of ids) {
      registerAction(id, actionMap[id]);
    }

    return () => {
      for (const id of ids) {
        unregisterAction(id);
      }
    };
  }, [actionMap, enabled, registerAction, unregisterAction]);
}

// ============================================================================
// Hook: useShortcutContext
// ============================================================================

/**
 * Set the active shortcut context for this component.
 * Automatically adds/removes context when component mounts/unmounts.
 */
export function useShortcutContext(context: ShortcutContext | ShortcutContext[]) {
  const setActiveContexts = useKeyboardShortcutsStore((state) => state.setActiveContexts);

  useEffect(() => {
    const contexts = Array.isArray(context) ? context : [context];
    setActiveContexts(contexts);
  }, [context, setActiveContexts]);
}

// ============================================================================
// Hook: useActiveShortcuts
// ============================================================================

/**
 * Get all shortcuts that are active in the current context.
 * Useful for displaying in the shortcuts modal.
 */
export function useActiveShortcuts() {
  return useKeyboardShortcutsStore((state) => state.getActiveShortcuts());
}

/**
 * Get active shortcuts grouped by category.
 */
export function useShortcutsByCategory() {
  return useKeyboardShortcutsStore((state) => state.getShortcutsByCategory());
}

// ============================================================================
// Legacy Hook: useKeyboardShortcuts (Backward Compatible)
// ============================================================================

export interface LegacyKeyboardShortcut {
  key: string;
  modifiers?: ('ctrl' | 'meta' | 'alt' | 'shift')[];
  description: string;
  category: string;
  action: () => void;
  enabled?: boolean;
}

interface UseLegacyKeyboardShortcutsOptions {
  enabled?: boolean;
}

/**
 * @deprecated Use useRegisterShortcuts instead.
 * Legacy hook for backward compatibility during migration.
 */
export function useKeyboardShortcuts(
  _shortcuts: LegacyKeyboardShortcut[],
  _options: UseLegacyKeyboardShortcutsOptions = {}
) {
  // This is now a no-op - shortcuts should be registered via useRegisterShortcuts
  // and the handler is set up at the app level via useKeyboardShortcutHandler
  console.warn(
    'useKeyboardShortcuts is deprecated. Migrate to useRegisterShortcuts and useKeyboardShortcutHandler.'
  );
}

export default useKeyboardShortcuts;
