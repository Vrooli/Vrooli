import type { WorldState } from './model'

export function conversationReady(state: WorldState, id: string, walking: boolean): boolean {
  const actor = state.actors[id]
  if (!actor) return false
  if (!walking) return true
  const visit = state.visitorConversation
  if (visit?.agentId !== id || !visit.goal || !visit.path || visit.path.length > 0) return false
  return Math.hypot(actor.position[0] - visit.position[0], actor.position[1] - visit.position[1]) <= 3.5 && actor.speed < .1
}

