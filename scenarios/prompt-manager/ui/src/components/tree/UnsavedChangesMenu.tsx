/**
 * UnsavedChangesMenu - Anchored dropdown menu showing all unsaved items.
 *
 * Displays when clicking on the "X unsaved" badge in the sidebar header.
 * Allows users to:
 * - View all items with unsaved changes
 * - Save or discard individual items
 * - Save or discard all changes at once
 */

import { useState, useRef, useEffect, useCallback } from 'react'
import { ChevronDown, Save, X, FileText, User, Users } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Skill } from '@/types'
import type { Agent } from '@/types/agent'

interface UnsavedItem {
  id: string
  name: string
  type: 'skill' | 'agent' | 'team-member'
}

interface UnsavedChangesMenuProps {
  /** Total count of unsaved items */
  dirtyCount: number
  /** Set of dirty skill IDs */
  dirtySkillIds: Set<string>
  /** Set of dirty agent IDs */
  dirtyAgentIds: Set<string>
  /** Set of dirty team member IDs (agent IDs for team context) */
  dirtyTeamMemberIds: Set<string>
  /** All skills for name lookup */
  skills: Skill[]
  /** All agents for name lookup */
  agents: Agent[]
  /** Callback to select/open a skill */
  onSelectSkill?: (skillId: string) => void
  /** Callback to select/open an agent */
  onSelectAgent?: (agentId: string) => void
  /** Callback to save a specific skill */
  onSaveSkill?: (skillId: string) => Promise<void>
  /** Callback to discard changes for a specific skill */
  onDiscardSkill?: (skillId: string) => void
  /** Callback to save a specific agent */
  onSaveAgent?: (agentId: string) => Promise<void>
  /** Callback to discard changes for a specific agent */
  onDiscardAgent?: (agentId: string) => void
  /** Callback to save all changes */
  onSaveAll?: () => Promise<void>
  /** Callback to discard all changes */
  onDiscardAll?: () => void
  /** Whether save operation is in progress */
  isSaving?: boolean
  /** Optional className */
  className?: string
}

/**
 * Anchored menu showing unsaved changes with save/discard options.
 */
