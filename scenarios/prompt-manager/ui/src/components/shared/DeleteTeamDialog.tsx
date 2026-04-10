/**
 * DeleteTeamDialog - Custom dialog for deleting a team with optional agent cleanup.
 *
 * Shows which agents are exclusive to this team (not in any other team)
 * and lets the user optionally select them for deletion too.
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { AlertTriangle, Loader2 } from 'lucide-react'
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
  const confirmButtonRef = useRef<HTMLButtonElement>(null)

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
      setTimeout(() => confirmButtonRef.current?.focus(), 0)
    }
  }, [isOpen])

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
      <div className="flex gap-3">
        <button
          type="button"
          onClick={onClose}
          disabled={isLoading}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
            'bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white',
            'border border-white/10 transition-colors',
            isLoading && 'opacity-50 cursor-not-allowed'
          )}
        >
          Cancel
        </button>
        <button
          ref={confirmButtonRef}
          type="button"
          onClick={handleConfirm}
          disabled={isLoading || isFetching}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            'bg-red-600 text-white hover:bg-red-500',
            (isLoading || isFetching) && 'opacity-50 cursor-not-allowed'
          )}
        >
          {confirmLabel}
        </button>
      </div>
    </Dialog>
  )
}
