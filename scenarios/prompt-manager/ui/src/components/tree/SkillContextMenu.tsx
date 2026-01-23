/**
 * SkillContextMenu - Context menu for skill items in the tree.
 *
 * Appears on right-click of skill items, providing options like:
 * - Copy skill
 */

import { useEffect, useRef } from 'react'
import { Copy } from 'lucide-react'
import { cn } from '@/lib/utils'

interface SkillContextMenuProps {
  x: number
  y: number
  skillName: string
  onClose: () => void
  onCopySkill: () => void
}

/**
 * Context menu component for skill right-click actions.
 */
export function SkillContextMenu({
  x,
  y,
  skillName,
  onClose,
  onCopySkill,
}: SkillContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)

  // Close on click outside or escape
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    }

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    // Add listeners after a brief delay to avoid immediate close from the right-click event
    const timer = setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside)
      document.addEventListener('keydown', handleEscape)
    }, 0)

    return () => {
      clearTimeout(timer)
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [onClose])

  // Adjust position to stay within viewport
  useEffect(() => {
    if (menuRef.current) {
      const rect = menuRef.current.getBoundingClientRect()
      const viewportWidth = window.innerWidth
      const viewportHeight = window.innerHeight

      let adjustedX = x
      let adjustedY = y

      if (x + rect.width > viewportWidth) {
        adjustedX = viewportWidth - rect.width - 8
      }
      if (y + rect.height > viewportHeight) {
        adjustedY = viewportHeight - rect.height - 8
      }

      menuRef.current.style.left = `${adjustedX}px`
      menuRef.current.style.top = `${adjustedY}px`
    }
  }, [x, y])

  const handleCopySkill = () => {
    onCopySkill()
    onClose()
  }

  return (
    <div
      ref={menuRef}
      className={cn(
        'fixed z-50 min-w-[160px] overflow-hidden rounded-md',
        'bg-popover border border-border shadow-lg',
        'animate-in fade-in-0 zoom-in-95 duration-100'
      )}
      style={{ left: x, top: y }}
    >
      <div className="p-1">
        <button
          type="button"
          onClick={handleCopySkill}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-foreground hover:bg-muted transition-colors',
            'cursor-pointer outline-none'
          )}
        >
          <Copy className="h-4 w-4" />
          <span>Copy "{skillName}"</span>
        </button>
      </div>
    </div>
  )
}
