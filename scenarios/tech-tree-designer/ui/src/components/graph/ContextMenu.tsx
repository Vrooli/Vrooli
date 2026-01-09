import React, { useEffect, useRef } from 'react'
import {
  Plus,
  Sparkles,
  Link2,
  Eye,
  EyeOff,
  Trash2,
  ChevronDown,
  Layers,
  PenSquare
} from 'lucide-react'
import '../../styles/ContextMenu.css'

export type ContextMenuIcon =
  | 'Plus'
  | 'Sparkles'
  | 'Link2'
  | 'Eye'
  | 'EyeOff'
  | 'Trash2'
  | 'ChevronDown'
  | 'Layers'
  | 'PenSquare'

export interface ContextMenuItem {
  id: string
  label: string
  icon: ContextMenuIcon
  onClick: () => void
  disabled?: boolean
  divider?: boolean
  danger?: boolean
}

export interface ContextMenuSection {
  title?: string
  items: ContextMenuItem[]
}

interface ContextMenuProps {
  x: number
  y: number
  sections: ContextMenuSection[]
  onClose: () => void
}

/**
 * Context menu component for graph interactions.
 * Displays at cursor position with keyboard navigation support.
 */
const iconMap: Record<ContextMenuIcon, React.ReactNode> = {
  Plus: <Plus size={16} />,
  Sparkles: <Sparkles size={16} />,
  Link2: <Link2 size={16} />,
  Eye: <Eye size={16} />,
  EyeOff: <EyeOff size={16} />,
  Trash2: <Trash2 size={16} />,
  ChevronDown: <ChevronDown size={16} />,
  Layers: <Layers size={16} />,
  PenSquare: <PenSquare size={16} />
}

const ContextMenu: React.FC<ContextMenuProps> = ({ x, y, sections, onClose }) => {
  const menuRef = useRef<HTMLDivElement>(null)

  // Close on outside click or escape key
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose()
      }
    }

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [onClose])

  // Adjust position to keep menu on screen
  useEffect(() => {
    if (!menuRef.current) return

    const rect = menuRef.current.getBoundingClientRect()
    const viewport = {
      width: window.innerWidth,
      height: window.innerHeight
    }

    let adjustedX = x
    let adjustedY = y

    if (x + rect.width > viewport.width) {
      adjustedX = viewport.width - rect.width - 8
    }

    if (y + rect.height > viewport.height) {
      adjustedY = viewport.height - rect.height - 8
    }

    if (adjustedX !== x || adjustedY !== y) {
      menuRef.current.style.left = `${adjustedX}px`
      menuRef.current.style.top = `${adjustedY}px`
    }
  }, [x, y])

  const handleItemClick = (item: ContextMenuItem) => {
    if (!item.disabled) {
      item.onClick()
      onClose()
    }
  }

  const handleKeyDown = (event: React.KeyboardEvent, item: ContextMenuItem) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      handleItemClick(item)
    }
  }

  return (
    <div
      ref={menuRef}
      className="context-menu"
      style={{ left: x, top: y }}
      role="menu"
      aria-label="Context menu"
    >
      {sections.map((section, sectionIdx) => (
        <div key={sectionIdx} className="context-menu__section">
          {section.title && (
            <div className="context-menu__section-title">{section.title}</div>
          )}
          {section.items.map((item) => (
            <React.Fragment key={item.id}>
              {item.divider && <div className="context-menu__divider" />}
              <button
                type="button"
                className={[
                  'context-menu__item',
                  item.disabled && 'context-menu__item--disabled',
                  item.danger && 'context-menu__item--danger'
                ]
                  .filter(Boolean)
                  .join(' ')}
                onClick={() => handleItemClick(item)}
                onKeyDown={(e) => handleKeyDown(e, item)}
                disabled={item.disabled}
                role="menuitem"
                tabIndex={item.disabled ? -1 : 0}
              >
                <span className="context-menu__item-icon">{iconMap[item.icon]}</span>
                <span className="context-menu__item-label">{item.label}</span>
              </button>
            </React.Fragment>
          ))}
        </div>
      ))}
    </div>
  )
}

export default ContextMenu
