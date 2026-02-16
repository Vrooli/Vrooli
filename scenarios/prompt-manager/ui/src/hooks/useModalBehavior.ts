/**
 * useModalBehavior - Shared hook for modal/popover/dialog dismiss behaviors.
 *
 * Extracts the identical escape-key, click-outside, and scroll-lock patterns
 * found across 16+ components into one composable hook.
 */

import { useEffect, useCallback, type RefObject } from 'react'

export interface UseModalBehaviorOptions {
  /** Whether the modal/popover is currently open */
  isOpen: boolean
  /** Callback to close the modal/popover */
  onClose: () => void
  /** Ref to the modal/popover container element */
  ref: RefObject<HTMLElement | null>
  /** Disable closing on Escape key press */
  disableCloseOnEsc?: boolean
  /** Disable closing on click outside the ref element */
  disableCloseOnOutsideClick?: boolean
  /** Lock body scroll while open (for modals with backdrops) */
  preventBodyScroll?: boolean
  /** Delay click-outside listener attachment (for context menus opened via right-click) */
  delayClickOutside?: boolean
  /** Block close when a loading/async operation is in progress */
  isLoading?: boolean
}

/**
 * Hook that manages common modal dismiss behaviors:
 * - Escape key to close
 * - Click outside to close
 * - Body scroll lock
 *
 * @example
 * ```tsx
 * const ref = useRef<HTMLDivElement>(null)
 * useModalBehavior({ isOpen, onClose, ref, preventBodyScroll: true })
 * ```
 */
export function useModalBehavior({
  isOpen,
  onClose,
  ref,
  disableCloseOnEsc = false,
  disableCloseOnOutsideClick = false,
  preventBodyScroll = false,
  delayClickOutside = false,
  isLoading = false,
}: UseModalBehaviorOptions): void {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !disableCloseOnEsc && !isLoading) {
        onClose()
      }
    },
    [onClose, disableCloseOnEsc, isLoading],
  )

  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (
        !disableCloseOnOutsideClick &&
        !isLoading &&
        ref.current &&
        !ref.current.contains(event.target as Node)
      ) {
        onClose()
      }
    },
    [onClose, ref, disableCloseOnOutsideClick, isLoading],
  )

  useEffect(() => {
    if (!isOpen) return

    // Escape listener is always attached immediately
    document.addEventListener('keydown', handleKeyDown)

    // Click-outside listener may be delayed for context menus
    let clickTimer: ReturnType<typeof setTimeout> | undefined
    if (!disableCloseOnOutsideClick) {
      if (delayClickOutside) {
        clickTimer = setTimeout(() => {
          document.addEventListener('mousedown', handleClickOutside)
        }, 0)
      } else {
        document.addEventListener('mousedown', handleClickOutside)
      }
    }

    // Scroll lock
    if (preventBodyScroll) {
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('mousedown', handleClickOutside)
      if (clickTimer !== undefined) {
        clearTimeout(clickTimer)
      }
      if (preventBodyScroll) {
        document.body.style.overflow = ''
      }
    }
  }, [isOpen, handleKeyDown, handleClickOutside, disableCloseOnOutsideClick, delayClickOutside, preventBodyScroll])
}
