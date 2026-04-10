/**
 * AgentContextMenu - Context menu for agent items in the sidebar.
 *
 * Appears on right-click of agent rows, providing options like:
 * - Duplicate agent
 * - Customize appearance
 * - Preview prompt
 * - Delete agent
 */

import { useCallback } from 'react'
import { Copy, Palette, Eye, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Popover } from '@/components/shared/Popover'
import { selectors } from '@/constants/selectors'

interface AgentContextMenuProps {
  x: number
  y: number
  agentId: string
  agentName: string
  onClose: () => void
  onDuplicate: (agentId: string) => void
  onCustomize: (agentId: string) => void
  onPreviewPrompt: (agentId: string) => void
  onDelete: (agentId: string) => void
}

/**
 * Context menu component for agent right-click actions.
 */
export function AgentContextMenu({
  x,
  y,
  agentId,
  agentName,
  onClose,
  onDuplicate,
  onCustomize,
  onPreviewPrompt,
  onDelete,
}: AgentContextMenuProps) {
  const handleDuplicate = useCallback(() => {
    onDuplicate(agentId)
    onClose()
  }, [onDuplicate, agentId, onClose])

  const handleCustomize = useCallback(() => {
    onCustomize(agentId)
    onClose()
  }, [onCustomize, agentId, onClose])

  const handlePreviewPrompt = useCallback(() => {
    onPreviewPrompt(agentId)
    onClose()
  }, [onPreviewPrompt, agentId, onClose])

  const handleDelete = useCallback(() => {
    onDelete(agentId)
    onClose()
  }, [onDelete, agentId, onClose])

  return (
    <Popover
      isOpen
      onClose={onClose}
      x={x}
      y={y}
      delayClickOutside
      testId={selectors.agents.contextMenu}
    >
      <div className="p-1">
        {/* Duplicate */}
        <button
          type="button"
          onClick={handleDuplicate}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-foreground hover:bg-muted transition-colors',
            'cursor-pointer outline-none',
          )}
        >
          <Copy className="h-4 w-4" />
          <span>Duplicate "{agentName}"</span>
        </button>

        {/* Customize Appearance */}
        <button
          type="button"
          onClick={handleCustomize}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-foreground hover:bg-muted transition-colors',
            'cursor-pointer outline-none',
          )}
        >
          <Palette className="h-4 w-4" />
          <span>Customize Appearance</span>
        </button>

        {/* Preview Prompt */}
        <button
          type="button"
          onClick={handlePreviewPrompt}
          className={cn(
            'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm',
            'text-foreground hover:bg-muted transition-colors',
            'cursor-pointer outline-none',
          )}
        >
          <Eye className="h-4 w-4" />
          <span>Preview Prompt</span>
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
          <span>Delete Agent</span>
        </button>
      </div>
    </Popover>
  )
}
