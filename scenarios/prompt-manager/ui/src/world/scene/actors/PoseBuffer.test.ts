import { describe, expect, it, vi } from 'vitest'
import { tuning } from '../../config'
import { makeWorld } from '../../sim/__tests__/fixtures'
import { groundSampler } from '../../sim'
import { bodyPose } from './pose'
import { createPoseBuffer, POSE, POSE_STRIDE, writePoseBuffer } from './PoseBuffer'

describe('PoseBuffer', () => {
  it('removes presentation-facing weight when the focused actor starts walking', () => {
    const state = makeWorld({ teams: 1, agents: 1 })
    const id = state.actorOrder[0]
    if (!id) throw new Error('fixture has no actor')
    const actor = state.actors[id]
    if (!actor) throw new Error('fixture has no focused actor')
    actor.speed = 0
    actor.facing = 0.3
    const buffer = createPoseBuffer(1)
    const presentation = { focusedId: id, cameraX: actor.position[0] + 10, cameraZ: actor.position[1], dt: tuning.actor.facing.blendSeconds }
    writePoseBuffer(buffer, state, tuning.actor, undefined, undefined, presentation)
    expect(buffer.facingWeights[0]).toBe(1)
    actor.speed = tuning.actor.facing.restSpeed + 0.1
    const before = JSON.stringify(state)
    const expected = bodyPose(actor, tuning.actor, groundSampler(state.terrain))
    writePoseBuffer(buffer, state, tuning.actor, undefined, undefined, presentation)
    expect(buffer.facingWeights[0]).toBe(0)
    expect(buffer.data[POSE.facing]).toBeCloseTo(actor.facing, 6)
    expect(buffer.data[POSE.x]).toBeCloseTo(expected.x, 5)
    expect(buffer.data[POSE.y]).toBeCloseTo(expected.y, 5)
    expect(buffer.data[POSE.z]).toBeCloseTo(expected.z, 5)
    expect(JSON.stringify(state)).toBe(before)
  })
  it('turns only the focused resting rendered actor without changing any simulation bytes', () => {
    const state = makeWorld({ teams: 2, agents: 10, treeVariants: 3 })
    const id = state.actorOrder[0]
    if (!id) throw new Error('fixture has no actor')
    const actor = state.actors[id]
    if (!actor) throw new Error('fixture has no focused actor')
    actor.speed = 0
    actor.facing = 0
    const before = JSON.stringify(state)
    const buffer = createPoseBuffer(state.actorOrder.length)
    for (let frame = 0; frame < 100; frame += 1) {
      writePoseBuffer(buffer, state, tuning.actor, undefined, undefined, {
        focusedId: id, cameraX: actor.position[0] + 10, cameraZ: actor.position[1], dt: 1 / 60,
      })
    }
    expect(buffer.data[POSE.facing]).toBeCloseTo(Math.PI / 2, 5)
    expect(buffer.facingWeights[0]).toBe(1)
    expect(JSON.stringify(state)).toBe(before)
    for (let index = 1; index < state.actorOrder.length; index += 1) expect(buffer.facingWeights[index]).toBe(0)
  })
  it('evaluates bodyPose once per actor and writes the shared stride', () => {
    const state = makeWorld({ teams: 2, agents: 10, treeVariants: 3 })
    const buffer = createPoseBuffer(state.actorOrder.length)
    const detail = vi.fn(() => true)
    const computePose = vi.fn(bodyPose)

    writePoseBuffer(buffer, state, tuning.actor, detail, computePose)

    expect(detail).toHaveBeenCalledTimes(state.actorOrder.length)
    expect(computePose).toHaveBeenCalledTimes(state.actorOrder.length)
    state.actorOrder.forEach((id, index) => {
      const actor = state.actors[id]
      if (!actor) throw new Error(`missing actor ${id}`)
      const expected = bodyPose(actor, tuning.actor, groundSampler(state.terrain))
      const offset = index * POSE_STRIDE
      const values = Array.from(buffer.data.slice(offset, offset + 6))
      const expectedValues = [
        expected.x,
        expected.y,
        expected.z,
        expected.facing,
        expected.scaleXZ,
        expected.scaleY,
      ]
      expectedValues.forEach((value, valueIndex) => expect(values[valueIndex]).toBeCloseTo(value, 5))
      expect(buffer.data[offset + POSE.visible]).toBe(1)
    })
  })
})
