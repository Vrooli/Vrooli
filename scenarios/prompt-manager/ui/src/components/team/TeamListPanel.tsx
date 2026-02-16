/**
 * TeamListPanel - Panel for listing and managing teams.
 */

import { useState, useMemo, useCallback } from 'react'
import { Plus, Users, Trash2, Download, Upload } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTeamData } from '@/hooks/useTeamData'
import { selectors } from '@/constants/selectors'
import * as teamService from '@/services/teamService'
import { CCTeamImportModal } from './CCTeamImportModal'
import { TeamContextMenu } from '@/components/team/TeamContextMenu'

interface TeamListPanelProps {
  selectedTeamId: string | null
  onSelectTeam: (id: string) => void
  /** Filter teams by display name */
  searchQuery?: string
  className?: string
  /** Called when user toggles team enabled/disabled via context menu */
  onToggleTeamEnabled?: (teamId: string) => void
}

/**
 * Trigger a browser download of a JSON object.
 */
function downloadJson(data: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/**
 * Team list panel for the sidebar.
 */
export function TeamListPanel({
  selectedTeamId,
  onSelectTeam,
  searchQuery,
  className,
  onToggleTeamEnabled,
}: TeamListPanelProps) {
  const { teams, isLoading, isError, createTeam, deleteTeam, refetch } = useTeamData()

  const filteredTeams = useMemo(() => {
    if (!searchQuery) return teams
    const lower = searchQuery.toLowerCase()
    return teams.filter((t) => t.displayName.toLowerCase().includes(lower))
  }, [teams, searchQuery])
  const [importModalOpen, setImportModalOpen] = useState(false)

  // Context menu state
  const [contextMenu, setContextMenu] = useState<{
    x: number
    y: number
    teamId: string
    teamName: string
    isEnabled: boolean
  } | null>(null)

  const handleCreateTeam = async () => {
    const name = `Team ${teams.length + 1}`
    const newTeam = await createTeam({
      displayName: name,
    })
    // Auto-select the newly created team
    onSelectTeam(newTeam.id)
  }

  const handleDeleteTeam = async (id: string) => {
    await deleteTeam(id)
  }

  const handleExportTeam = async (teamId: string, teamName: string) => {
    try {
      const data = await teamService.exportClaudeCodeTeam(teamId)
      downloadJson(data, `${teamName}-cc-config.json`)
    } catch (err) {
      console.error('Export failed:', err)
    }
  }

  const handleImported = (teamId: string) => {
    refetch()
    onSelectTeam(teamId)
  }

  const handleContextMenu = useCallback((e: React.MouseEvent, teamId: string, teamName: string, isEnabled: boolean) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ x: e.clientX, y: e.clientY, teamId, teamName, isEnabled })
  }, [])

  const handleCloseContextMenu = useCallback(() => {
    setContextMenu(null)
  }, [])

  if (isLoading) {
    return (
      <div className={cn('flex items-center justify-center py-8', className)}>
        <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className={cn('px-3 py-8 text-center', className)}>
        <p className="text-sm text-destructive">Failed to load teams</p>
      </div>
    )
  }

  return (
    <div
      className={cn('flex flex-col h-full', className)}
      data-testid={selectors.teams.list}
    >
      {/* Team list */}
      <div className="flex-1 overflow-y-auto py-1">
        {teams.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Users className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
            <p className="text-xs text-muted-foreground mb-4">No teams yet</p>
            <button
              type="button"
              onClick={() => void handleCreateTeam()}
              className="text-xs text-primary hover:underline"
            >
              Create your first team
            </button>
          </div>
        ) : filteredTeams.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Users className="h-8 w-8 mx-auto mb-2 text-muted-foreground opacity-60" />
            <p className="text-xs text-muted-foreground">No matching teams</p>
          </div>
        ) : (
          filteredTeams.map((team) => (
            <button
              key={team.id}
              type="button"
              onClick={() => onSelectTeam(team.id)}
              onContextMenu={(e) => handleContextMenu(e, team.id, team.displayName, team.enabled)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-left group',
                'hover:bg-muted/50 transition-colors',
                selectedTeamId === team.id && 'bg-primary/10'
              )}
              data-testid={selectors.teams.row}
              data-team-id={team.id}
            >
              {/* Team icon */}
              <div
                className={cn(
                  'w-8 h-8 rounded-full flex-shrink-0 flex items-center justify-center',
                  team.enabled ? 'bg-primary/20' : 'bg-muted'
                )}
              >
                <Users className={cn('h-4 w-4', team.enabled ? 'text-primary' : 'text-muted-foreground')} />
              </div>

              {/* Team info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {team.displayName}
                </p>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span>
                    {team.memberCount} member{team.memberCount !== 1 ? 's' : ''}
                  </span>
                  <span
                    className={cn(
                      'px-1.5 py-0.5 rounded-full text-[10px] font-medium uppercase tracking-wide',
                      team.enabled
                        ? 'bg-emerald-500/15 text-emerald-500'
                        : 'bg-slate-500/20 text-slate-400'
                    )}
                  >
                    {team.enabled ? 'On' : 'Off'}
                  </span>
                </div>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    void handleExportTeam(team.id, team.displayName)
                  }}
                  className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                  title="Export as Claude Code config"
                  data-testid={selectors.teams.exportButton}
                >
                  <Download className="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    void handleDeleteTeam(team.id)
                  }}
                  className="p-1 rounded hover:bg-destructive/20 text-muted-foreground hover:text-destructive transition-colors"
                  title="Delete team"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </button>
          ))
        )}
      </div>

      {/* Footer - New team + Import buttons */}
      <div className="flex-shrink-0 px-3 py-3 border-t border-border space-y-2">
        <button
          type="button"
          onClick={() => void handleCreateTeam()}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
          )}
          data-testid={selectors.teams.newButton}
        >
          <Plus className="h-4 w-4" />
          New Team
        </button>
        <button
          type="button"
          onClick={() => setImportModalOpen(true)}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'border border-border hover:bg-muted text-foreground rounded-lg transition-colors'
          )}
          data-testid={selectors.teams.importButton}
        >
          <Upload className="h-4 w-4" />
          Import from Claude Code
        </button>
      </div>

      {/* Import modal */}
      <CCTeamImportModal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onImported={handleImported}
      />

      {/* Context menu */}
      {contextMenu && (
        <TeamContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          teamId={contextMenu.teamId}
          teamName={contextMenu.teamName}
          isEnabled={contextMenu.isEnabled}
          onClose={handleCloseContextMenu}
          onToggleEnabled={onToggleTeamEnabled ?? (() => {})}
          onExport={(id, name) => void handleExportTeam(id, name)}
          onDelete={(id) => void handleDeleteTeam(id)}
        />
      )}
    </div>
  )
}
