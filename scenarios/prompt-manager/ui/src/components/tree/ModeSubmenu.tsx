/**
 * ModeSubmenu - Nested menu showing available folder paths.
 *
 * Lists existing mode paths from the skill tree
 * Includes a "New folder..." option to create custom paths
 */

import { useState, useCallback, useMemo } from 'react'
import { ChevronRight, FolderPlus, Folder } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ModeSubmenuProps {
  /** Current modes of the skill */
  currentModes: string[]
  /** All available mode paths from existing skills */
  availableModePaths: string[][]
  /** Callback when a path is selected */
  onSelectPath: (path: string[]) => void
  /** Callback when "New folder..." is selected */
  onCreateNewFolder: () => void
  className?: string
}

/**
 * Mode submenu component for context menu.
 */
export function ModeSubmenu({
  currentModes,
  availableModePaths,
  onSelectPath,
  onCreateNewFolder,
  className,
}: ModeSubmenuProps) {
  const [isOpen, setIsOpen] = useState(false)

  // Get unique top-level paths (deduplicated)
  const uniquePaths = useMemo(() => {
    const pathStrings = new Set<string>()
    const paths: string[][] = []

    for (const path of availableModePaths) {
      if (path.length > 0) {
        const pathStr = path.join('/')
        if (!pathStrings.has(pathStr)) {
          pathStrings.add(pathStr)
          paths.push(path)
        }
      }
    }

    // Sort by path string
    return paths.sort((a, b) => a.join('/').localeCompare(b.join('/')))
  }, [availableModePaths])

  const currentPathString = currentModes.filter(Boolean).join('/')

  const handleSelect = useCallback((path: string[]) => {
    onSelectPath(path)
  }, [onSelectPath])

  return (
    <div
      className={cn('relative', className)}
      onMouseEnter={() => setIsOpen(true)}
      onMouseLeave={() => setIsOpen(false)}
    >
      {/* Trigger */}
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

      {/* Submenu */}
      {isOpen && (
        <div
          className={cn(
            'absolute left-full top-0 ml-1 min-w-[180px] max-h-64 overflow-y-auto',
            'bg-popover border border-border rounded-md shadow-lg',
            'animate-in fade-in-0 zoom-in-95 duration-100'
          )}
        >
          <div className="p-1">
            {/* Root option */}
            <button
              type="button"
              onClick={() => handleSelect([])}
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
            {uniquePaths.map((path) => {
              const pathStr = path.join('/')
              const isSelected = pathStr === currentPathString

              return (
                <button
                  key={pathStr}
                  type="button"
                  onClick={() => handleSelect(path)}
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
              onClick={onCreateNewFolder}
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
  )
}
