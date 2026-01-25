/**
 * Tests for useLinkManagement hook.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useLinkManagement } from './useLinkManagement'

describe('useLinkManagement', () => {
  let mockEditor: {
    getAttributes: (attr: string) => Record<string, string>
    chain: () => {
      focus: () => {
        extendMarkRange: (mark: string) => {
          setLink: (options: { href: string }) => { run: () => void }
        }
        unsetLink: () => { run: () => void }
      }
    }
    isActive: (mark: string) => boolean
  }

  beforeEach(() => {
    mockEditor = {
      getAttributes: vi.fn().mockReturnValue({}),
      chain: vi.fn().mockReturnValue({
        focus: vi.fn().mockReturnValue({
          extendMarkRange: vi.fn().mockReturnValue({
            setLink: vi.fn().mockReturnValue({ run: vi.fn() }),
          }),
          unsetLink: vi.fn().mockReturnValue({ run: vi.fn() }),
        }),
      }),
      isActive: vi.fn().mockReturnValue(false),
    }

    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('initial state', () => {
    it('should have dialog closed by default', () => {
      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      expect(result.current.isDialogOpen).toBe(false)
      expect(result.current.linkUrl).toBe('')
    })

    it('should return hasLink false when no link active', () => {
      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      expect(result.current.hasLink).toBe(false)
    })

    it('should return hasLink true when link is active', () => {
      mockEditor.isActive = vi.fn().mockReturnValue(true)

      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      expect(result.current.hasLink).toBe(true)
    })
  })

  describe('openDialog', () => {
    it('should open dialog and set isDialogOpen to true', () => {
      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      act(() => {
        result.current.openDialog()
      })

      expect(result.current.isDialogOpen).toBe(true)
    })

    it('should pre-populate with existing link URL', () => {
      mockEditor.getAttributes = vi.fn().mockReturnValue({ href: 'https://example.com' })

      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      act(() => {
        result.current.openDialog()
      })

      expect(result.current.linkUrl).toBe('https://example.com')
    })

    it('should do nothing if editor is null', () => {
      const { result } = renderHook(() =>
        useLinkManagement({ editor: null })
      )

      act(() => {
        result.current.openDialog()
      })

      expect(result.current.isDialogOpen).toBe(false)
    })
  })

  describe('closeDialog', () => {
    it('should close dialog and clear URL', () => {
      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      act(() => {
        result.current.openDialog()
        result.current.setLinkUrl('https://test.com')
      })

      expect(result.current.isDialogOpen).toBe(true)
      expect(result.current.linkUrl).toBe('https://test.com')

      act(() => {
        result.current.closeDialog()
      })

      expect(result.current.isDialogOpen).toBe(false)
      expect(result.current.linkUrl).toBe('')
    })
  })

  describe('setLinkUrl', () => {
    it('should update the link URL', () => {
      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      act(() => {
        result.current.setLinkUrl('https://newurl.com')
      })

      expect(result.current.linkUrl).toBe('https://newurl.com')
    })
  })

  describe('saveLink', () => {
    it('should call setLink with normalized URL and close dialog', () => {
      const runMock = vi.fn()
      const setLinkMock = vi.fn().mockReturnValue({ run: runMock })
      const extendMarkRangeMock = vi.fn().mockReturnValue({ setLink: setLinkMock })
      const focusMock = vi.fn().mockReturnValue({ extendMarkRange: extendMarkRangeMock })
      mockEditor.chain = vi.fn().mockReturnValue({ focus: focusMock })

      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      act(() => {
        result.current.setLinkUrl('https://example.com')
      })

      act(() => {
        result.current.saveLink()
      })

      expect(setLinkMock).toHaveBeenCalledWith({ href: 'https://example.com' })
      expect(runMock).toHaveBeenCalled()
      expect(result.current.isDialogOpen).toBe(false)
    })

    it('should add https:// if no protocol specified', () => {
      const runMock = vi.fn()
      const setLinkMock = vi.fn().mockReturnValue({ run: runMock })
      const extendMarkRangeMock = vi.fn().mockReturnValue({ setLink: setLinkMock })
      const focusMock = vi.fn().mockReturnValue({ extendMarkRange: extendMarkRangeMock })
      mockEditor.chain = vi.fn().mockReturnValue({ focus: focusMock })

      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      act(() => {
        result.current.setLinkUrl('example.com')
      })

      act(() => {
        result.current.saveLink()
      })

      expect(setLinkMock).toHaveBeenCalledWith({ href: 'https://example.com' })
    })

    it('should not call setLink if URL is empty', () => {
      const runMock = vi.fn()
      const setLinkMock = vi.fn().mockReturnValue({ run: runMock })
      const extendMarkRangeMock = vi.fn().mockReturnValue({ setLink: setLinkMock })
      const focusMock = vi.fn().mockReturnValue({ extendMarkRange: extendMarkRangeMock })
      mockEditor.chain = vi.fn().mockReturnValue({ focus: focusMock })

      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      act(() => {
        result.current.setLinkUrl('')
      })

      act(() => {
        result.current.saveLink()
      })

      expect(setLinkMock).not.toHaveBeenCalled()
    })
  })

  describe('removeLink', () => {
    it('should call unsetLink', () => {
      const unsetLinkMock = vi.fn().mockReturnValue({ run: vi.fn() })
      mockEditor.chain = vi.fn().mockReturnValue({
        focus: vi.fn().mockReturnValue({
          unsetLink: unsetLinkMock,
        }),
      })

      const { result } = renderHook(() =>
        useLinkManagement({ editor: mockEditor as never })
      )

      act(() => {
        result.current.removeLink()
      })

      expect(unsetLinkMock).toHaveBeenCalled()
    })
  })
})
