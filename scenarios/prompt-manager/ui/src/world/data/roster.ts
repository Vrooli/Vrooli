/**
 * The team graph as the sim wants it: teams with member ids and agents with
 * colours and skill counts. Sourced from the existing team and agent hooks.
 */
import { useQueries, useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { useAgentData } from '@/hooks/useAgentData'
import { useTeamData } from '@/hooks/useTeamData'
import { api } from '@/lib/api'
import type { Agent, GraphResponse } from '@/lib/schemas'
import type { AgentInput, TeamInput } from '../sim'
import { contractorMembersFromObservations, useEffortObservations } from '@/services/effortService'

export interface WorldRoster {
  teams: TeamInput[]
  agents: AgentInput[]
  ready: boolean
}

/** Skills per agent from the graph: edges from an agent node to skill nodes. */
export function skillCounts(graph: GraphResponse | undefined): Record<string, number> {
  const counts: Record<string, number> = {}
  if (!graph) return counts
  const { nodes, edges } = graph.graph
  const skills = new Set(nodes.filter((n) => n.type === 'skill').map((n) => n.id))
  const agents = new Map(nodes.filter((n) => n.type === 'agent').map((n) => [n.id, n]))
  for (const edge of edges) {
    if (agents.has(edge.from) && skills.has(edge.to)) counts[edge.from] = (counts[edge.from] ?? 0) + 1
  }
  return counts
}

export function agentToInput(agent: Agent, skillCount = 0): AgentInput {
  return {
    id: agent.id,
    name: agent.displayName,
    colors: agent.appearance ? { body: agent.appearance.body, head: agent.appearance.head, accent: agent.appearance.accent } : undefined,
    skillCount,
  }
}

export function useWorldRoster(): WorldRoster {
  const { teams, isLoading: teamsLoading } = useTeamData()
  const { agents, isLoading: agentsLoading } = useAgentData()
  const details = useQueries({
    queries: teams.map((team) => ({
      queryKey: ['team', team.id],
      queryFn: () => api.getTeam(team.id),
      staleTime: 60_000,
    })),
  })
  const graph = useQuery({ queryKey: ['graph'], queryFn: () => api.getGraph(), staleTime: 60_000 })
  const effortRefs = useMemo(() => teams.flatMap((team) => team.effortRefs ?? []), [teams])
  const effort = useEffortObservations('', effortRefs)
  const membersReady = details.every((q) => q.data !== undefined || q.isError)
  return useMemo(() => {
    const counts = skillCounts(graph.data)
    const teamInputs: TeamInput[] = teams.map((team, i) => ({
      id: team.id,
      name: team.displayName,
      memberIds: (details[i]?.data?.members ?? []).map((m) => m.agentId),
      managedTeamIds: team.managedTeamIds ?? [],
    }))
    const contractors = contractorMembersFromObservations(effort.data ?? { boards: [], unavailable: [] }, effortRefs)
    const contractorTeamByEffort = new Map<string, string>()
    teams.forEach((team) => (team.effortRefs ?? []).forEach((ref) => contractorTeamByEffort.set(ref, team.id)))
    const contractorTeams = new Map<string, string[]>()
    contractors.forEach((contractor) => {
      const teamId = contractorTeamByEffort.get(contractor.effortRef)
      if (!teamId) return
      const ids = contractorTeams.get(teamId) ?? []
      ids.push(contractor.id)
      contractorTeams.set(teamId, ids)
    })
    return {
      teams: teamInputs.map((team) => ({ ...team, memberIds: [...team.memberIds, ...(contractorTeams.get(team.id) ?? [])] })),
      agents: [
        ...agents.map((agent) => agentToInput(agent, counts[agent.id] ?? counts[`agent:${agent.id}`] ?? 0)),
        ...contractors.map((contractor) => ({ id: contractor.id, name: contractor.displayName, memberType: 'contractor' as const })),
      ],
      ready: !teamsLoading && !agentsLoading && membersReady && (!effortRefs.length || effort.isFetched),
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [teams, agents, teamsLoading, agentsLoading, membersReady, graph.data, effort.data, effort.isFetched, effortRefs.join('|'), details.map((d) => d.dataUpdatedAt).join(',')])
}
