import { PerformanceMonitor } from '@react-three/drei'
import type { QualityProfile, QualityTuning } from '../../config'
import { governorBounds } from './governor'

interface QualityGovernorProps {
  auto: boolean
  profile: QualityProfile
  quality: QualityTuning
  onVerdict: (verdict: 'decline' | 'incline') => void
}

/**
 * Renders drei's PerformanceMonitor only while auto is on, with bounds derived
 * from the active profile's own frame cap. A manual profile mounts nothing,
 * so nothing can override it.
 */
export function QualityGovernor({ auto, profile, quality, onVerdict }: QualityGovernorProps) {
  if (!auto) return null
  return (
    <PerformanceMonitor
      bounds={() => governorBounds(profile, quality)}
      flipflops={quality.monitorFlipflops}
      onDecline={() => onVerdict('decline')}
      onIncline={() => onVerdict('incline')}
      onFallback={() => onVerdict('decline')}
    />
  )
}
