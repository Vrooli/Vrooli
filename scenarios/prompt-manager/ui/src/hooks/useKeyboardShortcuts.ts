/**
 * useKeyboardShortcuts - Hook for managing keyboard shortcuts.
 *
 * Provides fixed keyboard shortcuts for the prompt manager:
 * - Ctrl/Cmd+S: Save current prompt
 * - Ctrl/Cmd+Shift+S: Save all changes
 * - Ctrl/Cmd+N: New prompt
 * - Ctrl/Cmd+K: Focus search
 * - Escape: Discard changes / close dialogs
 * - , (comma): Open settings
 *
 * All shortcuts use Cmd on macOS and Ctrl on other platforms.
 */

import { useEffect, useCallback, useRef } from 'react'

// Type for the modern NavigatorUAData API
interface NavigatorUAData {
  platform: string
}

// Extend Navigator type with userAgentData
declare global {
  interface Navigator {
    userAgentData?: NavigatorUAData
  }
}

export interface KeyboardShortcutHandlers {
  /** Save the current prompt */
  onSave?: () => void
  /** Save all unsaved changes */
  onSaveAll?: () => void
  /** Create a new prompt */
  onNew?: () => void
  /** Focus the search input */
  onFocusSearch?: () => void
  /** Discard changes or close dialogs */
  onEscape?: () => void
  /** Open settings modal */
  onOpenSettings?: () => void
}

/**
 * Check if the event target is an input element where we shouldn't intercept shortcuts.
 * Allows shortcuts in Monaco editor and certain inputs.
 */
function isInputElement(target: EventTarget | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false

  const tagName = target.tagName.toLowerCase()
  const isInput = tagName === 'input' || tagName === 'textarea'

  // Allow shortcuts in Monaco editor (which uses contenteditable)
  const isMonaco = target.closest('.monaco-editor') !== null

  return isInput && !isMonaco
}

/**
 * Detect if the current platform is macOS or iOS.
 * Uses userAgentData when available, falls back to userAgent.
 */
function isMacPlatform(): boolean {
  if (typeof navigator === 'undefined') return false
  // Modern API (Chrome/Edge 90+)
  const platform = navigator.userAgentData?.platform
  if (platform) {
    return platform === 'macOS' || platform === 'iOS'
  }
  // Fallback to userAgent
  return /Mac|iPod|iPhone|iPad/.test(navigator.userAgent)
}

/**
 * Check if the modifier key is pressed (Cmd on Mac, Ctrl elsewhere).
 */
function hasModifier(e: KeyboardEvent): boolean {
  return isMacPlatform() ? e.metaKey : e.ctrlKey
}

export function useKeyboardShortcuts(handlers: KeyboardShortcutHandlers): void {
  const handlersRef = useRef(handlers)

  // Keep handlers ref up to date
  useEffect(() => {
    handlersRef.current = handlers
  }, [handlers])

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    const h = handlersRef.current

    // Skip if target is a text input (except for Escape)
    const isTextInput = isInputElement(e.target)

    // Escape - always allowed
    if (e.key === 'Escape') {
      if (h.onEscape) {
        h.onEscape()
        // Don't prevent default for Escape - let it close native dialogs too
      }
      return
    }

    // Skip other shortcuts in text inputs
    if (isTextInput) return

    // Ctrl/Cmd+S - Save current
    if (hasModifier(e) && e.key === 's' && !e.shiftKey) {
      e.preventDefault()
      h.onSave?.()
      return
    }

    // Ctrl/Cmd+Shift+S - Save all
    if (hasModifier(e) && e.key === 'S' && e.shiftKey) {
      e.preventDefault()
      h.onSaveAll?.()
      return
    }

    // Ctrl/Cmd+N - New prompt
    if (hasModifier(e) && e.key === 'n' && !e.shiftKey) {
      e.preventDefault()
      h.onNew?.()
      return
    }

    // Ctrl/Cmd+K - Focus search
    if (hasModifier(e) && e.key === 'k' && !e.shiftKey) {
      e.preventDefault()
      h.onFocusSearch?.()
      return
    }

    // Comma - Open settings (only when not in text input)
    if (e.key === ',' && !hasModifier(e) && !e.shiftKey && !e.altKey) {
      // Double-check we're not in an input
      if (!isInputElement(e.target)) {
        e.preventDefault()
        h.onOpenSettings?.()
      }
      return
    }
  }, [])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}

/**
 * Get the display string for a keyboard shortcut.
 * Shows Cmd on Mac, Ctrl elsewhere.
 */
export function getShortcutDisplay(shortcut: string): string {
  const modifier = isMacPlatform() ? '⌘' : 'Ctrl+'

  switch (shortcut) {
    case 'save':
      return `${modifier}S`
    case 'saveAll':
      return `${modifier}Shift+S`
    case 'new':
      return `${modifier}N`
    case 'search':
      return `${modifier}K`
    case 'escape':
      return 'Esc'
    case 'settings':
      return ','
    default:
      return shortcut
  }
}
