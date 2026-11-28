import type { WeatherTuning } from '../../config'
import type { WorldState } from '../model'

export function weatherPressure(state: WorldState, tuning: WeatherTuning): number {
  const recent = state.events.filter((event) => state.time - event.at <= tuning.pressure.eventWindowSeconds)
  const outcomes = recent.filter((event) => event.kind === 'run.finished' || event.kind === 'run.failed')
  const failures = outcomes.filter((event) => event.kind === 'run.failed').length
  const failureShare = outcomes.length === 0 ? 0 : failures / outcomes.length
  const actors = state.actorOrder.map((id) => state.actors[id]).filter(Boolean)
  const failedShare = actors.length === 0 ? 0 : actors.filter((actor) => actor?.state === 'failed').length / actors.length
  const gatherings = Object.values(state.gatherings)
  const expiredShare = gatherings.length === 0 ? 0 : gatherings.filter((gathering) => state.time >= gathering.until).length / gatherings.length
  const weights = tuning.pressure
  const total = weights.recentFailureWeight + weights.failedActorWeight + weights.expiredGatheringWeight
  if (total === 0) return 0
  return Math.max(0, Math.min(1, (failureShare * weights.recentFailureWeight + failedShare * weights.failedActorWeight + expiredShare * weights.expiredGatheringWeight) / total))
}

export function smoothPressure(previous: number, target: number, dt: number, tuning: WeatherTuning): number {
  const alpha = Math.min(1, dt / tuning.pressureSmoothingSeconds)
  return previous + (target - previous) * alpha
}
