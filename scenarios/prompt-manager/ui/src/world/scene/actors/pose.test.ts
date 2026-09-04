import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { createWorld } from '../../sim'
import { actorSeed, bodyOffset, bodyPose } from './pose'

function actor() {
  const state = createWorld({ seed: 1, now: 0, teams: [], agents: [{ id: 'a', name: 'A' }], scene: 'office' }, tuning)
  const a = state.actors.a
  if (!a) throw new Error('missing')
  return a
}

describe('bodyPose', () => {
  it('adds the sampled ground elevation once', () => {
    const a = actor()
    const flat = bodyPose(a, tuning.actor)
    const raised = bodyPose(a, tuning.actor, { heightAt: (x, z) => x * 0.25 + z * 0.5 })
    expect(raised.y - flat.y).toBeCloseTo(a.position[0] * 0.25 + a.position[1] * 0.5, 9)
  })
  it('rests on the ground with the squashed body and no hop', () => {
    const a = actor()
    a.anim.hopPhase = 0
    a.anim.breathPhase = 0
    const p = bodyPose(a, tuning.actor)
    expect(p.y).toBeCloseTo(tuning.actor.bodyRadius * tuning.actor.look.bodySquashY, 9)
    expect(p.scaleXZ).toBeCloseTo(tuning.actor.bodyRadius, 9)
  })

  it('hops in the middle of the hop phase and shrinks when seated', () => {
    const a = actor()
    a.anim.hopPhase = 0.5
    a.anim.breathPhase = 0
    expect(bodyPose(a, tuning.actor).y).toBeCloseTo(tuning.actor.hopHeight + tuning.actor.bodyRadius * tuning.actor.look.bodySquashY, 9)
    a.anim.hopPhase = 0
    a.anim.seated = true
    expect(bodyPose(a, tuning.actor).scaleXZ).toBeCloseTo(tuning.actor.bodyRadius * tuning.actor.seatedScale, 9)
  })

  it('offsets follow the facing direction', () => {
    const p = { x: 0, y: 1, z: 0, facing: 0, scaleXZ: 1, scaleY: 1 }
    expect(bodyOffset(p, 0, 0, 2).map((v) => Number(v.toFixed(6)))).toEqual([0, 1, 2])
    expect(bodyOffset({ ...p, facing: Math.PI / 2 }, 0, 0, 2).map((v) => Number(v.toFixed(6)))).toEqual([2, 1, 0])
    expect(bodyOffset(p, 1, 0, 0).map((v) => Number(v.toFixed(6)))).toEqual([1, 1, 0])
  })

  it('seeds are stable and in range', () => {
    expect(actorSeed('agent-1')).toBe(actorSeed('agent-1'))
    expect(actorSeed('agent-1')).not.toBe(actorSeed('agent-2'))
    expect(actorSeed('x')).toBeGreaterThanOrEqual(0)
    expect(actorSeed('x')).toBeLessThan(1)
  })
})
