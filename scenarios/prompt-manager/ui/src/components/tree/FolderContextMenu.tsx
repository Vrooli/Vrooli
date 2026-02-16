/**
 * FolderContextMenu - Context menu for folder nodes in the tree.
 *
 * Appears on right-click of folder nodes, providing options like:
 * - Add skill to this folder
 * - Delete folder (with all skills in it)
 */

import { Plus, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Popover } from '@/components/shared/Popover'

interface FolderContextMenuProps {
  x: number
  y: number
  folderLabel: string
  skillCount: number
  onClose: () => void
  onAddSkill: () => void
  onDeleteFolder: () => void
}

/**
 * Context menu component for folder right-click actions.
 */
export function FolderContextMenu({
  x,
  y,
  folderLabel,
  skillCount,
  onClose,
  onAddSkill,
  onDeleteFolder,
}: FolderContextMenuProps) {
  const handleAddSkill = () => {
    onAddSkill()
    onClose()
  }

  const handleDeleteFolder = () => {
    onDeleteFolder()
    onClose()
  }

  return (
    <Popover isOpen onClose={onClose} x={x} y={y} delayClickOutside className="min-w-[200px]">
      <div className="p-1">
        <button
          type="button"
          onClick={handleAddSkill}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-foreground hover:bg-muted transition-colors',
            'cursor-pointer outline-none'
          )}
        >
          <Plus className="h-4 w-4" />
          <span>Add skill to "{folderLabel}"</span>
        </button>
        <div className="my-1 h-px bg-border" />
        <button
          type="button"
          onClick={handleDeleteFolder}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-red-400 hover:bg-red-500/10 transition-colors',
            'cursor-pointer outline-none'
          )}
        >
          <Trash2 className="h-4 w-4" />
          <span>Delete folder ({skillCount} skill{skillCount !== 1 ? 's' : ''})</span>
        </button>
      </div>
    </Popover>
  )
}
