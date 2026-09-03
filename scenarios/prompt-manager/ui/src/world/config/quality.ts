import type { QualityProfileId } from './tuning.schema'

/** The user-facing quality choice: a profile and whether the governor may move it. */
export interface QualityState {
  /** Whether the governor may move between profiles. A manual pick sets this false. */
  auto: boolean
  profileId: QualityProfileId
}

export function isQualityProfileId(value: string | null | undefined): value is QualityProfileId {
  return value === 'low' || value === 'medium' || value === 'high' || value === 'ultra'
}
