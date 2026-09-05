import { Color, type Scene, type WebGLRenderer } from 'three'
import type { LightingPeriod } from '../../config'

/** Apply period state without taking ownership of the outdoor environment. */
export function applyPeriodBackground(
  scene: Scene,
  renderer: Pick<WebGLRenderer, 'toneMappingExposure'>,
  outdoor: boolean,
  period: Pick<LightingPeriod, 'backgroundColor' | 'exposure'>,
) {
  // Environment owns the outdoor cube map. A passive effect must not replace
  // the texture installed by its layout effect with a flat color.
  if (!outdoor) scene.background = new Color(period.backgroundColor)
  renderer.toneMappingExposure = period.exposure
}
