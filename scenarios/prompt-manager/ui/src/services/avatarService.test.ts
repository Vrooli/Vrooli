/**
 * Tests for avatarService.ts
 *
 * Tests cover:
 * - Avatar state machine transitions
 * - Look rotation calculations
 * - Animation calculations (idle sway, wave, celebration)
 * - Easing functions
 * - Interpolation helpers
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  AvatarStateMachine,
  calculateLookRotation,
  calculateIdleSway,
  calculateWaveAnimation,
  calculateCelebrationAnimation,
  easing,
  lerp,
  lerpPosition,
} from './avatarService'

describe('AvatarStateMachine', () => {
  let machine: AvatarStateMachine

  beforeEach(() => {
    machine = new AvatarStateMachine()
  })

  it('should start in idle state', () => {
    expect(machine.getState()).toBe('idle')
  })

  it('should start with custom initial state', () => {
    const lookingMachine = new AvatarStateMachine('looking')
    expect(lookingMachine.getState()).toBe('looking')
  })

  it('should transition from idle to looking', () => {
    const result = machine.transition('looking')

    expect(result).toBe(true)
    expect(machine.getState()).toBe('looking')
  })

  it('should transition from idle to waving', () => {
    const result = machine.transition('waving')

    expect(result).toBe(true)
    expect(machine.getState()).toBe('waving')
  })

  it('should reject invalid transitions', () => {
    // Can't go directly from idle to celebrating
    const result = machine.transition('celebrating')

    expect(result).toBe(false)
    expect(machine.getState()).toBe('idle')
  })

  it('should force transition regardless of rules', () => {
    machine.forceTransition('celebrating')

    expect(machine.getState()).toBe('celebrating')
  })

  it('should track state time', () => {
    // Machine just started
    const initialTime = machine.getStateTime()

    expect(initialTime).toBeGreaterThanOrEqual(0)
    expect(initialTime).toBeLessThan(100) // Should be very small
  })

  it('should report state completion for timed states', () => {
    machine.forceTransition('waving')

    // Just started, should not be complete
    expect(machine.isStateComplete()).toBe(false)
  })

  it('should notify subscribers on state change', () => {
    const listener = vi.fn()
    machine.subscribe(listener)

    machine.transition('looking')

    expect(listener).toHaveBeenCalledWith('looking')
  })

  it('should allow unsubscribing', () => {
    const listener = vi.fn()
    const unsubscribe = machine.subscribe(listener)

    unsubscribe()
    machine.transition('looking')

    expect(listener).not.toHaveBeenCalled()
  })

  it('should reset state time on transition', () => {
    machine.transition('looking')
    const time1 = machine.getStateTime()

    machine.transition('idle')
    const time2 = machine.getStateTime()

    expect(time2).toBeLessThanOrEqual(time1 + 10) // Allow small timing variance
  })
})

describe('calculateLookRotation', () => {
  it('should return zero rotation when looking straight ahead', () => {
    const headPos: [number, number, number] = [0, 0, 0]
    const target = { x: 0, y: 0, z: 5 }

    const [rotX, rotY] = calculateLookRotation(headPos, target)

    expect(Math.abs(rotX)).toBeLessThan(0.1)
    expect(Math.abs(rotY)).toBeLessThan(0.1)
  })

  it('should rotate right for positive X target', () => {
    const headPos: [number, number, number] = [0, 0, 0]
    const target = { x: 5, y: 0, z: 5 }

    const [, rotY] = calculateLookRotation(headPos, target)

    expect(rotY).toBeGreaterThan(0) // Positive rotation for right
  })

  it('should rotate left for negative X target', () => {
    const headPos: [number, number, number] = [0, 0, 0]
    const target = { x: -5, y: 0, z: 5 }

    const [, rotY] = calculateLookRotation(headPos, target)

    expect(rotY).toBeLessThan(0) // Negative rotation for left
  })

  it('should rotate up for positive Y target', () => {
    const headPos: [number, number, number] = [0, 0, 0]
    const target = { x: 0, y: 5, z: 5 }

    const [rotX] = calculateLookRotation(headPos, target)

    // Rotation should be non-zero when looking up
    expect(rotX).not.toBe(0)
  })

  it('should clamp rotation to max value', () => {
    const headPos: [number, number, number] = [0, 0, 0]
    const target = { x: 100, y: 100, z: 0.1 } // Extreme values

    const [rotX, rotY] = calculateLookRotation(headPos, target, Math.PI / 4)

    expect(Math.abs(rotY)).toBeLessThanOrEqual(Math.PI / 4 + 0.01)
    expect(Math.abs(rotX)).toBeLessThanOrEqual(Math.PI / 8 + 0.01)
  })
})

describe('calculateIdleSway', () => {
  it('should return position and rotation offsets', () => {
    const sway = calculateIdleSway(0)

    expect(sway.positionOffset).toHaveLength(3)
    expect(sway.rotationOffset).toHaveLength(3)
  })

  it('should vary with time', () => {
    const sway1 = calculateIdleSway(0)
    const sway2 = calculateIdleSway(Math.PI)

    // Values should be different at different times
    expect(sway1.positionOffset[0]).not.toBe(sway2.positionOffset[0])
  })

  it('should stay within reasonable bounds', () => {
    for (let t = 0; t < 10; t += 0.1) {
      const sway = calculateIdleSway(t)

      sway.positionOffset.forEach((offset) => {
        expect(Math.abs(offset)).toBeLessThan(0.1)
      })

      sway.rotationOffset.forEach((offset) => {
        expect(Math.abs(offset)).toBeLessThan(0.1)
      })
    }
  })
})

describe('calculateWaveAnimation', () => {
  it('should return three rotation values', () => {
    const wave = calculateWaveAnimation(0.5)

    expect(wave).toHaveLength(3)
  })

  it('should start at reasonable values', () => {
    const wave = calculateWaveAnimation(0)

    wave.forEach((value) => {
      expect(Math.abs(value)).toBeLessThan(1)
    })
  })

  it('should change through animation progress', () => {
    const wave1 = calculateWaveAnimation(0)
    const wave2 = calculateWaveAnimation(0.5)
    const wave3 = calculateWaveAnimation(1)

    // Values should vary across the animation
    const totalVariation =
      Math.abs(wave1[0] - wave2[0]) +
      Math.abs(wave2[0] - wave3[0]) +
      Math.abs(wave1[1] - wave2[1])

    expect(totalVariation).toBeGreaterThan(0)
  })
})

describe('calculateCelebrationAnimation', () => {
  it('should return scale, rotation, and particle burst flag', () => {
    const celebration = calculateCelebrationAnimation(0.5)

    expect(typeof celebration.scale).toBe('number')
    expect(typeof celebration.rotation).toBe('number')
    expect(typeof celebration.particleBurst).toBe('boolean')
  })

  it('should trigger particle burst at start', () => {
    const celebration = calculateCelebrationAnimation(0.05)

    expect(celebration.particleBurst).toBe(true)
  })

  it('should trigger particle burst at end', () => {
    const celebration = calculateCelebrationAnimation(0.95)

    expect(celebration.particleBurst).toBe(true)
  })

  it('should not trigger particle burst in middle', () => {
    const celebration = calculateCelebrationAnimation(0.5)

    expect(celebration.particleBurst).toBe(false)
  })

  it('should complete full rotation at progress 1', () => {
    const celebration = calculateCelebrationAnimation(1)

    expect(celebration.rotation).toBeCloseTo(Math.PI * 2, 2)
  })
})

describe('easing functions', () => {
  describe('linear', () => {
    it('should return input unchanged', () => {
      expect(easing.linear(0)).toBe(0)
      expect(easing.linear(0.5)).toBe(0.5)
      expect(easing.linear(1)).toBe(1)
    })
  })

  describe('easeInOut', () => {
    it('should start slow', () => {
      expect(easing.easeInOut(0.1)).toBeLessThan(0.1)
    })

    it('should end slow', () => {
      expect(easing.easeInOut(0.9)).toBeGreaterThan(0.9)
    })

    it('should be 0.5 at midpoint', () => {
      expect(easing.easeInOut(0.5)).toBeCloseTo(0.5, 5)
    })
  })

  describe('easeOut', () => {
    it('should start fast', () => {
      expect(easing.easeOut(0.1)).toBeGreaterThan(0.1)
    })

    it('should end at 1', () => {
      expect(easing.easeOut(1)).toBe(1)
    })
  })

  describe('easeIn', () => {
    it('should start slow', () => {
      expect(easing.easeIn(0.1)).toBeLessThan(0.1)
    })

    it('should end at 1', () => {
      expect(easing.easeIn(1)).toBe(1)
    })
  })

  describe('bounce', () => {
    it('should start at 0', () => {
      expect(easing.bounce(0)).toBeCloseTo(0)
    })

    it('should end at 1', () => {
      expect(easing.bounce(1)).toBeCloseTo(1, 4)
    })

    it('should overshoot near end (bounce effect)', () => {
      // Bounce easing has characteristic bounce pattern
      const nearEnd = easing.bounce(0.9)
      expect(nearEnd).toBeLessThan(1.05)
    })
  })

  describe('elastic', () => {
    it('should start at 0', () => {
      expect(easing.elastic(0)).toBe(0)
    })

    it('should end at 1', () => {
      expect(easing.elastic(1)).toBe(1)
    })
  })
})

describe('lerp', () => {
  it('should return start at t=0', () => {
    expect(lerp(0, 100, 0)).toBe(0)
  })

  it('should return end at t=1', () => {
    expect(lerp(0, 100, 1)).toBe(100)
  })

  it('should return midpoint at t=0.5', () => {
    expect(lerp(0, 100, 0.5)).toBe(50)
  })

  it('should work with negative values', () => {
    expect(lerp(-100, 100, 0.5)).toBe(0)
  })

  it('should extrapolate beyond 0-1 range', () => {
    expect(lerp(0, 100, 2)).toBe(200)
    expect(lerp(0, 100, -0.5)).toBe(-50)
  })
})

describe('lerpPosition', () => {
  it('should interpolate all three components', () => {
    const start: [number, number, number] = [0, 0, 0]
    const end: [number, number, number] = [10, 20, 30]

    const result = lerpPosition(start, end, 0.5)

    expect(result).toEqual([5, 10, 15])
  })

  it('should return start at t=0', () => {
    const start: [number, number, number] = [1, 2, 3]
    const end: [number, number, number] = [10, 20, 30]

    const result = lerpPosition(start, end, 0)

    expect(result).toEqual([1, 2, 3])
  })

  it('should return end at t=1', () => {
    const start: [number, number, number] = [1, 2, 3]
    const end: [number, number, number] = [10, 20, 30]

    const result = lerpPosition(start, end, 1)

    expect(result).toEqual([10, 20, 30])
  })
})
