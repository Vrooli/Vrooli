/**
 * Hook for managing avatar behavior and animations.
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import type { AvatarState } from '@/types/world'
import {
  AvatarStateMachine,
  calculateLookRotation,
  calculateIdleSway,
  calculateWaveAnimation,
  calculateCelebrationAnimation,
  easing,
} from '@/services/avatarService'

interface UseAvatarBehaviorOptions {
  position?: [number, number, number]
  selectedNodeCount?: number
  onAnimationComplete?: () => void
}

export function useAvatarBehavior(options: UseAvatarBehaviorOptions = {}) {
  const { position = [0, 0, 0], selectedNodeCount = 0, onAnimationComplete } = options

  // State machine
  const stateMachineRef = useRef<AvatarStateMachine>(new AvatarStateMachine())
  const [currentState, setCurrentState] = useState<AvatarState>('idle')

  // Cursor tracking
  const [cursorPosition, setCursorPosition] = useState<{ x: number; y: number } | null>(null)

  // Animation time
  const animationTimeRef = useRef(0)
  const lastFrameTimeRef = useRef(Date.now())

  // Computed values
  const [lookRotation, setLookRotation] = useState<[number, number]>([0, 0])
  const [idleSway, setIdleSway] = useState({
    positionOffset: [0, 0, 0] as [number, number, number],
    rotationOffset: [0, 0, 0] as [number, number, number],
  })
  const [waveProgress, setWaveProgress] = useState(0)
  const [celebrationProgress, setCelebrationProgress] = useState(0)

  // Subscribe to state machine changes
  useEffect(() => {
    const unsubscribe = stateMachineRef.current.subscribe(setCurrentState)
    return unsubscribe
  }, [])

  // Handle cursor movement
  const handleCursorMove = useCallback(
    (x: number, y: number) => {
      setCursorPosition({ x, y })

      // Transition to looking state if idle
      if (stateMachineRef.current.getState() === 'idle') {
        stateMachineRef.current.transition('looking')
      }
    },
    []
  )

  // Handle cursor leave
  const handleCursorLeave = useCallback(() => {
    setCursorPosition(null)

    // Return to idle after a delay
    if (stateMachineRef.current.getState() === 'looking') {
      setTimeout(() => {
        stateMachineRef.current.transition('idle')
      }, 500)
    }
  }, [])

  // Trigger wave animation
  const wave = useCallback(() => {
    if (stateMachineRef.current.transition('waving')) {
      setWaveProgress(0)
    }
  }, [])

  // Trigger celebration animation
  const celebrate = useCallback(() => {
    stateMachineRef.current.forceTransition('celebrating')
    setCelebrationProgress(0)
  }, [])

  // React to selection changes
  useEffect(() => {
    if (selectedNodeCount > 0 && selectedNodeCount < 3) {
      // Wave for small selections
      wave()
    } else if (selectedNodeCount >= 3) {
      // Celebrate for multiple selections
      celebrate()
    }
  }, [selectedNodeCount, wave, celebrate])

  // Animation frame update
  useEffect(() => {
    let animationFrameId: number

    const animate = () => {
      const now = Date.now()
      const deltaTime = (now - lastFrameTimeRef.current) / 1000
      lastFrameTimeRef.current = now
      animationTimeRef.current += deltaTime

      const state = stateMachineRef.current.getState()
      const stateTime = stateMachineRef.current.getStateTime()

      // Update look rotation
      if (cursorPosition) {
        const newRotation = calculateLookRotation(position, cursorPosition)
        setLookRotation((prev) => [
          prev[0] + (newRotation[0] - prev[0]) * 0.1,
          prev[1] + (newRotation[1] - prev[1]) * 0.1,
        ])
      } else {
        setLookRotation((prev) => [prev[0] * 0.95, prev[1] * 0.95])
      }

      // Update idle sway
      setIdleSway(calculateIdleSway(animationTimeRef.current))

      // Update wave animation
      if (state === 'waving') {
        const duration = 1500
        const progress = Math.min(stateTime / duration, 1)
        setWaveProgress(easing.easeInOut(progress))

        if (progress >= 1) {
          stateMachineRef.current.forceTransition('idle')
          onAnimationComplete?.()
        }
      }

      // Update celebration animation
      if (state === 'celebrating') {
        const duration = 2000
        const progress = Math.min(stateTime / duration, 1)
        setCelebrationProgress(easing.bounce(progress))

        if (progress >= 1) {
          stateMachineRef.current.forceTransition('idle')
          onAnimationComplete?.()
        }
      }

      animationFrameId = requestAnimationFrame(animate)
    }

    animationFrameId = requestAnimationFrame(animate)
    return () => cancelAnimationFrame(animationFrameId)
  }, [position, cursorPosition, onAnimationComplete])

  // Compute derived animation values
  const waveAnimation = calculateWaveAnimation(waveProgress)
  const celebrationAnimation = calculateCelebrationAnimation(celebrationProgress)

  return {
    // State
    currentState,
    isAnimating: currentState !== 'idle' && currentState !== 'looking',

    // Cursor tracking
    cursorPosition,
    handleCursorMove,
    handleCursorLeave,
    lookRotation,

    // Animations
    idleSway,
    waveAnimation,
    waveProgress,
    celebrationAnimation,
    celebrationProgress,

    // Actions
    wave,
    celebrate,

    // Direct state control
    setState: (state: AvatarState) => stateMachineRef.current.forceTransition(state),
  }
}
