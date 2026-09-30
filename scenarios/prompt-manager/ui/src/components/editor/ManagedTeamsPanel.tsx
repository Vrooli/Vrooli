import { useMemo, useState } from 'react'
import { useQueries, useQuery, useQueryClient } from '@tanstack/react-query'
import { Network, Plus, X } from 'lucide-react'
import type { TeamDetails, Team } from '@/types/team'
import type { ManagedTeamEdge } from '@/types/orgChart'
import { useTeamData } from '@/hooks/useTeamData'
import * as orgChartService from '@/services/orgChartService'
import { useEffortObservations } from '@/services/effortService'

/** Team-level supervision is deliberately separate from members and runtime contractors. */
export function ManagedTeamsPanel({ team }: { team: TeamDetails }) {
  const queryClient = useQueryClient()
  const { teams } = useTeamData()
  // The standing supervisor's scope is dynamic and owner-authoritative. Keep
  // it visible here without persisting effort rows as fake Prompt Manager
  // teams. Other teams avoid the global board read entirely.
  const effort = useEffortObservations('', team.id === 'effort-supervision' ? undefined : [])
  const chart = useQuery({
    queryKey: ['team-org-chart', team.id],
    queryFn: () => orgChartService.getOrgChart(team.id),
    staleTime: 15_000,
  })
  const allCharts = useQueries({
    queries: teams.filter((candidate) => candidate.id !== team.id).map((candidate) => ({
      queryKey: ['team-org-chart', candidate.id],
      queryFn: () => orgChartService.getOrgChart(candidate.id),
      staleTime: 15_000,
    })),
  })
  const [selectedTeamId, setSelectedTeamId] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const outgoing = chart.data?.managedTeamEdges ?? []
  const outgoingIds = new Set(outgoing.map((edge) => edge.managedTeamId))
  const teamById = useMemo(() => new Map(teams.map((candidate) => [candidate.id, candidate])), [teams])
  const incoming = useMemo(() => {
    const edges: ManagedTeamEdge[] = []
    allCharts.forEach((query) => (query.data?.managedTeamEdges ?? []).forEach((edge) => {
      if (edge.managedTeamId === team.id) edges.push(edge)
    }))
    return edges
  }, [allCharts, team.id])
  const available = teams.filter((candidate) => candidate.id !== team.id && !outgoingIds.has(candidate.id) && !candidate.archived)
  const discoveredEfforts = useMemo(() => effort.data?.boards.flatMap((board) => board.rows.map((row) => ({
    id: row.enrollment?.effortRef ?? '',
    name: row.enrollment?.displayName || row.enrollment?.effortRef || 'Unnamed effort',
    state: row.runtimeState || row.outcomeStanding?.state || 'observed',
  })).filter((row) => row.id)) ?? [], [effort.data])

  async function save(edges: ManagedTeamEdge[]) {
    setSaving(true)
    setError(null)
    try {
      await orgChartService.setManagedTeamEdges(team.id, edges)
      await queryClient.invalidateQueries({ queryKey: ['team-org-chart'] })
      await queryClient.invalidateQueries({ queryKey: ['team', team.id] })
      setSelectedTeamId('')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to update managed teams')
    } finally {
      setSaving(false)
    }
  }

  const label = (id: string) => teamById.get(id)?.displayName ?? id
  return (
    <section className="border-b border-border bg-violet-500/[.03] px-3 py-3" data-testid="managed-teams-panel" aria-label="Managed teams">
      <div className="flex items-center gap-2">
        <Network className="h-4 w-4 text-violet-600 dark:text-violet-400" />
        <h3 className="text-xs font-semibold uppercase tracking-wide">Managed teams</h3>
        <span className="text-[11px] text-muted-foreground">Team relationships, separate from members and runtime workers</span>
      </div>
      <div className="mt-2 grid gap-2 sm:grid-cols-2">
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">This team supervises</p>
          <ul className="mt-1 space-y-1">
            {outgoing.map((edge) => (
              <li key={edge.managedTeamId} className="flex items-center justify-between rounded-md border border-violet-500/20 bg-background/70 px-2 py-1 text-xs">
                <span className="truncate" title={edge.managedTeamId}>{label(edge.managedTeamId)}</span>
                <button type="button" disabled={saving} onClick={() => void save(outgoing.filter((candidate) => candidate.managedTeamId !== edge.managedTeamId))} className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-destructive" aria-label={`Remove ${label(edge.managedTeamId)} from managed teams`}><X className="h-3 w-3" /></button>
              </li>
            ))}
            {outgoing.length === 0 && <li className="text-xs text-muted-foreground">None configured</li>}
          </ul>
          <div className="mt-2 flex gap-1">
            <select value={selectedTeamId} onChange={(event) => setSelectedTeamId(event.target.value)} disabled={saving || available.length === 0} className="min-w-0 flex-1 rounded-md border bg-background px-2 py-1 text-xs" aria-label="Select team to supervise">
              <option value="">Add a team…</option>
              {available.map((candidate: Team) => <option key={candidate.id} value={candidate.id}>{candidate.displayName}</option>)}
            </select>
            <button type="button" disabled={saving || !selectedTeamId} onClick={() => void save([...outgoing, { managerTeamId: team.id, managedTeamId: selectedTeamId, relationship: 'supervises', status: 'active' }])} className="rounded-md border px-2 text-xs hover:bg-muted disabled:opacity-50" aria-label="Add managed team"><Plus className="h-3 w-3" /></button>
          </div>
        </div>
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">This team is supervised by</p>
          <ul className="mt-1 space-y-1">
            {incoming.map((edge) => <li key={edge.managerTeamId} className="rounded-md border border-sky-500/20 bg-background/70 px-2 py-1 text-xs" title={edge.managerTeamId}>{label(edge.managerTeamId)}</li>)}
            {incoming.length === 0 && <li className="text-xs text-muted-foreground">None configured</li>}
          </ul>
        </div>
      </div>
      {team.id === 'effort-supervision' && (
        <div className="mt-3 border-t border-border/70 pt-3" data-testid="discovered-efforts">
          <div className="flex items-baseline justify-between gap-2">
            <p className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">Automatically discovered orchestration efforts</p>
            <span className="text-[11px] text-muted-foreground">Agent Manager owner board</span>
          </div>
          <ul className="mt-1 grid gap-1 sm:grid-cols-2">
            {discoveredEfforts.map((item) => <li key={item.id} className="flex items-center justify-between gap-2 rounded-md border border-amber-500/20 bg-background/70 px-2 py-1 text-xs"><span className="min-w-0 truncate" title={item.id}>{item.name}</span><span className="shrink-0 text-[10px] text-muted-foreground">{item.state}</span></li>)}
          </ul>
          {discoveredEfforts.length === 0 && <p className="mt-1 text-xs text-muted-foreground">No current efforts returned by the owner board{effort.isError ? ' (owner read unavailable)' : ''}.</p>}
        </div>
      )}
      {error && <p role="alert" className="mt-2 text-xs text-destructive">{error}</p>}
    </section>
  )
}
