/**
 * ProceduralClouds - Renders volumetric clouds using drei's Cloud component.
 * Cloud colors adjust based on time of day.
 *
 * PERF: Uses stable references for all props to prevent R3F reconciliation overhead.
 * See: skills/core/r3f-coherence.md - Pattern 3: Avoid Inline Objects in JSX
 */

import { memo, useMemo, useRef } from 'react'
import { Cloud, Clouds } from '@react-three/drei'
import { useEnvironmentStore } from '@/stores/environmentStore'
import type { SceneType } from '@/types/environment'

// Stable constants for Cloud props - NEVER inline these in JSX
const CLOUD_BOUNDS: [number, number, number] = [12, 3, 12]
const CLOUD_VOLUME = 10
const CLOUD_FADE = 50
const CLOUD_SEGMENTS_HIGH = 40
const CLOUD_SEGMENTS_LOW = 20

interface ProceduralCloudsProps {
  /** Override the time value (0-24 hours) */
  timeValue?: number
  /** Override scene type */
  sceneType?: SceneType
  /** Number of cloud instances */
  count?: number
  /** Whether to use reduced count for performance */
  lowQuality?: boolean
}

/**
 * Cloud color based on time of day.
 */
function getCloudColor(hour: number): string {
  const h = ((hour % 24) + 24) % 24

  if (h >= 6 && h < 8) {
    // Sunrise - golden/pink tint
    return '#FFD4A8'
  } else if (h >= 8 && h < 16) {
    // Midday - bright white
    return '#FFFFFF'
  } else if (h >= 16 && h < 18) {
    // Afternoon - slightly warm
    return '#FFF5E8'
  } else if (h >= 18 && h < 20) {
    // Sunset - golden/orange
    return '#FFB366'
  } else if (h >= 20 || h < 4) {
    // Night - silvery/blue tint
    return '#A8B8D0'
  } else {
    // Pre-dawn
    return '#C8D0E0'
  }
}

/**
 * Cloud opacity based on time of day.
 */
function getCloudOpacity(hour: number): number {
  const h = ((hour % 24) + 24) % 24

  if (h >= 20 || h < 6) {
    // Night - more transparent
    return 0.4
  }
  // Day - fuller clouds
  return 0.7
}

interface CloudData {
  position: [number, number, number]
  speed: number
}

/**
 * Generate random cloud data (positions and speeds) in a dome above the scene.
 * All random values are generated here so they stay stable across renders.
 */
function generateCloudData(count: number): CloudData[] {
  const clouds: CloudData[] = []
  const radius = 35
  const minHeight = 8
  const maxHeight = 18

  for (let i = 0; i < count; i++) {
    // Distribute clouds in a circular pattern
    const angle = (i / count) * Math.PI * 2 + Math.random() * 0.5
    const distance = 10 + Math.random() * (radius - 10)
    const height = minHeight + Math.random() * (maxHeight - minHeight)

    clouds.push({
      position: [
        Math.cos(angle) * distance,
        height,
        Math.sin(angle) * distance,
      ],
      speed: 0.1 + Math.random() * 0.1,
    })
  }

  return clouds
}

/**
 * Procedural clouds that respond to time of day.
 * Disabled in abstract-space, reduced opacity in indoor-office.
 *
 * Wrapped in memo to prevent re-renders from parent cursor tracking updates.
 */
export const ProceduralClouds = memo(function ProceduralClouds({
  timeValue: timeValueProp,
  sceneType: sceneTypeProp,
  count = 8,
  lowQuality = false,
}: ProceduralCloudsProps) {
  // PERF: Use granular selectors - only subscribe to exactly what we need
  // See: skills/core/r3f-coherence.md - Pattern 2: Zustand Selectors
  const storeTimeValue = useEnvironmentStore((state) => state.timeValue)
  const storeSceneType = useEnvironmentStore((state) => state.current.type)

  const timeValue = timeValueProp ?? storeTimeValue
  const sceneType = sceneTypeProp ?? storeSceneType

  // Reduce count for low quality mode
  const actualCount = lowQuality ? Math.ceil(count / 2) : count

  // IMPORTANT: All hooks must be called before any conditional returns (Rules of Hooks)
  // Use ref to store cloud data - this persists across ALL re-renders and never regenerates
  // Only regenerate if count actually changes
  const cloudDataRef = useRef<{ count: number; data: CloudData[] } | null>(null)
  if (!cloudDataRef.current || cloudDataRef.current.count !== actualCount) {
    cloudDataRef.current = {
      count: actualCount,
      data: generateCloudData(actualCount),
    }
  }
  const cloudData = cloudDataRef.current.data

  // Get time-based cloud appearance
  const cloudColor = useMemo(() => getCloudColor(timeValue), [timeValue])
  const cloudOpacity = useMemo(() => {
    let opacity = getCloudOpacity(timeValue)
    // Reduce opacity for indoor scenes
    if (sceneType === 'indoor-office') {
      opacity *= 0.5
    }
    return opacity
  }, [timeValue, sceneType])

  // Don't render clouds for abstract-space (after all hooks)
  if (sceneType === 'abstract-space') {
    return null
  }

  const segments = lowQuality ? CLOUD_SEGMENTS_LOW : CLOUD_SEGMENTS_HIGH

  return (
    <Clouds>
      {cloudData.map((cloud, idx) => (
        <Cloud
          key={`cloud-${idx}`}
          position={cloud.position}
          speed={cloud.speed}
          opacity={cloudOpacity}
          color={cloudColor}
          segments={segments}
          bounds={CLOUD_BOUNDS}
          volume={CLOUD_VOLUME}
          fade={CLOUD_FADE}
        />
      ))}
    </Clouds>
  )
})
