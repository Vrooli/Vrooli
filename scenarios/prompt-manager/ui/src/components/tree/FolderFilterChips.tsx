/**
 * FolderFilterChips - Toggle buttons for filtering by storage location.
 *
 * Shows folder options (core, local, drafts) as toggleable chips.
 * Simpler than TagFilterChips since there are only 3 fixed folders.
 */

import { Database, HardDrive, FileEdit } from 'lucide-react'
import { cn } from '@/lib/utils'

interface FolderFilterChipsProps {
  /** Currently selected folders */
  selectedFolders: string[]
  /** All available folders in the skills */
  availableFolders: string[]
  /** Callback when folder selection changes */
  onToggleFolder: (folder: string) => void
  className?: string
}

/** Folder display info */
const FOLDER_INFO: Record<string, { label: string; icon: typeof Database; color: string }> = {
  core: {
    label: 'Core',
    icon: Database,
    color: 'text-blue-400 bg-blue-500/20',
  },
  local: {
    label: 'Local',
    icon: HardDrive,
    color: 'text-green-400 bg-green-500/20',
  },
  drafts: {
    label: 'Drafts',
    icon: FileEdit,
    color: 'text-amber-400 bg-amber-500/20',
  },
}

/**
 * Displays folder filter options as toggleable chips.
 */
export function FolderFilterChips({
  selectedFolders,
  availableFolders,
  onToggleFolder,
  className,
}: FolderFilterChipsProps) {
  if (availableFolders.length === 0) {
    return null
  }

  return (
    <div className={cn('flex items-center gap-1', className)}>
      {availableFolders.map((folder) => {
        const info = FOLDER_INFO[folder]
        if (!info) return null

        const isSelected = selectedFolders.includes(folder)
        const Icon = info.icon

        return (
          <button
            key={folder}
            type="button"
            onClick={() => onToggleFolder(folder)}
            className={cn(
              'inline-flex items-center gap-1 px-2 py-0.5',
              'text-[10px] rounded-full transition-colors',
              'whitespace-nowrap flex-shrink-0',
              isSelected
                ? info.color
                : 'bg-muted/50 text-muted-foreground hover:bg-muted'
            )}
            title={`${isSelected ? 'Hide' : 'Show'} ${info.label.toLowerCase()} skills`}
          >
            <Icon className="h-3 w-3" />
            {info.label}
          </button>
        )
      })}
    </div>
  )
}
