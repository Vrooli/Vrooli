import { useProgress } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import { useEffect, useRef } from 'react'
import type { QualityTuning } from '../config'
import { recordInteractionInput } from './diagnostics/store'
import { ownsKeyboardInput } from './camera/input'
import type { AnimationLeases } from './animationLeases'

interface FrameWorldStore {
  getState(): { actors: Record<string, { speed: number }> }
  subscribe(listener: () => void): () => void
}

interface FrameDriverProps {
  settings: QualityTuning['frameDriver']
  store: FrameWorldStore
  weatherActive: boolean
  diagnosticsOpen: boolean
  continuous: boolean
  intro: boolean
  settleSeconds: number
  animationLeases?: AnimationLeases
}

/** Owns demand-render invalidation for simulation, motion, input and async assets. */
export function FrameDriver({ settings, store, weatherActive, diagnosticsOpen, continuous, intro, settleSeconds, animationLeases }: FrameDriverProps) {
  const invalidate = useThree((state) => state.invalidate)
  const canvas = useThree((state) => state.gl.domElement)
  const { active, progress } = useProgress()
  const activeUntil = useRef(performance.now() + (intro ? settings.introMs : 0))

  useEffect(() => store.subscribe(invalidate), [invalidate, store])
  useEffect(() => { invalidate() }, [active, invalidate, progress])

  useEffect(() => {
    let raf = 0
    let heartbeat = 0
    const settleMs = Math.max(settings.minimumSettleMs, settleSeconds * 1000)
    const requestSettle = () => {
      activeUntil.current = Math.max(activeUntil.current, performance.now() + settleMs)
      invalidate()
    }
    const animate = () => {
      raf = 0
      if (document.hidden) return
      const moving = Object.values(store.getState().actors).some((actor) => actor.speed > settings.movingSpeed)
      if (continuous || weatherActive || moving || (animationLeases?.count ?? 0) > 0 || performance.now() < activeUntil.current) {
        invalidate()
        raf = requestAnimationFrame(animate)
      }
    }
    const wake = () => {
      if (document.hidden) return
      requestSettle()
      if (!raf) raf = requestAnimationFrame(animate)
    }
    const offStore = store.subscribe(wake)
    // Lease changes need one final frame to remove expired effects, but must not
    // extend the input settle window when the last decorative owner releases.
    const offLeases = animationLeases?.subscribe(() => {
      invalidate()
      if (!raf) raf = requestAnimationFrame(animate)
    })
    const inputWake = () => { recordInteractionInput(); wake() }
    const keyWake = (event: KeyboardEvent) => {
      if (!ownsKeyboardInput(event.target) && !event.ctrlKey && !event.metaKey && !event.altKey && ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', '=', '+', '-', 'Escape'].includes(event.key)) inputWake()
    }
    document.addEventListener('visibilitychange', wake)
    canvas.addEventListener('pointerdown', inputWake, { passive: true })
    canvas.addEventListener('pointermove', inputWake, { passive: true })
    canvas.addEventListener('wheel', inputWake, { passive: true })
    window.addEventListener('keydown', keyWake)
    if (diagnosticsOpen) heartbeat = window.setInterval(invalidate, settings.diagnosticsHeartbeatMs)
    wake()
    return () => {
      offStore()
      offLeases?.()
      cancelAnimationFrame(raf)
      window.clearInterval(heartbeat)
      canvas.removeEventListener('pointerdown', inputWake)
      canvas.removeEventListener('pointermove', inputWake)
      canvas.removeEventListener('wheel', inputWake)
      window.removeEventListener('keydown', keyWake)
      document.removeEventListener('visibilitychange', wake)
    }
  }, [animationLeases, canvas, continuous, diagnosticsOpen, invalidate, settleSeconds, settings, store, weatherActive])

  return null
}
