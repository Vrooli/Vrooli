import { useEffect, useRef, type RefObject } from 'react'
import { WALK, type NavigationMode, type NavigationPreferences } from '../../../config/navigation'
import { normalizeWheelUnits, ownsKeyboardInput } from '../input'
import type { Walker } from '../walking'

interface Options {
  mode: NavigationMode
  canvas: HTMLCanvasElement
  walker: RefObject<Walker | null>
  keys: RefObject<Set<string>>
  navigation: NavigationPreferences
  onExit: () => void
  onLock: (locked: boolean) => void
  onError: (message: string) => void
  invalidate: () => void
}

/** Own pointer capture, pointer lock and dismissal as one disposable walking input scope. */
export function useWalkingInput(options: Options) {
  const latest = useRef(options)
  latest.current = options
  const { canvas, mode } = options
  useEffect(() => {
    if (mode === 'explore') return
    let dragging: number | null = null
    let travel = 0
    const down = (event: PointerEvent) => {
      if (event.button !== 0) return
      canvas.focus({ preventScroll: true })
      travel = 0
      dragging = event.pointerId
      canvas.setPointerCapture(event.pointerId)
    }
    const move = (event: PointerEvent) => {
      if (document.pointerLockElement !== canvas && dragging !== event.pointerId) return
      travel += Math.hypot(event.movementX || 0, event.movementY || 0)
      const { walker, navigation, invalidate } = latest.current
      walker.current?.look(event.movementX, event.movementY, navigation.sensitivity, navigation.invertLook)
      event.stopPropagation()
      invalidate()
    }
    const click = (event: MouseEvent) => { if (travel > 4) event.stopImmediatePropagation() }
    const up = () => { dragging = null }
    const clear = () => { latest.current.keys.current.clear(); dragging = null }
    const lock = () => { clear(); latest.current.onLock(document.pointerLockElement === canvas) }
    const lockError = () => latest.current.onError('Mouse capture unavailable. Drag the world to look around.')
    const escape = (event: KeyboardEvent) => {
      if (event.code === 'Space' && document.activeElement === canvas && !event.defaultPrevented && !ownsKeyboardInput(event.target)) {
        event.preventDefault()
        event.stopPropagation()
        if (!event.repeat) latest.current.walker.current?.jump()
        latest.current.invalidate()
        return
      }
      if (event.key !== 'Escape' || event.defaultPrevented || ownsKeyboardInput(event.target)) return
      event.preventDefault()
      event.stopPropagation()
      clear()
      if (document.pointerLockElement === canvas) document.exitPointerLock()
      else latest.current.onExit()
    }
    const wheel = (event: WheelEvent) => {
      event.preventDefault(); event.stopImmediatePropagation()
      normalizeWheelUnits(event, canvas)
      const body = latest.current.walker.current
      if (body && latest.current.mode === 'third-person') body.boom = Math.max(WALK.minBoom, Math.min(WALK.maxBoom, body.boom + event.deltaY * WALK.boomMetresPerPixel))
    }
    canvas.addEventListener('pointerdown', down, true)
    canvas.addEventListener('pointermove', move, true)
    canvas.addEventListener('click', click, true)
    canvas.addEventListener('pointerup', up)
    canvas.addEventListener('pointercancel', clear)
    canvas.addEventListener('wheel', wheel, { capture: true, passive: false })
    document.addEventListener('pointerlockchange', lock)
    document.addEventListener('pointerlockerror', lockError)
    document.addEventListener('visibilitychange', clear)
    window.addEventListener('blur', clear)
    window.addEventListener('keydown', escape, true)
    return () => {
      if (document.pointerLockElement === canvas) document.exitPointerLock()
      canvas.removeEventListener('pointerdown', down, true)
      canvas.removeEventListener('pointermove', move, true)
      canvas.removeEventListener('click', click, true)
      canvas.removeEventListener('pointerup', up)
      canvas.removeEventListener('pointercancel', clear)
      canvas.removeEventListener('wheel', wheel, true)
      document.removeEventListener('pointerlockchange', lock)
      document.removeEventListener('pointerlockerror', lockError)
      document.removeEventListener('visibilitychange', clear)
      window.removeEventListener('blur', clear)
      window.removeEventListener('keydown', escape, true)
      clear()
      latest.current.onLock(false)
    }
  }, [mode, canvas])
}
