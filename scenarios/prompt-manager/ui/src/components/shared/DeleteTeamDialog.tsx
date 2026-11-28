/**
 * DeleteTeamDialog - Custom dialog for deleting a team with optional agent cleanup.
 *
 * Shows which agents are exclusive to this team (not in any other team)
 * and lets the user optionally select them for deletion too.
 */

import { useState, useEffect, useId, useCallback } from 'react'
import { AlertTriangle, Loader2 } from 'lucide-react'
import { OverlayActionBar, type OverlayActionList } from '@vrooli/react-component-library/OverlayActionBar/1.0.1'
import { cn } from '@/lib/utils'
import { Dialog } from './Dialog'
import { getTeamExclusiveMembers } from '@/services/teamService'
import type { ExclusiveMember } from '@/lib/schemas'

interface DeleteTeamDialogProps {
  isOpen: boolean
  teamId: string | null
  teamName: string
  onClose: () => void
  onConfirm: (agentIdsToDelete: string[]) => void
  isLoading?: boolean
}

export function DeleteTeamDialog({
  isOpen,
  teamId,
  teamName,
  onClose,
  onConfirm,
  isLoading = false,
}: DeleteTeamDialogProps) {
  const confirmActionTestId = useId()

  const [exclusiveMembers, setExclusiveMembers] = useState<ExclusiveMember[]>([])
  const [selectedAgentIds, setSelectedAgentIds] = useState<Set<string>>(new Set())
  const [isFetching, setIsFetching] = useState(false)

  // Fetch exclusive members when dialog opens
  useEffect(() => {
    if (!isOpen || !teamId) {
      setExclusiveMembers([])
      setSelectedAgentIds(new Set())
      return
    }

    let cancelled = false
    setIsFetching(true)

    void getTeamExclusiveMembers(teamId).then((members) => {
      if (!cancelled) {
        setExclusiveMembers(members)
        setSelectedAgentIds(new Set(members.map((m) => m.agentId)))
        setIsFetching(false)
      }
    })

    return () => {
      cancelled = true
    }
  }, [isOpen, teamId])

  // Auto-focus confirm button when dialog opens
  useEffect(() => {
    if (isOpen) {
      const timer = window.setTimeout(() => {
        document.querySelector<HTMLButtonElement>(`[data-testid="${confirmActionTestId}"]`)?.focus()
      }, 0)
      return () => window.clearTimeout(timer)
    }
  }, [confirmActionTestId, isOpen])

  const toggleAgent = useCallback((agentId: string) => {
    setSelectedAgentIds((prev) => {
      const next = new Set(prev)
      if (next.has(agentId)) {
        next.delete(agentId)
      } else {
        next.add(agentId)
      }
      return next
    })
  }, [])

  const toggleAll = useCallback(() => {
    if (selectedAgentIds.size === exclusiveMembers.length) {
      setSelectedAgentIds(new Set())
    } else {
      setSelectedAgentIds(new Set(exclusiveMembers.map((m) => m.agentId)))
    }
  }, [selectedAgentIds.size, exclusiveMembers])

  const handleConfirm = useCallback(() => {
    onConfirm(Array.from(selectedAgentIds))
  }, [onConfirm, selectedAgentIds])

  const agentCount = selectedAgentIds.size
  const confirmLabel = isLoading
    ? 'Deleting...'
    : agentCount > 0
      ? `Delete team & ${agentCount} agent${agentCount !== 1 ? 's' : ''}`
      : 'Delete team'

  const actions = [
    { id: 'delete-team', label: confirmLabel, onSelect: handleConfirm, disabled: isLoading || isFetching, primary: true, testId: confirmActionTestId },
    { id: 'cancel', label: 'Cancel', onSelect: onClose, disabled: isLoading },
  ] satisfies OverlayActionList

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      isLoading={isLoading}
      maxWidth="max-w-md"
      titleId="delete-team-title"
      descriptionId="delete-team-description"
    >
      {/* Icon */}
      <div className="w-12 h-12 mx-auto mb-4 rounded-full flex items-center justify-center bg-red-500/20">
        <AlertTriangle className="h-6 w-6 text-red-400" />
      </div>

      {/* Title */}
      <h2
        id="delete-team-title"
        className="text-lg font-semibold text-white text-center mb-2"
      >
        Delete Team
      </h2>

      {/* Message */}
      <p
        id="delete-team-description"
        className="text-sm text-slate-400 text-center mb-4"
      >
        Are you sure you want to delete &ldquo;{teamName}&rdquo;? This cannot be undone.
      </p>

      {/* Exclusive members section */}
      {isFetching ? (
        <div className="flex items-center justify-center py-3 mb-4">
          <Loader2 className="h-4 w-4 animate-spin text-slate-400 mr-2" />
          <span className="text-sm text-slate-400">Checking for exclusive agents...</span>
        </div>
      ) : exclusiveMembers.length > 0 ? (
        <div className="mb-4">
          <div className="flex items-center justify-between mb-2">
            <p className="text-sm text-slate-300">
              These agents only belong to this team:
            </p>
            <button
              type="button"
              onClick={toggleAll}
              disabled={isLoading}
              className="text-xs text-primary hover:text-primary/80 transition-colors"
            >
              {selectedAgentIds.size === exclusiveMembers.length ? 'Deselect all' : 'Select all'}
            </button>
          </div>
          <div className="max-h-40 overflow-y-auto rounded-lg border border-white/10 bg-slate-800/50">
            {exclusiveMembers.map((member) => (
              <label
                key={member.agentId}
                className={cn(
                  'flex items-center gap-3 px-3 py-2 cursor-pointer',
                  'hover:bg-white/5 transition-colors',
                  isLoading && 'opacity-50 cursor-not-allowed'
                )}
              >
                <input
                  type="checkbox"
                  checked={selectedAgentIds.has(member.agentId)}
                  onChange={() => toggleAgent(member.agentId)}
                  disabled={isLoading}
                  className="rounded border-slate-600 bg-slate-700 text-primary focus:ring-primary/50"
                />
                <div className="min-w-0 flex-1">
                  <span className="text-sm text-white truncate block">
                    {member.displayName}
                  </span>
                  <span className="text-xs text-slate-500 truncate block">
                    {member.agentId}
                  </span>
                </div>
              </label>
            ))}
          </div>
        </div>
      ) : null}

      {/* Actions */}
      <OverlayActionBar actions={actions} className="prompt-overlay-actions prompt-overlay-actions--danger prompt-overlay-actions--secondary-left" />
    </Dialog>
  )
}
