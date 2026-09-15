import type { ActorTuning } from '../../config'
import { wrapAngle } from '../../sim/motion/move'

/** Presentation-only yaw: simulation seat and walk headings remain authoritative. */
export function cameraFacing(simFacing: number, cameraAzimuth: number, weight: number, maxYaw: number): number {
  if (weight <= 0) return simFacing
  const limit = Math.max(0, Math.min(Math.PI, maxYaw))
  const delta = Math.max(-limit, Math.min(limit, wrapAngle(cameraAzimuth - simFacing)))
  return wrapAngle(simFacing + delta * Math.min(1, weight))
}

export function facingWeight(previous: number, focused: boolean, speed: number, dt: number, tuning: ActorTuning['facing']): number {
  // Walking must use its true heading, even if focus was just released.
  if (speed > tuning.restSpeed) return 0
  const step = Math.max(0, dt) / tuning.blendSeconds
  return focused ? Math.min(1, previous + step) : Math.max(0, previous - step)
}
