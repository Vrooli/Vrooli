/**
 * Tests for useKeyboardShortcuts hook.
 *
 * Tests cover:
 * - Keyboard shortcut handler registration
 * - Modifier key detection (Cmd/Ctrl)
 * - Input field filtering
 * - getShortcutDisplay function
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useKeyboardShortcuts, getShortcutDisplay } from './useKeyboardShortcuts'

// Type for mock calls
type EventListenerCall = [string, EventListenerOrEventListenerObject, ...unknown[]]

describe('useKeyboardShortcuts', () => {
  const addCalls: EventListenerCall[] = []
  const removeCalls: EventListenerCall[] = []

  beforeEach(() => {
    addCalls.length = 0
    removeCalls.length = 0
    vi.spyOn(window, 'addEventListener').mockImplementation((type: string, listener: EventListenerOrEventListenerObject, ...rest: unknown[]) => {
      addCalls.push([type, listener, ...rest])
    })
    vi.spyOn(window, 'removeEventListener').mockImplementation((type: string, listener: EventListenerOrEventListenerObject, ...rest: unknown[]) => {
      removeCalls.push([type, listener, ...rest])
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('event listener lifecycle', () => {
    it('should add keydown event listener on mount', () => {
      const handlers = { onSave: vi.fn() }
      renderHook(() => useKeyboardShortcuts(handlers))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      expect(keydownCall).toBeDefined()
      expect(typeof keydownCall?.[1]).toBe('function')
    })

    it('should remove keydown event listener on unmount', () => {
      const handlers = { onSave: vi.fn() }
      const { unmount } = renderHook(() => useKeyboardShortcuts(handlers))

      unmount()

      const keydownCall = removeCalls.find((call) => call[0] === 'keydown')
      expect(keydownCall).toBeDefined()
    })
  })

  describe('handler calls', () => {
    it('should call onEscape when Escape is pressed', () => {
      const onEscape = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onEscape }))

      // Get the registered keydown handler
      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      // Simulate Escape key press
      const event = new KeyboardEvent('keydown', { key: 'Escape' })
      keydownHandler(event)

      expect(onEscape).toHaveBeenCalledTimes(1)
    })

    it('should call onSave when Ctrl+S is pressed (non-Mac)', () => {
      const onSave = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onSave }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      // Simulate Ctrl+S (non-Mac)
      const event = new KeyboardEvent('keydown', {
        key: 's',
        ctrlKey: true,
        shiftKey: false,
      })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      expect(onSave).toHaveBeenCalledTimes(1)
      expect(event.preventDefault).toHaveBeenCalled()
    })

    it('should call onSaveAll when Ctrl+Shift+S is pressed', () => {
      const onSaveAll = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onSaveAll }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      // Simulate Ctrl+Shift+S
      const event = new KeyboardEvent('keydown', {
        key: 'S',
        ctrlKey: true,
        shiftKey: true,
      })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      expect(onSaveAll).toHaveBeenCalledTimes(1)
    })

    it('should call onNew when Ctrl+N is pressed', () => {
      const onNew = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onNew }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      const event = new KeyboardEvent('keydown', {
        key: 'n',
        ctrlKey: true,
        shiftKey: false,
      })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      expect(onNew).toHaveBeenCalledTimes(1)
    })

    it('should call onFocusSearch when Ctrl+K is pressed', () => {
      const onFocusSearch = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onFocusSearch }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
        shiftKey: false,
      })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      expect(onFocusSearch).toHaveBeenCalledTimes(1)
    })

    it('should call onOpenSettings when comma is pressed', () => {
      const onOpenSettings = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onOpenSettings }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      const event = new KeyboardEvent('keydown', {
        key: ',',
        ctrlKey: false,
        shiftKey: false,
        altKey: false,
      })
      Object.defineProperty(event, 'target', {
        value: document.createElement('div'),
        writable: true,
      })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      expect(onOpenSettings).toHaveBeenCalledTimes(1)
    })
  })

  describe('input field handling', () => {
    it('should skip shortcuts (except Escape) when target is input element', () => {
      const onSave = vi.fn()
      const onEscape = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onSave, onEscape }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      // Create an input element as target
      const inputElement = document.createElement('input')

      // Test Ctrl+S in input - should be skipped
      const saveEvent = new KeyboardEvent('keydown', {
        key: 's',
        ctrlKey: true,
      })
      Object.defineProperty(saveEvent, 'target', { value: inputElement })
      Object.defineProperty(saveEvent, 'preventDefault', { value: vi.fn() })
      keydownHandler(saveEvent)

      expect(onSave).not.toHaveBeenCalled()

      // Test Escape in input - should still work
      const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape' })
      Object.defineProperty(escapeEvent, 'target', { value: inputElement })
      keydownHandler(escapeEvent)

      expect(onEscape).toHaveBeenCalledTimes(1)
    })

    it('should skip shortcuts when target is textarea element', () => {
      const onSave = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onSave }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      const textareaElement = document.createElement('textarea')
      const event = new KeyboardEvent('keydown', {
        key: 's',
        ctrlKey: true,
      })
      Object.defineProperty(event, 'target', { value: textareaElement })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      expect(onSave).not.toHaveBeenCalled()
    })

    it('should skip shortcuts when target is contenteditable element', () => {
      const onOpenSettings = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onOpenSettings }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      const editableElement = document.createElement('div')
      editableElement.setAttribute('contenteditable', 'true')

      const event = new KeyboardEvent('keydown', {
        key: ',',
        ctrlKey: false,
        shiftKey: false,
        altKey: false,
      })
      Object.defineProperty(event, 'target', { value: editableElement })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      expect(onOpenSettings).not.toHaveBeenCalled()
    })

    it('should skip shortcuts when target is Monaco editor surface', () => {
      const onOpenSettings = vi.fn()
      renderHook(() => useKeyboardShortcuts({ onOpenSettings }))

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      const monacoElement = document.createElement('div')
      monacoElement.classList.add('monaco-editor')
      const child = document.createElement('div')
      monacoElement.appendChild(child)

      const event = new KeyboardEvent('keydown', {
        key: ',',
        ctrlKey: false,
        shiftKey: false,
        altKey: false,
      })
      Object.defineProperty(event, 'target', { value: child })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      expect(onOpenSettings).not.toHaveBeenCalled()
    })
  })

  describe('handler updates', () => {
    it('should use updated handlers when they change', () => {
      const onSave1 = vi.fn()
      const onSave2 = vi.fn()

      const { rerender } = renderHook(
        ({ onSave }: { onSave: () => void }) => useKeyboardShortcuts({ onSave }),
        { initialProps: { onSave: onSave1 } }
      )

      // Update to new handler
      rerender({ onSave: onSave2 })

      const keydownCall = addCalls.find((call) => call[0] === 'keydown')
      const keydownHandler = keydownCall?.[1] as ((e: KeyboardEvent) => void) | undefined
      if (!keydownHandler || typeof keydownHandler !== 'function') throw new Error('No keydown handler found')

      const event = new KeyboardEvent('keydown', {
        key: 's',
        ctrlKey: true,
      })
      Object.defineProperty(event, 'preventDefault', { value: vi.fn() })
      keydownHandler(event)

      // Should call the updated handler
      expect(onSave1).not.toHaveBeenCalled()
      expect(onSave2).toHaveBeenCalledTimes(1)
    })
  })
})

describe('getShortcutDisplay', () => {
  it('should return correct display for save shortcut', () => {
    const display = getShortcutDisplay('save')
    // On non-Mac (in test env), should show Ctrl+
    expect(display).toMatch(/S$/)
  })

  it('should return correct display for saveAll shortcut', () => {
    const display = getShortcutDisplay('saveAll')
    expect(display).toMatch(/Shift\+S$/)
  })

  it('should return correct display for new shortcut', () => {
    const display = getShortcutDisplay('new')
    expect(display).toMatch(/N$/)
  })

  it('should return correct display for search shortcut', () => {
    const display = getShortcutDisplay('search')
    expect(display).toMatch(/K$/)
  })

  it('should return correct display for escape shortcut', () => {
    const display = getShortcutDisplay('escape')
    expect(display).toBe('Esc')
  })

  it('should return correct display for settings shortcut', () => {
    const display = getShortcutDisplay('settings')
    expect(display).toBe(',')
  })

  it('should return input as-is for unknown shortcuts', () => {
    const display = getShortcutDisplay('unknown')
    expect(display).toBe('unknown')
  })
})
