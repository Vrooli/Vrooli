import { describe, expect, it, vi } from 'vitest'
import { tuning } from '../../config'
import { world } from '../../sim/__tests__/fixtures'
import { groundSampler } from '../../sim'
import { bodyPose } from './pose'
import { createPoseBuffer, POSE, POSE_STRIDE, writePoseBuffer } from './PoseBuffer'

describe('PoseBuffer', () => {
  it('evaluates bodyPose once per actor and writes the shared stride', () => {
    const state = world(2, 5)
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
