/**
 * MemberPickerModal - Modal for selecting agents to add as members.
 */

import { useState } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'

export interface MemberPickerModalProps {
  availableAgents: Agent[]
  onSelect: (agentId: string) => void
  onClose: () => void
}

export function MemberPickerModal({ availableAgents, onSelect, onClose }: MemberPickerModalProps) {
  const [search, setSearch] = useState('')

  // Filter agents by search query
  const filteredAgents = availableAgents.filter(
    (agent) =>
      agent.displayName.toLowerCase().includes(search.toLowerCase()) ||
      agent.description?.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative w-full max-w-md mx-4 bg-card border border-border rounded-xl shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="font-medium">Add Team Member</h3>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Search */}
        <div className="px-4 py-3 border-b border-border">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search agents..."
            className={cn(
              'w-full px-3 py-2 text-sm',
              'bg-muted border border-border rounded-lg',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
            autoFocus
          />
        </div>

        {/* Agents list */}
        <div className="max-h-64 overflow-y-auto">
          {filteredAgents.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-muted-foreground">
              {availableAgents.length === 0
                ? 'All agents are already team members'
                : 'No agents match your search'}
            </div>
          ) : (
            <ul className="p-2 space-y-1">
              {filteredAgents.map((agent) => (
                <li key={agent.id}>
                  <button
                    type="button"
                    onClick={() => onSelect(agent.id)}
                    className={cn(
                      'w-full flex items-center gap-3 px-3 py-2',
                      'rounded-lg text-left',
                      'hover:bg-muted transition-colors'
                    )}
                  >
                    {/* Agent color badge */}
                    <AgentColorBadge appearance={agent.appearance} size="sm" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium">{agent.displayName}</p>
                      {agent.description && (
                        <p className="text-xs text-muted-foreground line-clamp-1">
                          {agent.description}
                        </p>
                      )}
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
