/**
 * A deterministic roster for goldens, smoke runs and demos: `?actors=N`
 * builds N agents across teams of `perTeam` with stable ids and colours.
 */
import type { AgentInput, TeamInput } from '../sim'
import { Rng, hashString } from '../sim/rng'

const PALETTE = ['#6366f1', '#f97316', '#10b981', '#ec4899', '#0ea5e9', '#eab308', '#8b5cf6', '#14b8a6']
const HEADS = ['#818cf8', '#fb923c', '#34d399', '#f472b6', '#38bdf8', '#facc15', '#a78bfa', '#2dd4bf']

export function syntheticRoster(actors: number, perTeam = 5, seed = 1): { teams: TeamInput[]; agents: AgentInput[] } {
  const rng = new Rng(hashString(`roster:${seed}`))
  const teams: TeamInput[] = []
  const agents: AgentInput[] = []
  const teamCount = Math.max(1, Math.ceil(actors / perTeam))
  for (let t = 0; t < teamCount; t += 1) {
    const memberIds: string[] = []
    for (let m = 0; m < perTeam && agents.length < actors; m += 1) {
      const id = `demo-${t}-${m}`
      const colour = (t + m) % PALETTE.length
      memberIds.push(id)
      agents.push({
        id,
        name: `Agent ${String.fromCharCode(65 + (t % 26))}${m + 1}`,
        colors: { body: PALETTE[colour] ?? '#6366f1', head: HEADS[colour] ?? '#818cf8', accent: '#fbbf24' },
        skillCount: rng.int(30),
      })
    }
    teams.push({ id: `demo-team-${t}`, name: `Team ${String.fromCharCode(65 + (t % 26))}`, memberIds })
  }
  return { teams, agents }
}
