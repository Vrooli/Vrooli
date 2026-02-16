/**
 * TeamContextMenu - Context menu for team items in the sidebar.
 *
 * Appears on right-click of team rows, providing options like:
 * - Enable/Disable team
 * - Export Claude Code config
 * - Delete team
 */

import { useCallback } from 'react'
import { Power, Download, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Popover } from '@/components/shared/Popover'
import { selectors } from '@/constants/selectors'

interface TeamContextMenuProps {
  x: number
  y: number
  teamId: string
  teamName: string
  isEnabled: boolean
  onClose: () => void
  onToggleEnabled: (teamId: string) => void
  onExport: (teamId: string, teamName: string) => void
  onDelete: (teamId: string) => void
}

/**
 * Context menu component for team right-click actions.
 */
export function TeamContextMenu({
  x,
  y,
  teamId,
  teamName,
  isEnabled,
  onClose,
  onToggleEnabled,
  onExport,
  onDelete,
}: TeamContextMenuProps) {
  const handleToggleEnabled = useCallback(() => {
    onToggleEnabled(teamId)
    onClose()
  }, [onToggleEnabled, teamId, onClose])

  const handleExport = useCallback(() => {
    onExport(teamId, teamName)
    onClose()
  }, [onExport, teamId, teamName, onClose])

  const handleDelete = useCallback(() => {
    onDelete(teamId)
    onClose()
  }, [onDelete, teamId, onClose])

  return (
    <Popover
      isOpen
      onClose={onClose}
      x={x}
      y={y}
      delayClickOutside
      testId={selectors.teams.contextMenu}
    >
      <div className="p-1">
        {/* Enable/Disable */}
        <button
          type="button"
          onClick={handleToggleEnabled}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-foreground hover:bg-muted transition-colors',
            'cursor-pointer outline-none',
          )}
        >
          <Power className="h-4 w-4" />
          <span>{isEnabled ? 'Disable Team' : 'Enable Team'}</span>
        </button>

        {/* Export */}
        <button
          type="button"
          onClick={handleExport}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-foreground hover:bg-muted transition-colors',
            'cursor-pointer outline-none',
          )}
        >
          <Download className="h-4 w-4" />
          <span>Export Claude Code Config</span>
        </button>

        {/* Divider */}
        <div className="my-1 border-t border-border" />

        {/* Delete */}
        <button
          type="button"
          onClick={handleDelete}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-red-400 hover:bg-red-500/10 transition-colors',
            'cursor-pointer outline-none',
          )}
        >
          <Trash2 className="h-4 w-4" />
          <span>Delete Team</span>
        </button>
      </div>
    </Popover>
  )
}
