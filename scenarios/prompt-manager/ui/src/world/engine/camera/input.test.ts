import { describe, expect, it } from 'vitest'
import CameraControls from 'camera-controls'
import { tuning, type CameraTuning } from '../../config'
import { MouseActionSchema, SingleTouchActionSchema, MultiTouchActionSchema } from '../../config/tuning.schema'
import { applyInputMap } from './input'

function controls() {
  return { mouseButtons: { left: -1, middle: -1, right: -1, wheel: -1 }, touches: { one: -1, two: -1, three: -1 } }
}

describe('declared camera grammar', () => {
  it('assigns every button and finger slot, with semantic mouse/touch parity', () => {
    const c = controls()
    applyInputMap(c, tuning.camera.input)
    const A = CameraControls.ACTION
    expect(c.mouseButtons).toEqual({ left: A.ROTATE, middle: A.TRUCK, right: A.TRUCK, wheel: A.DOLLY })
    // Mouse and touch have different library constants for the same semantic action.
    expect(tuning.camera.input.touch.one).toBe(tuning.camera.input.mouse.left)
    expect(c.touches).toEqual({ one: A.TOUCH_ROTATE, two: A.TOUCH_DOLLY_TRUCK, three: A.TOUCH_TRUCK })
    expect(tuning.camera.input.touch.two.split('-')).toEqual([tuning.camera.input.mouse.wheel, tuning.camera.input.mouse.right])
    expect(tuning.camera.dollyToCursor).toBe(true)
  })
  it('maps every schema value to an actual library action', () => {
    const defined = new Set<number>(Object.values(CameraControls.ACTION))
    const c = controls()
    for (const value of MouseActionSchema.options) {
      applyInputMap(c, { ...tuning.camera.input, mouse: { left: value, middle: value, right: value, wheel: value } })
      for (const result of Object.values(c.mouseButtons)) expect(defined.has(result)).toBe(true)
    }
    for (const value of SingleTouchActionSchema.options) {
      applyInputMap(c, { ...tuning.camera.input, touch: { ...tuning.camera.input.touch, one: value } })
      expect(defined.has(c.touches.one)).toBe(true)
    }
    for (const value of MultiTouchActionSchema.options) {
      applyInputMap(c, { ...tuning.camera.input, touch: { ...tuning.camera.input.touch, two: value, three: value } })
      expect(defined.has(c.touches.two)).toBe(true)
      expect(defined.has(c.touches.three)).toBe(true)
    }
  })
  it('rejects unmapped runtime values atomically, including prototype property names', () => {
    const c = controls()
    const before = structuredClone(c)
    for (const bad of ['teleport', 'toString']) {
      const input = { ...tuning.camera.input, touch: { ...tuning.camera.input.touch, three: bad } } as CameraTuning['input']
      expect(() => applyInputMap(c, input)).toThrow('Unmapped')
      expect(c).toEqual(before)
    }
  })
})
