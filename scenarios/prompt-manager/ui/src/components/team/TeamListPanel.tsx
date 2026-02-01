/**
 * TeamListPanel - Panel for listing and managing teams.
 */

import { Plus, Users, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTeamData } from '@/hooks/useTeamData'
import { selectors } from '@/constants/selectors'

interface TeamListPanelProps {
  selectedTeamId: string | null
  onSelectTeam: (id: string) => void
  className?: string
}

/**
 * Team list panel for the sidebar.
 */
export function TeamListPanel({
  selectedTeamId,
  onSelectTeam,
  className,
}: TeamListPanelProps) {
  const { teams, isLoading, isError, createTeam, deleteTeam } = useTeamData()

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
        ) : (
          teams.map((team) => (
            <button
              key={team.id}
              type="button"
              onClick={() => onSelectTeam(team.id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-left group',
                'hover:bg-muted/50 transition-colors',
                selectedTeamId === team.id && 'bg-primary/10'
              )}
              data-testid={selectors.teams.row}
              data-team-id={team.id}
            >
              {/* Team icon */}
              <div className="w-8 h-8 rounded-full flex-shrink-0 flex items-center justify-center bg-primary/20">
                <Users className="h-4 w-4 text-primary" />
              </div>

              {/* Team info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {team.displayName}
                </p>
                <p className="text-xs text-muted-foreground">
                  {team.memberCount} member{team.memberCount !== 1 ? 's' : ''}
                </p>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
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

      {/* Footer - New team button */}
      <div className="flex-shrink-0 px-3 py-3 border-t border-border">
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
      </div>
    </div>
  )
}