export function UnsavedChangesMenu({
  dirtyCount,
  dirtySkillIds,
  dirtyAgentIds,
  dirtyTeamMemberIds,
  skills,
  agents,
  onSelectSkill,
  onSelectAgent,
  onSaveSkill,
  onDiscardSkill,
  onSaveAgent,
  onDiscardAgent,
  onSaveAll,
  onDiscardAll,
  isSaving = false,
  className,
}: UnsavedChangesMenuProps) {
  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  // Build list of unsaved items
  const unsavedItems: UnsavedItem[] = []

  // Add dirty skills
  for (const skillId of dirtySkillIds) {
    const skill = skills.find((s) => s.id === skillId)
    unsavedItems.push({
      id: skillId,
      name: skill?.name ?? 'Unknown Skill',
      type: 'skill',
    })
  }

  // Add dirty agents
  for (const agentId of dirtyAgentIds) {
    const agent = agents.find((a) => a.id === agentId)
    unsavedItems.push({
      id: agentId,
      name: agent?.displayName ?? 'Unknown Agent',
      type: 'agent',
    })
  }

  // Add dirty team members (these are also agents)
  for (const memberId of dirtyTeamMemberIds) {
    // Skip if already added as an agent
    if (dirtyAgentIds.has(memberId)) continue
    const agent = agents.find((a) => a.id === memberId)
    unsavedItems.push({
      id: memberId,
      name: agent?.displayName ?? 'Unknown Member',
      type: 'team-member',
    })
  }

  // Close on click outside
  useEffect(() => {
    if (!isOpen) return

    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    // Delay adding listener to prevent immediate close on the same click
    const timeoutId = setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside)
    }, 0)

    return () => {
      clearTimeout(timeoutId)
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  // Close on Escape key
  useEffect(() => {
    if (!isOpen) return

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false)
      }
    }

    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [isOpen])

  const handleSelectItem = useCallback((item: UnsavedItem) => {
    if (item.type === 'skill' && onSelectSkill) {
      onSelectSkill(item.id)
      setIsOpen(false)
    } else if ((item.type === 'agent' || item.type === 'team-member') && onSelectAgent) {
      onSelectAgent(item.id)
      setIsOpen(false)
    }
  }, [onSelectSkill, onSelectAgent])

  const handleSaveItem = useCallback(async (item: UnsavedItem) => {
    if (item.type === 'skill' && onSaveSkill) {
      await onSaveSkill(item.id)
    } else if ((item.type === 'agent' || item.type === 'team-member') && onSaveAgent) {
      await onSaveAgent(item.id)
    }
  }, [onSaveSkill, onSaveAgent])

  const handleDiscardItem = useCallback((item: UnsavedItem) => {
    if (item.type === 'skill' && onDiscardSkill) {
      onDiscardSkill(item.id)
    } else if ((item.type === 'agent' || item.type === 'team-member') && onDiscardAgent) {
      onDiscardAgent(item.id)
    }
  }, [onDiscardSkill, onDiscardAgent])

  const handleSaveAll = useCallback(async () => {
    if (onSaveAll) {
      await onSaveAll()
      setIsOpen(false)
    }
  }, [onSaveAll])

  const handleDiscardAll = useCallback(() => {
    if (onDiscardAll) {
      onDiscardAll()
      setIsOpen(false)
    }
  }, [onDiscardAll])

  const getItemIcon = (type: UnsavedItem['type']) => {
    switch (type) {
      case 'skill':
        return <FileText className="h-3.5 w-3.5" />
      case 'agent':
        return <User className="h-3.5 w-3.5" />
      case 'team-member':
        return <Users className="h-3.5 w-3.5" />
    }
  }

  if (dirtyCount === 0) return null

  return (
    <div ref={menuRef} className={cn('relative', className)}>
      {/* Trigger button - the unsaved badge */}
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium',
          'bg-amber-500/20 text-amber-400 rounded',
          'hover:bg-amber-500/30 transition-colors cursor-pointer'
        )}
        title="Click to view unsaved changes"
      >
        {dirtyCount} unsaved
        <ChevronDown className={cn('h-3 w-3 transition-transform', isOpen && 'rotate-180')} />
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div
          className={cn(
            'absolute left-0 top-full mt-1 z-50',
            'bg-card border border-border rounded-lg shadow-lg',
            'min-w-[280px] max-w-[320px]',
            'animate-in fade-in-0 zoom-in-95 duration-100'
          )}
        >
          {/* Header */}
          <div className="flex items-center justify-between px-3 py-2 border-b border-border">
            <span className="text-xs font-medium text-foreground">
              Unsaved Changes ({dirtyCount})
            </span>
            <div className="flex items-center gap-1">
              {onSaveAll && (
                <button
                  type="button"
                  onClick={() => void handleSaveAll()}
                  disabled={isSaving}
                  className={cn(
                    'flex items-center gap-1 px-2 py-1 text-[10px] font-medium rounded',
                    'bg-primary/10 text-primary hover:bg-primary/20 transition-colors',
                    isSaving && 'opacity-50 cursor-not-allowed'
                  )}
                  title="Save all changes"
                >
                  <Save className="h-3 w-3" />
                  Save All
                </button>
              )}
              {onDiscardAll && (
                <button
                  type="button"
                  onClick={handleDiscardAll}
                  disabled={isSaving}
                  className={cn(
                    'flex items-center gap-1 px-2 py-1 text-[10px] font-medium rounded',
                    'bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors',
                    isSaving && 'opacity-50 cursor-not-allowed'
                  )}
                  title="Discard all changes"
                >
                  <X className="h-3 w-3" />
                  Discard
                </button>
              )}
            </div>
          </div>

          {/* Items list */}
          <div className="max-h-[300px] overflow-y-auto py-1">
            {unsavedItems.map((item) => (
              <div
                key={`${item.type}-${item.id}`}
                className="flex items-center justify-between px-1 py-0.5 group"
              >
                <button
                  type="button"
                  onClick={() => handleSelectItem(item)}
                  className={cn(
                    'flex items-center gap-2 min-w-0 flex-1 px-2 py-1 rounded',
                    'hover:bg-muted/50 transition-colors text-left'
                  )}
                  title={`Open ${item.name}`}
                >
                  <span className="text-muted-foreground flex-shrink-0">
                    {getItemIcon(item.type)}
                  </span>
                  <span className="text-xs text-foreground truncate">
                    {item.name}
                  </span>
                  <span className="text-[10px] text-muted-foreground capitalize flex-shrink-0">
                    ({item.type === 'team-member' ? 'team' : item.type})
                  </span>
                </button>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 pr-1">
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      void handleSaveItem(item)
                    }}
                    disabled={isSaving}
                    className={cn(
                      'p-1 rounded hover:bg-primary/20 text-primary transition-colors',
                      isSaving && 'opacity-50 cursor-not-allowed'
                    )}
                    title={`Save ${item.name}`}
                  >
                    <Save className="h-3 w-3" />
                  </button>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDiscardItem(item)
                    }}
                    disabled={isSaving}
                    className={cn(
                      'p-1 rounded hover:bg-destructive/20 text-destructive transition-colors',
                      isSaving && 'opacity-50 cursor-not-allowed'
                    )}
                    title={`Discard changes to ${item.name}`}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

/**
 * Collapsed version of the unsaved changes indicator (for collapsed sidebar).
 * Shows a small badge that can be clicked to expand the sidebar.
 */
export function UnsavedChangesCollapsedBadge({
  dirtyCount,
  onClick,
}: {
  dirtyCount: number
  onClick?: () => void
}) {
  if (dirtyCount === 0) return null

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-6 h-6 flex items-center justify-center text-[10px] font-medium',
        'bg-amber-500/20 text-amber-400 rounded-full',
        'hover:bg-amber-500/30 transition-colors cursor-pointer'
      )}
      title={`${dirtyCount} unsaved changes - click to expand`}
    >
      {dirtyCount}
    </button>
  )
}
