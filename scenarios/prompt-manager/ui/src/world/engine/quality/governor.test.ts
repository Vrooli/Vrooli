import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { applyVerdict, chooseInitialProfile, governorBounds, pickProfile, setAuto, stepDown, stepUp, type QualityState } from './governor'

describe('quality governor', () => {
  it('derives the degraded threshold from the profile cap, not a fixed number', () => {
    const medium = tuning.quality.profiles.medium
    const [lower, upper] = governorBounds(medium, tuning.quality)
    expect(lower).toBeCloseTo(medium.frameCapFps * tuning.quality.degradedRatio)
    expect(upper).toBeCloseTo(medium.frameCapFps * tuning.quality.recoverRatio)
    // A profile capped at 30 fps that renders 30 fps is healthy, never degraded.
    const capped = { ...medium, frameCapFps: 30 }
    expect(governorBounds(capped, tuning.quality)[0]).toBeLessThan(30)
  })

  it('a manual profile is never overridden by monitor verdicts', () => {
    const manual = pickProfile('ultra')
    expect(manual).toEqual({ auto: false, profileId: 'ultra' })
    expect(applyVerdict(manual, 'decline')).toBe(manual)
    expect(applyVerdict(manual, 'incline')).toBe(manual)
  })

  it('auto mode steps one profile per verdict and clamps at both ends', () => {
    let state: QualityState = { auto: true, profileId: 'high' }
    state = applyVerdict(state, 'decline')
    expect(state.profileId).toBe('medium')
    state = applyVerdict(state, 'decline')
    state = applyVerdict(state, 'decline')
    expect(state.profileId).toBe('low')
    state = applyVerdict(state, 'incline')
    expect(state.profileId).toBe('medium')
    expect(stepUp('ultra')).toBe('ultra')
    expect(stepDown('low')).toBe('low')
  })

  it('re-enabling auto keeps the current profile as the starting point', () => {
    const state = setAuto(pickProfile('low'), true)
    expect(state).toEqual({ auto: true, profileId: 'low' })
  })

  it('calibrates a fast 60 Hz display to high and a slow one to low', () => {
    expect(chooseInitialProfile(59, 60, tuning.quality)).toBe('high')
    expect(chooseInitialProfile(35, 60, tuning.quality)).toBe('low')
    expect(chooseInitialProfile(116, 120, tuning.quality)).toBe('ultra')
  })

  it('uses configured display and performance thresholds for the initial profile', () => {
    const quality = { ...tuning.quality, ultraMinRefreshRate: 144, recoverRatio: 0.8, degradedRatio: 0.6 }
    expect(chooseInitialProfile(110, 120, quality)).toBe('high')
    expect(chooseInitialProfile(140, 144, quality)).toBe('ultra')
    expect(chooseInitialProfile(40, 60, quality)).toBe('medium')
    expect(chooseInitialProfile(30, 60, quality)).toBe('low')
  })

  it('caps an ultra governor bound at the reachable refresh rate', () => {
    const [, upper] = governorBounds(tuning.quality.profiles.ultra, tuning.quality, 60)
    expect(upper).toBeLessThan(60)
  })
})
