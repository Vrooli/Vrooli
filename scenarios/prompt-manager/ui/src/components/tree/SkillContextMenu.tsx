/**
 * SkillContextMenu - Context menu for skill items in the tree.
 *
 * Appears on right-click of skill items, providing options like:
 * - Copy skill
 * - Move to folder
 * - Change storage location
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { Copy, Folder, FolderPlus, HardDrive, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FolderType } from '@/types'

interface SkillContextMenuProps {
  x: number
  y: number
  skillId: string
  skillName: string
  currentModes: string[]
  currentFolder: FolderType
  availableModePaths: string[][]
  onClose: () => void
  onCopySkill: () => void
  onMoveToFolder: (path: string[]) => void
  onChangeStorage: (folder: FolderType) => void
  onCreateNewFolder: () => void
}

/**
 * Context menu component for skill right-click actions.
 */
export function SkillContextMenu({
  x,
  y,
  skillId: _skillId,
  skillName,
  currentModes,
  currentFolder,
  availableModePaths,
  onClose,
  onCopySkill,
  onMoveToFolder,
  onChangeStorage,
  onCreateNewFolder,
}: SkillContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)
  const [activeSubmenu, setActiveSubmenu] = useState<'move' | 'storage' | null>(null)

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

  const handleCopySkill = useCallback(() => {
    onCopySkill()
    onClose()
  }, [onCopySkill, onClose])

  const handleMoveToFolder = useCallback((path: string[]) => {
    onMoveToFolder(path)
    onClose()
  }, [onMoveToFolder, onClose])

  const handleChangeStorage = useCallback((folder: FolderType) => {
    onChangeStorage(folder)
    onClose()
  }, [onChangeStorage, onClose])

  const handleCreateNewFolder = useCallback(() => {
    onCreateNewFolder()
    onClose()
  }, [onCreateNewFolder, onClose])

  // Get unique mode paths for the submenu
  const uniqueModePaths = Array.from(
    new Set(availableModePaths.map((p) => p.join('/')))
  ).map((s) => s.split('/').filter(Boolean))

  const currentPathString = currentModes.filter(Boolean).join('/')

  return (
    <div
      ref={menuRef}
      className={cn(
        'fixed z-50 min-w-[180px] overflow-visible rounded-md',
        'bg-popover border border-border shadow-lg',
        'animate-in fade-in-0 zoom-in-95 duration-100'
      )}
      style={{ left: x, top: y }}
    >
      <div className="p-1">
        {/* Copy skill */}
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

        {/* Divider */}
        <div className="my-1 border-t border-border" />

        {/* Move to submenu */}
        <div
          className="relative"
          onMouseEnter={() => setActiveSubmenu('move')}
          onMouseLeave={() => setActiveSubmenu(null)}
        >
          <div
            className={cn(
              'flex items-center justify-between gap-2 px-2 py-1.5 text-sm rounded-sm',
              'text-foreground hover:bg-muted transition-colors cursor-pointer'
            )}
          >
            <div className="flex items-center gap-2">
              <Folder className="h-4 w-4" />
              <span>Move to...</span>
            </div>
            <ChevronRight className="h-4 w-4" />
          </div>

          {/* Move submenu */}
          {activeSubmenu === 'move' && (
            <div
              className={cn(
                'absolute left-full top-0 ml-1 min-w-[160px] max-h-64 overflow-y-auto',
                'bg-popover border border-border rounded-md shadow-lg',
                'animate-in fade-in-0 zoom-in-95 duration-100'
              )}
            >
              <div className="p-1">
                {/* Root option */}
                <button
                  type="button"
                  onClick={() => handleMoveToFolder([])}
                  className={cn(
                    'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
                    'text-foreground hover:bg-muted transition-colors text-left',
                    currentPathString === '' && 'bg-primary/20 text-primary'
                  )}
                >
                  <Folder className="h-4 w-4" />
                  <span className="italic">(Root)</span>
                </button>

                {/* Existing paths */}
                {uniqueModePaths.map((path) => {
                  const pathStr = path.join('/')
                  const isSelected = pathStr === currentPathString

                  return (
                    <button
                      key={pathStr}
                      type="button"
                      onClick={() => handleMoveToFolder(path)}
                      className={cn(
                        'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
                        'text-foreground hover:bg-muted transition-colors text-left',
                        isSelected && 'bg-primary/20 text-primary'
                      )}
                    >
                      <Folder className="h-4 w-4" />
                      <span className="truncate">{path.join(' / ')}</span>
                    </button>
                  )
                })}

                {/* Divider */}
                <div className="my-1 border-t border-border" />

                {/* New folder option */}
                <button
                  type="button"
                  onClick={handleCreateNewFolder}
                  className={cn(
                    'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
                    'text-foreground hover:bg-muted transition-colors text-left'
                  )}
                >
                  <FolderPlus className="h-4 w-4" />
                  <span>New folder...</span>
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Storage submenu */}
        <div
          className="relative"
          onMouseEnter={() => setActiveSubmenu('storage')}
          onMouseLeave={() => setActiveSubmenu(null)}
        >
          <div
            className={cn(
              'flex items-center justify-between gap-2 px-2 py-1.5 text-sm rounded-sm',
              'text-foreground hover:bg-muted transition-colors cursor-pointer'
            )}
          >
            <div className="flex items-center gap-2">
              <HardDrive className="h-4 w-4" />
              <span>Storage: {currentFolder === 'core' ? 'Core' : 'Local'}</span>
            </div>
            <ChevronRight className="h-4 w-4" />
          </div>

          {/* Storage submenu */}
          {activeSubmenu === 'storage' && (
            <div
              className={cn(
                'absolute left-full top-0 ml-1 min-w-[140px]',
                'bg-popover border border-border rounded-md shadow-lg',
                'animate-in fade-in-0 zoom-in-95 duration-100'
              )}
            >
              <div className="p-1">
                <button
                  type="button"
                  onClick={() => handleChangeStorage('local')}
                  className={cn(
                    'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
                    'text-foreground hover:bg-muted transition-colors text-left',
                    currentFolder === 'local' && 'bg-primary/20 text-primary'
                  )}
                >
                  <HardDrive className="h-4 w-4" />
                  <div>
                    <div>Local</div>
                    <div className="text-[10px] text-muted-foreground">Personal, gitignored</div>
                  </div>
                </button>
                <button
                  type="button"
                  onClick={() => handleChangeStorage('core')}
                  className={cn(
                    'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
                    'text-foreground hover:bg-muted transition-colors text-left',
                    currentFolder === 'core' && 'bg-primary/20 text-primary'
                  )}
                >
                  <Folder className="h-4 w-4" />
                  <div>
                    <div>Core</div>
                    <div className="text-[10px] text-muted-foreground">Shared, git-tracked</div>
                  </div>
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
