import { Color } from 'three'
import type { ActorTuning } from '../../config'
import type { EmoteKind } from '../../sim'

/** Pure colour construction kept separate from React component exports. */
export function accessoryColors(settings: ActorTuning['extras']) {
  const glow = (value: { color: string; intensity: number }) => new Color(value.color).multiplyScalar(value.intensity)
  const emotes = Object.fromEntries(Object.entries(settings.emotes).map(([kind, value]) => [kind, glow(value)])) as Record<EmoteKind, Color>
  return { failed: glow(settings.failed), gathered: glow(settings.gathered), working: glow(settings.working), off: new Color(settings.offColor), emotes }
}
