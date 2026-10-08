import { Color } from 'three'
import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { accessoryColors } from './accessoryColors'

describe('accessory colours', () => {
  it('preserves the former status and emote radiance defaults', () => {
    const colors = accessoryColors(tuning.actor.extras)
    expect(colors.failed).toEqual(new Color('#ff3b3b').multiplyScalar(3))
    expect(colors.gathered).toEqual(new Color('#ffb020').multiplyScalar(2.2))
    expect(colors.working).toEqual(new Color('#59d0ff').multiplyScalar(2.6))
    expect(colors.off).toEqual(new Color('#000000'))
    for (const [kind, color, intensity] of [
      ['start', '#59d0ff', 2], ['done', '#5ce27a', 2],
      ['fail', '#ff3b3b', 2.5], ['message', '#ffffff', 1.8], ['gather', '#ffb020', 2],
    ] as const) expect(colors.emotes[kind]).toEqual(new Color(color).multiplyScalar(intensity))
    expect(tuning.actor.extras.tierSizes).toEqual([0, 0.45, 0.65, 0.85, 1.1])
    expect(tuning.actor.extras.tierColors).toEqual(['#000000', '#f5f0e6', '#e0b35a', '#6b4a2b', '#3b6ea5'])
  })

  it('uses overridden colours and intensity without mutating configuration or sharing mutable colours', () => {
    const settings = structuredClone(tuning.actor.extras)
    settings.failed = { color: '#123456', intensity: 4 }
    settings.emotes.message = { color: '#abcdef', intensity: 0.7 }
    const first = accessoryColors(settings)
    const second = accessoryColors(settings)
    expect(first.failed).toEqual(new Color('#123456').multiplyScalar(4))
    expect(first.emotes.message).toEqual(new Color('#abcdef').multiplyScalar(0.7))
    first.failed.set('#ffffff')
    expect(second.failed).toEqual(new Color('#123456').multiplyScalar(4))
    expect(settings.failed.color).toBe('#123456')
  })
})
