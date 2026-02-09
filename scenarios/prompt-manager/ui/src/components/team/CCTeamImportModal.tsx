/**
 * CCTeamImportModal - Modal for importing Claude Code teams.
 *
 * Fetches available CC teams from disk and allows the user to import one.
 */

import { useEffect, useState, useMemo } from 'react'
import { Search, Users, Loader2, AlertCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import * as teamService from '@/services/teamService'
import type { AvailableCCTeam } from '@/types/team'

interface CCTeamImportModalProps {
  open: boolean
  onClose: () => void
  onImported: (teamId: string) => void
}

/**
 * Modal/popover for browsing and importing Claude Code teams.
 */
export function CCTeamImportModal({ open, onClose, onImported }: CCTeamImportModalProps) {
  const [teams, setTeams] = useState<AvailableCCTeam[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [importingName, setImportingName] = useState<string | null>(null)
  const [filter, setFilter] = useState('')

  useEffect(() => {
    if (!open) return
    let active = true

    const load = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const data = await teamService.listAvailableCCTeams()
        if (active) setTeams(data)
      } catch {
        if (active) setError('Failed to load Claude Code teams')
      } finally {
        if (active) setIsLoading(false)
      }
    }

    void load()
    return () => { active = false }
  }, [open])

  const filtered = useMemo(() => {
    if (!filter) return teams
    const lower = filter.toLowerCase()
    return teams.filter((t) => t.name.toLowerCase().includes(lower))
  }, [teams, filter])

  const handleImport = async (name: string) => {
    setImportingName(name)
    setError(null)
    try {
      const imported = await teamService.importClaudeCodeTeam(name)
      onImported(imported.id)
      onClose()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Import failed'
      if (msg.includes('409') || msg.includes('already exists')) {
        setError(`Team "${name}" already exists in prompt-manager`)
      } else {
        setError(msg)
      }
    } finally {
      setImportingName(null)
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
        onKeyDown={(e) => e.key === 'Escape' && onClose()}
        role="button"
        tabIndex={-1}
        aria-label="Close modal"
      />

      {/* Modal */}
      <div className="relative bg-popover border border-border rounded-xl shadow-lg w-full max-w-md mx-4 max-h-[70vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h2 className="text-sm font-medium text-foreground">Import from Claude Code</h2>
          <button
            type="button"
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground text-lg leading-none px-1"
          >
            &times;
          </button>
        </div>

        {/* Search */}
        <div className="px-4 py-2 border-b border-border">
          <div className="relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="Filter teams..."
              className="w-full pl-7 pr-3 py-1.5 text-sm bg-muted rounded-md border-none outline-none placeholder:text-muted-foreground/60"
            />
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-2">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : error ? (
            <div className="flex items-start gap-2 px-3 py-4">
              <AlertCircle className="h-4 w-4 text-destructive flex-shrink-0 mt-0.5" />
              <p className="text-sm text-destructive">{error}</p>
            </div>
          ) : filtered.length === 0 ? (
            <div className="px-3 py-8 text-center">
              <Users className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
              <p className="text-xs text-muted-foreground">
                {teams.length === 0
                  ? 'No Claude Code teams found'
                  : 'No teams match your filter'}
              </p>
              {teams.length === 0 && (
                <p className="text-xs text-muted-foreground/70 mt-1">
                  Teams are stored at ~/.claude/teams/
                </p>
              )}
            </div>
          ) : (
            filtered.map((team) => (
              <button
                key={team.name}
                type="button"
                onClick={() => void handleImport(team.name)}
                disabled={importingName !== null}
                className={cn(
                  'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-left',
                  'hover:bg-muted/50 transition-colors',
                  importingName === team.name && 'opacity-60'
                )}
              >
                <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center flex-shrink-0">
                  <Users className="h-4 w-4 text-primary" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground truncate">
                    {team.name}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {team.memberCount} member{team.memberCount !== 1 ? 's' : ''}
                  </p>
                </div>
                {importingName === team.name && (
                  <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                )}
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
