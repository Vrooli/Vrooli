import { Color, type Scene, type WebGLRenderer } from 'three'
import type { LightingPeriod } from '../../config'

/** Set the fallback sky color and exposure; the HDRI supplies reflections only. */
export function applyPeriodBackground(
  scene: Scene,
  renderer: Pick<WebGLRenderer, 'toneMappingExposure'>,
  _outdoor: boolean,
  period: Pick<LightingPeriod, 'backgroundColor' | 'exposure'>,
) {
  scene.background = new Color(period.backgroundColor)
  renderer.toneMappingExposure = period.exposure
}
