import { useProgress } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import { useEffect, useRef } from 'react'

interface FrameWorldStore {
  getState(): { actors: Record<string, { speed: number }> }
  subscribe(listener: () => void): () => void
}

interface FrameDriverProps {
  store: FrameWorldStore
  weatherActive: boolean
  diagnosticsOpen: boolean
  continuous: boolean
  intro: boolean
  settleSeconds: number
}

/** Owns demand-render invalidation for simulation, motion, input and async assets. */
export function FrameDriver({ store, weatherActive, diagnosticsOpen, continuous, intro, settleSeconds }: FrameDriverProps) {
  const invalidate = useThree((state) => state.invalidate)
  const { active, progress } = useProgress()
  const activeUntil = useRef(performance.now() + (intro ? 5000 : 0))

  useEffect(() => store.subscribe(invalidate), [invalidate, store])
  useEffect(() => { invalidate() }, [active, invalidate, progress])

  useEffect(() => {
    let raf = 0
    let heartbeat = 0
    const settleMs = Math.max(250, settleSeconds * 1000)
    const requestSettle = () => {
      activeUntil.current = performance.now() + settleMs
      invalidate()
    }
    const animate = () => {
      raf = 0
      const moving = Object.values(store.getState().actors).some((actor) => actor.speed > 0.001)
      if (continuous || weatherActive || moving || performance.now() < activeUntil.current) {
        invalidate()
        raf = requestAnimationFrame(animate)
      }
    }
    const wake = () => {
      requestSettle()
      if (!raf) raf = requestAnimationFrame(animate)
    }
    const offStore = store.subscribe(wake)
    window.addEventListener('pointerdown', wake, { passive: true })
    window.addEventListener('pointermove', wake, { passive: true })
    window.addEventListener('wheel', wake, { passive: true })
    window.addEventListener('keydown', wake)
    if (diagnosticsOpen) heartbeat = window.setInterval(invalidate, 250)
    wake()
    return () => {
      offStore()
      cancelAnimationFrame(raf)
      window.clearInterval(heartbeat)
      window.removeEventListener('pointerdown', wake)
      window.removeEventListener('pointermove', wake)
      window.removeEventListener('wheel', wake)
      window.removeEventListener('keydown', wake)
    }
  }, [continuous, diagnosticsOpen, invalidate, settleSeconds, store, weatherActive])

  return null
}
