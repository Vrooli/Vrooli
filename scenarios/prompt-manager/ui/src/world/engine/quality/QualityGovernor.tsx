import { PerformanceMonitor } from '@react-three/drei'
import type { QualityProfile, QualityTuning } from '../../config'
import type { QualityProfileId } from '../../config'
import { governorBounds, stepDown, stepUp, type QualityVerdictRecord } from './governor'

interface QualityGovernorProps {
  auto: boolean
  profile: QualityProfile
  quality: QualityTuning
  profileId: QualityProfileId
  onVerdict: (verdict: QualityVerdictRecord) => void
}

/**
 * Renders drei's PerformanceMonitor only while auto is on, with bounds derived
 * from the active profile's own frame cap. A manual profile mounts nothing,
 * so nothing can override it.
 */
export function QualityGovernor({ auto, profile, profileId, quality, onVerdict }: QualityGovernorProps) {
  if (!auto) return null
  const emit = (verdict: 'decline' | 'incline', fps: number, refreshRate: number, fallback = false) => {
    const bounds = governorBounds(profile, quality, refreshRate)
    const to = verdict === 'decline' ? stepDown(profileId) : profileId === 'high' && refreshRate < quality.ultraMinRefreshRate ? profileId : stepUp(profileId)
    onVerdict({ verdict, measuredFps: fps, boundFps: verdict === 'decline' ? bounds[0] : bounds[1], from: profileId, to, reason: `${fps.toFixed(1)} fps ${verdict === 'decline' ? 'below' : 'above'} ${bounds[verdict === 'decline' ? 0 : 1].toFixed(1)} fps${fallback ? ' (monitor fallback)' : ''}`, at: new Date().toISOString() })
  }
  return (
    <PerformanceMonitor
      bounds={(refreshRate) => governorBounds(profile, quality, refreshRate)}
      flipflops={quality.monitorFlipflops}
      onDecline={(api) => emit('decline', api.fps, api.refreshrate)}
      onIncline={(api) => emit('incline', api.fps, api.refreshrate)}
      onFallback={(api) => emit('decline', api.fps, api.refreshrate, true)}
    />
  )
}
