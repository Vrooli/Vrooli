/**
 * Pure helpers turning sim actor state into render transforms. Kept out of
 * the components so they are testable in Node.
 */
import type { ActorTuning } from '../../config'
import type { Actor, GroundSampler } from '../../sim'

export interface BodyPose {
  x: number
  y: number
  z: number
  facing: number
  scaleXZ: number
  scaleY: number
}

const TAU = Math.PI * 2

/** Body centre and scale for one actor from its animation phase. */
export function bodyPose(actor: Actor, t: ActorTuning, ground: GroundSampler = { heightAt: () => 0 }): BodyPose {
  const hop = actor.anim.hopPhase > 0 ? Math.sin(actor.anim.hopPhase * Math.PI) * t.hopHeight : 0
  const breath = Math.sin(actor.anim.breathPhase * TAU) * t.breathAmplitude
  const seated = actor.anim.seated ? t.seatedScale : 1
  const radius = t.bodyRadius * seated
  const scaleXZ = radius * (1 + breath)
  const scaleY = radius * t.look.bodySquashY * (1 - breath)
  return {
    x: actor.position[0],
    y: ground.heightAt(actor.position[0], actor.position[1]) + hop + scaleY,
    z: actor.position[1],
    facing: actor.facing,
    scaleXZ,
    scaleY,
  }
}

/** Point in world space at a local offset (right, up, forward) from the body, following its facing. */
export function bodyOffset(pose: BodyPose, right: number, up: number, forward: number): [number, number, number] {
  const sin = Math.sin(pose.facing)
  const cos = Math.cos(pose.facing)
  // facing 0 looks along +z; right is +x rotated with it
  return [pose.x + right * cos + forward * sin, pose.y + up, pose.z - right * sin + forward * cos]
}

/** Stable per-actor seed in [0, 1) for shader variation. */
export function actorSeed(id: string): number {
  let h = 2166136261
  for (let i = 0; i < id.length; i += 1) h = Math.imul(h ^ id.charCodeAt(i), 16777619)
  return ((h >>> 0) % 10000) / 10000
}
