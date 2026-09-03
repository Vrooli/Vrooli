import type { QualityProfile, QualityProfileId, QualityState, QualityTuning } from '../../config'
import { QUALITY_PROFILE_IDS, isQualityProfileId } from '../../config'

export { isQualityProfileId }

export type { QualityState }

/** Frame-rate bounds the performance monitor compares against, derived from the active profile's own cap. */
export function governorBounds(profile: QualityProfile, quality: QualityTuning): [lower: number, upper: number] {
  return [profile.frameCapFps * quality.degradedRatio, profile.frameCapFps * quality.recoverRatio]
}

export function stepDown(id: QualityProfileId): QualityProfileId {
  const index = QUALITY_PROFILE_IDS.indexOf(id)
  return QUALITY_PROFILE_IDS[Math.max(0, index - 1)] ?? id
}

export function stepUp(id: QualityProfileId): QualityProfileId {
  const index = QUALITY_PROFILE_IDS.indexOf(id)
  return QUALITY_PROFILE_IDS[Math.min(QUALITY_PROFILE_IDS.length - 1, index + 1)] ?? id
}

/**
 * Pure transition: given the current state and a monitor verdict, return the
 * next state. A manual profile never changes; auto mode steps one profile at
 * a time and never below low or above ultra.
 */
export function applyVerdict(state: QualityState, verdict: 'decline' | 'incline'): QualityState {
  if (!state.auto) return state
  const next = verdict === 'decline' ? stepDown(state.profileId) : stepUp(state.profileId)
  return next === state.profileId ? state : { ...state, profileId: next }
}

/** A user pick is always manual; the governor stops adjusting until auto is re-enabled. */
export function pickProfile(profileId: QualityProfileId): QualityState {
  return { auto: false, profileId }
}

export function setAuto(state: QualityState, auto: boolean): QualityState {
  return { ...state, auto }
}

