import type { CameraTuning } from '../../config'
import { freezeCamera } from './pose'
import { applyInputMap, normalizeWheelUnits, type WorldCameraControls } from './input'
import { recordInteractionInput } from '../diagnostics/store'

import { NAV_MOTION, type NavigationPreferences, type NavigationTool } from '../../config/navigation'
export { DEFAULT_NAVIGATION, parseNavigationPreferences } from '../../config/navigation'
export type { NavigationCommand, NavigationMode, NavigationPreferences, NavigationPreset, NavigationState, NavigationTool } from '../../config/navigation'

/** Pointer maps and wheel semantics have one owner; browser pinch never changes the lens. */
export function bindExploreInput(canvas: HTMLElement, c: WorldCameraControls, tuning: CameraTuning,
  preferences: NavigationPreferences, tool: NavigationTool, reducedMotion: boolean, navigate: () => void): () => void {
  applyInputMap(c, { ...tuning.input, mouse: { ...tuning.input.mouse, left: tool === 'pan' ? 'truck' : tuning.input.mouse.left },
    touch: { ...tuning.input.touch, one: tool === 'pan' ? 'truck' : tuning.input.touch.one } })
  c.draggingSmoothTime = !reducedMotion && preferences.smoothing ? NAV_MOTION.dragSmoothing : 0
  c.azimuthRotateSpeed = preferences.sensitivity
  c.polarRotateSpeed = preferences.sensitivity * (preferences.invertLook ? -1 : 1)
  c.truckSpeed = tuning.truckSpeed * preferences.sensitivity
  c.dollySpeed = tuning.dollySpeed * preferences.sensitivity
  const wheel = (event: WheelEvent) => {
    if (!c.enabled) return
    normalizeWheelUnits(event, canvas)
    event.preventDefault()
    event.stopImmediatePropagation()
    recordInteractionInput()
    navigate()
    const rect = canvas.getBoundingClientRect()
    const x = (event.clientX - rect.left) / Math.max(1, rect.width) * 2 - 1
    const y = 1 - (event.clientY - rect.top) / Math.max(1, rect.height) * 2
    if (preferences.device === 'trackpad' && !event.ctrlKey && !event.metaKey) c.panPixels(event.deltaX, event.deltaY)
    else c.dollyPixels(event.deltaY * (preferences.invertZoom ? -1 : 1), x, y)
  }
  let dragging = false
  const down = () => { if (c.enabled) { dragging = true; canvas.focus({ preventScroll: true }); navigate() } }
  // Endpoints must not keep moving after a drag or a focus loss.
  const stop = () => { if (c.enabled && dragging) freezeCamera(c); dragging = false }
  const blur = () => { if (c.enabled) freezeCamera(c); dragging = false }
  const visibility = () => { if (document.hidden) blur() }
  canvas.addEventListener('wheel', wheel, { capture: true, passive: false })
  canvas.addEventListener('pointerdown', down, true)
  window.addEventListener('pointerup', stop)
  canvas.addEventListener('pointercancel', stop)
  window.addEventListener('blur', blur)
  document.addEventListener('visibilitychange', visibility)
  return () => {
    canvas.removeEventListener('wheel', wheel, true)
    canvas.removeEventListener('pointerdown', down, true)
    window.removeEventListener('pointerup', stop)
    canvas.removeEventListener('pointercancel', stop)
    window.removeEventListener('blur', blur)
    document.removeEventListener('visibilitychange', visibility)
  }
}
