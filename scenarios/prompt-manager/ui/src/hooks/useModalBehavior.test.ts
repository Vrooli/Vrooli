/**
 * Tests for useModalBehavior hook.
 */

import { renderHook } from '@testing-library/react'
import { fireEvent } from '@testing-library/react'
import { useModalBehavior, type UseModalBehaviorOptions } from './useModalBehavior'

function createRef(element: HTMLElement | null = null) {
  return { current: element }
}

function setup(overrides: Partial<UseModalBehaviorOptions> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const onClose = vi.fn()

  const defaults: UseModalBehaviorOptions = {
    isOpen: true,
    onClose,
    ref: createRef(container),
    ...overrides,
  }

  const { unmount, rerender } = renderHook(
    (props: UseModalBehaviorOptions) => useModalBehavior(props),
    { initialProps: defaults },
  )

  return { onClose, container, unmount, rerender, defaults }
}

afterEach(() => {
  document.body.innerHTML = ''
  document.body.style.overflow = ''
})

describe('useModalBehavior', () => {
  describe('escape key', () => {
    it('calls onClose when Escape is pressed', () => {
      const { onClose } = setup()
      fireEvent.keyDown(document, { key: 'Escape' })
      expect(onClose).toHaveBeenCalledOnce()
    })

    it('does not close on Escape when disableCloseOnEsc is true', () => {
      const { onClose } = setup({ disableCloseOnEsc: true })
      fireEvent.keyDown(document, { key: 'Escape' })
      expect(onClose).not.toHaveBeenCalled()
    })

    it('does not close on Escape when isLoading is true', () => {
      const { onClose } = setup({ isLoading: true })
      fireEvent.keyDown(document, { key: 'Escape' })
      expect(onClose).not.toHaveBeenCalled()
    })

    it('ignores non-Escape keys', () => {
      const { onClose } = setup()
      fireEvent.keyDown(document, { key: 'Enter' })
      expect(onClose).not.toHaveBeenCalled()
    })
  })

  describe('click outside', () => {
    it('calls onClose when clicking outside the ref element', () => {
      const { onClose } = setup()
      fireEvent.mouseDown(document.body)
      expect(onClose).toHaveBeenCalledOnce()
    })

    it('does not close when clicking inside the ref element', () => {
      const { onClose, container } = setup()
      fireEvent.mouseDown(container)
      expect(onClose).not.toHaveBeenCalled()
    })

    it('does not close on outside click when disableCloseOnOutsideClick is true', () => {
      const { onClose } = setup({ disableCloseOnOutsideClick: true })
      fireEvent.mouseDown(document.body)
      expect(onClose).not.toHaveBeenCalled()
    })

    it('does not close on outside click when isLoading is true', () => {
      const { onClose } = setup({ isLoading: true })
      fireEvent.mouseDown(document.body)
      expect(onClose).not.toHaveBeenCalled()
    })

    it('delays click-outside listener when delayClickOutside is true', () => {
      vi.useFakeTimers()
      const { onClose } = setup({ delayClickOutside: true })

      // Immediate click should not trigger close (listener not attached yet)
      fireEvent.mouseDown(document.body)
      expect(onClose).not.toHaveBeenCalled()

      // After timer fires, click should trigger close
      vi.runAllTimers()
      fireEvent.mouseDown(document.body)
      expect(onClose).toHaveBeenCalledOnce()

      vi.useRealTimers()
    })
  })

  describe('scroll lock', () => {
    it('locks body scroll when preventBodyScroll is true and isOpen', () => {
      setup({ preventBodyScroll: true })
      expect(document.body.style.overflow).toBe('hidden')
    })

    it('does not lock body scroll when preventBodyScroll is false', () => {
      setup({ preventBodyScroll: false })
      expect(document.body.style.overflow).toBe('')
    })

    it('restores body scroll on unmount', () => {
      const { unmount } = setup({ preventBodyScroll: true })
      expect(document.body.style.overflow).toBe('hidden')
      unmount()
      expect(document.body.style.overflow).toBe('')
    })
  })

  describe('isOpen state', () => {
    it('does not attach listeners when isOpen is false', () => {
      const { onClose } = setup({ isOpen: false })
      fireEvent.keyDown(document, { key: 'Escape' })
      fireEvent.mouseDown(document.body)
      expect(onClose).not.toHaveBeenCalled()
    })

    it('cleans up listeners when isOpen transitions to false', () => {
      const { onClose, rerender, defaults } = setup()

      // Escape works while open
      fireEvent.keyDown(document, { key: 'Escape' })
      expect(onClose).toHaveBeenCalledOnce()
      onClose.mockClear()

      // Close the modal
      rerender({ ...defaults, isOpen: false })

      // Escape should no longer trigger close
      fireEvent.keyDown(document, { key: 'Escape' })
      expect(onClose).not.toHaveBeenCalled()
    })
  })
})
