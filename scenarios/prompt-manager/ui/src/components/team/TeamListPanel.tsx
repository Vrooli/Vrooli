/**
 * TeamListPanel - Panel for listing and managing teams.
 */

import { useState, useMemo } from 'react'
import { Download, Plus, Power, Trash2, Upload, Users, Network } from 'lucide-react'
import { CollectionList } from '@vrooli/react-component-library/CollectionList/1.0.0'
import type { RowAction } from '@vrooli/react-component-library/useCollection/1'
import { cn } from '@/lib/utils'
import { useTeamData } from '@/hooks/useTeamData'
import { selectors } from '@/constants/selectors'
import { buildDefaultCreateTeamRequest } from '@/lib/schemas'
import * as teamService from '@/services/teamService'
import { CCTeamImportModal } from './CCTeamImportModal'
import type { HeartbeatControlStatus } from '@/services/heartbeatService'
import type { Team } from '@/types/team'
import { Dialog } from '@/components/shared/Dialog'
import { TeamModelHelp, TeamPurposeBadges, teamPresets, teamPurposeLabels, teamLifetimeLabels } from './TeamPurposePanel'
import { TeamEffortsPanel } from './TeamEffortsPanel'

function teamActions(
  onToggleEnabled: (id: string) => void,
  onExport: (id: string, name: string) => void,
  onDelete: (id: string) => void,
): RowAction<Team>[] {
  return [
    { id: 'toggle', label: 'Toggle Team', icon: <Power />, onSelect: (rows) => rows[0] && onToggleEnabled(rows[0].id) },
    { id: 'export', label: 'Export Claude Code Config', icon: <Download />, onSelect: (rows) => rows[0] && onExport(rows[0].id, rows[0].displayName) },
    { id: 'delete', label: 'Delete Team', icon: <Trash2 />, tone: 'destructive', separatorBefore: true, onSelect: (rows) => rows[0] && onDelete(rows[0].id) },
  ]
}

