import type { ActorTuning, SimTuning } from '../../config'
import type { Actor, Vec2 } from '../model'

const TAU = Math.PI * 2
const HALF = 0.5

export function wrapAngle(angle: number): number {
  let a = angle % TAU
  if (a > Math.PI) a -= TAU
  if (a < -Math.PI) a += TAU
  return a
}

/** Turn `facing` toward `target` by at most `rate * dt`. */
export function turnToward(facing: number, target: number, rate: number, dt: number): number {
  const delta = wrapAngle(target - facing)
  const step = rate * dt
  if (Math.abs(delta) <= step) return wrapAngle(target)
  return wrapAngle(facing + Math.sign(delta) * step)
}

export function headingTo(from: Vec2, to: Vec2): number {
  return Math.atan2(to[0] - from[0], to[1] - from[1])
}

/**
 * Advance an actor along its path. Speed ramps toward the target speed over
 * `accelSeconds` (the "never snap" rule) and the actor turns at `turnRate`.
 * Returns true on the tick the actor reaches the final waypoint.
 */
export function moveAlongPath(actor: Actor, dt: number, sim: SimTuning): boolean {
  const target = actor.path[0]
  if (!target) {
    actor.speed = Math.max(0, actor.speed - (sim.walkSpeed / sim.accelSeconds) * dt)
    return false
  }
  const goalSpeed = actor.hurrying ? sim.hurrySpeed : sim.walkSpeed
  const accel = goalSpeed / sim.accelSeconds
  actor.speed = actor.speed < goalSpeed ? Math.min(goalSpeed, actor.speed + accel * dt) : Math.max(goalSpeed, actor.speed - accel * dt)
  const heading = headingTo(actor.position, target)
  actor.facing = turnToward(actor.facing, heading, sim.turnRateRadPerSec, dt)
  const dist = Math.hypot(target[0] - actor.position[0], target[1] - actor.position[1])
  const travel = actor.speed * dt
  if (travel >= dist || dist <= sim.arriveRadius * HALF) {
    actor.position = [target[0], target[1]]
    actor.path = actor.path.slice(1)
    if (actor.path.length === 0) {
      actor.speed = 0
      return true
    }
    return false
  }
  actor.position = [actor.position[0] + Math.sin(heading) * travel, actor.position[1] + Math.cos(heading) * travel]
  return false
}

/** Per-actor animation phase, updated from motion so the scene only reads it. */
export function updateAnimation(actor: Actor, dt: number, moving: boolean, tuning: ActorTuning, rngValue: () => number): void {
  const anim = actor.anim
  anim.breathPhase = (anim.breathPhase + dt * tuning.breathHz) % 1
  if (moving && !anim.seated) {
    const before = anim.hopPhase
    anim.hopPhase = (anim.hopPhase + dt * tuning.hopHz) % 1
    if (anim.hopPhase < before) anim.squash = tuning.squashOnLand
  } else {
    anim.hopPhase = 0
  }
  anim.squash = Math.min(1, anim.squash + (1 - anim.squash) * Math.min(1, tuning.squashRecoverPerSec * dt))
  anim.blinkTimer -= dt
  if (anim.blinkTimer <= 0) {
    if (anim.blinking) {
      anim.blinking = false
      anim.blinkTimer = tuning.blinkIntervalSeconds.min + (tuning.blinkIntervalSeconds.max - tuning.blinkIntervalSeconds.min) * rngValue()
    } else {
      anim.blinking = true
      anim.blinkTimer = tuning.blinkSeconds
    }
  }
  if (anim.emote) {
    anim.emote.remaining -= dt
    if (anim.emote.remaining <= 0) anim.emote = undefined
  }
}