interface TeamListPanelProps {
  selectedTeamId: string | null
  onSelectTeam: (id: string) => void
  /** Filter teams by display name */
  searchQuery?: string
  className?: string
  /** Called when user toggles team enabled/disabled via context menu */
  onToggleTeamEnabled?: (teamId: string) => void
  /** Selection mode: show checkboxes and toggle instead of navigate */
  isSelectMode?: boolean
  /** IDs currently selected (for checkbox state) */
  selectedIds?: Set<string>
  /** Called when an item is toggled in selection mode */
  onToggleSelection?: (id: string) => void
  heartbeatControlStatus?: HeartbeatControlStatus | null
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
  isSelectMode,
  selectedIds,
  onToggleSelection,
  heartbeatControlStatus,
}: TeamListPanelProps) {
  const { teams, isLoading, isError, createTeam, deleteTeam, refetch } = useTeamData()
  const [purposeFilter, setPurposeFilter] = useState('')
  const [lifetimeFilter, setLifetimeFilter] = useState('')
  const [preset, setPreset] = useState<keyof typeof teamPresets>('custom')
  const [effortsOpen, setEffortsOpen] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const filteredTeams = useMemo(() => {
    const lower = searchQuery?.toLowerCase() ?? ''
    return teams.filter((team) => {
      const purpose = team.purpose ? teamPurposeLabels[team.purpose] : 'Purpose unspecified'
      const lifetime = team.lifetime ? teamLifetimeLabels[team.lifetime] : 'Lifetime unspecified'
      return `${team.displayName} ${team.id} ${purpose} ${lifetime}`.toLowerCase().includes(lower)
        && (!purposeFilter || (team.purpose || 'unspecified') === purposeFilter)
        && (!lifetimeFilter || (team.lifetime || 'unspecified') === lifetimeFilter)
    })
  }, [teams, searchQuery, purposeFilter, lifetimeFilter])
  const [importModalOpen, setImportModalOpen] = useState(false)

  const handleCreateTeam = async () => {
    setCreating(true)
    setCreateError(null)
    try {
      const selected = teamPresets[preset]
      const name = `Team ${teams.length + 1}`
      const newTeam = await createTeam({ ...buildDefaultCreateTeamRequest(name), purpose: selected.purpose, lifetime: selected.lifetime })
      onSelectTeam(newTeam.id)
    } catch (error) {
      setCreateError(error instanceof Error ? error.message : 'Team could not be created.')
    } finally { setCreating(false) }
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


  const actions = useMemo(
    () => teamActions(
      onToggleTeamEnabled ?? (() => {}),
      (id, name) => void handleExportTeam(id, name),
      (id) => void handleDeleteTeam(id),
    ),
    [deleteTeam, onToggleTeamEnabled, teamService.exportClaudeCodeTeam],
  )

  const syncSelection = (keys: string[]) => {
    if (!onToggleSelection) return
    const next = new Set(keys)
    selectedIds?.forEach((id) => { if (!next.has(id)) onToggleSelection(id) })
    next.forEach((id) => { if (!selectedIds?.has(id)) onToggleSelection(id) })
  }

  const statusByTeam = useMemo(() => {
    const map = new Map<string, HeartbeatControlStatus>()
    for (const team of heartbeatControlStatus?.teams ?? []) {
      if (team.teamId) map.set(team.teamId, team)
    }
    return map
  }, [heartbeatControlStatus?.teams])

  const renderHeartbeatChip = (teamId: string) => {
    const status = statusByTeam.get(teamId)
    if (!status || status.status === 'active') return null
    const isPaused = status.status === 'paused-auto-idle' || status.status === 'paused-manual'
    return (
      <span
        className={cn(
          'px-1.5 py-0.5 rounded-full text-[10px] font-medium uppercase tracking-wide',
          isPaused
            ? 'bg-red-500/15 text-red-500'
            : 'bg-amber-500/15 text-amber-500'
        )}
        title={status.pausedReason || status.resumeHint || 'Heartbeat auto-pause state'}
      >
        {status.status === 'warning-idle-soon' ? 'Idle soon' : status.scope === 'global' ? 'Paused global' : 'Paused'}
      </span>
    )
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
      className={cn('flex flex-col min-h-0', className)}
      data-testid={selectors.teams.list}
    >
      <div className="space-y-2 border-b border-border px-3 py-2">
        <TeamModelHelp />
        <div className="grid grid-cols-2 gap-2">
          <select aria-label="Filter teams by purpose" value={purposeFilter} onChange={event => setPurposeFilter(event.target.value)} className="min-w-0 rounded border border-border bg-background p-1 text-xs">
            <option value="">All purposes</option>
            {Object.entries(teamPurposeLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            <option value="unspecified">Unspecified</option>
          </select>
          <select aria-label="Filter teams by lifetime" value={lifetimeFilter} onChange={event => setLifetimeFilter(event.target.value)} className="min-w-0 rounded border border-border bg-background p-1 text-xs">
            <option value="">All lifetimes</option>
            {Object.entries(teamLifetimeLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            <option value="unspecified">Unspecified</option>
          </select>
        </div>
        <button type="button" onClick={() => setEffortsOpen(true)} className="flex items-center gap-2 text-xs text-primary hover:underline"><Network className="h-3.5 w-3.5" />Efforts and orchestration</button>
        <p className="text-[11px] text-muted-foreground">Includes observed efforts without a registered team.</p>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto py-1">
        <CollectionList
          items={filteredTeams}
          getKey={(team) => team.id}
          label="Teams"
          height="100%"
          selection={{ mode: isSelectMode ? 'multi' : 'none', selected: selectedIds ? [...selectedIds] : undefined, onChange: syncSelection }}
          onOpen={(team) => onSelectTeam(team.id)}
          actions={actions}
          empty={teams.length === 0 ? (
            <div className="px-3 py-8 text-center"><Users className="mx-auto mb-2 h-8 w-8 text-muted-foreground" /><p className="mb-4 text-xs text-muted-foreground">No teams yet</p><button type="button" onClick={() => void handleCreateTeam()} className="text-xs text-primary hover:underline">Create your first team</button></div>
          ) : (
            <div className="px-3 py-8 text-center"><Users className="mx-auto mb-2 h-8 w-8 text-muted-foreground opacity-60" /><p className="text-xs text-muted-foreground">No matching teams</p></div>
          )}
          renderItem={(team) => <button
            type="button"
            className={cn('flex w-full items-center gap-3 px-3 py-2 text-left', !isSelectMode && selectedTeamId === team.id && 'bg-primary/10')}
            data-testid={selectors.teams.row}
            data-team-id={team.id}
            onPointerDown={(event) => event.stopPropagation()}
            onClick={(event) => {
              event.stopPropagation()
              if (!isSelectMode) onSelectTeam(team.id)
            }}
          >
            <div className={cn('flex h-8 w-8 shrink-0 items-center justify-center rounded-full', team.enabled ? 'bg-primary/20' : 'bg-muted')}><Users className={cn('h-4 w-4', team.enabled ? 'text-primary' : 'text-muted-foreground')} /></div>
            <div className="min-w-0 flex-1 space-y-1"><p className="truncate text-sm font-medium text-foreground">{team.displayName}</p><TeamPurposeBadges team={team} /><div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground"><span>{team.memberCount} member{team.memberCount !== 1 ? 's' : ''}</span><span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-medium">{team.enabled ? 'Enabled' : 'Disabled'}</span>{renderHeartbeatChip(team.id)}</div></div>
          </button>}
          className="h-full w-full"
        />
      </div>

      {/* Footer - New team + Import buttons (hidden in select mode) */}
      {!isSelectMode && (
        <div className="flex-shrink-0 px-3 py-3 border-t border-border space-y-2">
          <label className="block text-xs text-muted-foreground">New team preset
            <select aria-label="New team preset" value={preset} disabled={creating} onChange={event => setPreset(event.target.value as keyof typeof teamPresets)} className="mt-1 block w-full rounded border border-border bg-background p-2 text-foreground">
              {Object.entries(teamPresets).map(([value, item]) => <option key={value} value={value}>{item.label}</option>)}
            </select>
          </label>
          <button
            type="button"
            onClick={() => void handleCreateTeam()}
            disabled={creating}
            className={cn(
              'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
              'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
            )}
            data-testid={selectors.teams.newButton}
          >
            <Plus className="h-4 w-4" />
            {creating ? 'Creating…' : 'New Team'}
          </button>
          {createError ? <p role="alert" className="text-xs text-destructive">{createError}</p> : null}
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
      )}

      <Dialog isOpen={effortsOpen} onClose={() => setEffortsOpen(false)} title="Efforts and orchestration" maxWidth="max-w-5xl" appearance="theme">
        {effortsOpen ? <TeamEffortsPanel teams={teams} teamRegistryComplete onOpenTeam={id => { onSelectTeam(id); setEffortsOpen(false) }} /> : null}
      </Dialog>

      {/* Import modal */}
      <CCTeamImportModal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onImported={handleImported}
      />
    </div>
  )
}
