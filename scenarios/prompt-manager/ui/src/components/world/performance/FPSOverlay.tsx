/**
 * FPSOverlay - Displays FPS and performance metrics in the 3D scene.
 * Shows current FPS, average FPS, and performance tier information.
 */

import { Html } from '@react-three/drei'
import { usePerformanceStore } from '@/stores/performanceStore'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { useLODStore } from '@/stores/lodStore'
import { useShallow } from 'zustand/react/shallow'

interface FPSOverlayProps {
  /** Position in 3D space */
  position?: [number, number, number]
  /** Whether to show detailed stats */
  detailed?: boolean
}

/**
 * FPS overlay component that displays performance metrics.
 * Must be used inside an R3F Canvas.
 *
 * @example
 * ```tsx
 * function Scene() {
 *   return (
 *     <>
 *       <SceneContent />
 *       <FPSOverlay position={[-5, 5, 0]} />
 *     </>
 *   )
 * }
 * ```
 */
export function FPSOverlay({
  position = [-5, 4, 0],
  detailed = false,
}: FPSOverlayProps) {
  // Performance metrics
  const metrics = usePerformanceStore((state) => state.metrics)
  const showOverlay = usePerformanceStore((state) => state.config.showOverlay)
  const autoAdjust = usePerformanceStore((state) => state.config.autoAdjust)

  // Graphics tier
  const tier = useGraphicsStore((state) => state.tier)

  // LOD stats — useShallow prevents infinite re-render from the inline object literal
  const lodStats = useLODStore(useShallow((state) => ({
    objectCount: state.objectCount,
    levelCounts: state.levelCounts,
  })))

  if (!showOverlay) return null

  // Color based on FPS
  const fpsColor =
    metrics.averageFps >= 55
      ? '#22c55e' // Green - good
      : metrics.averageFps >= 40
        ? '#eab308' // Yellow - acceptable
        : '#ef4444' // Red - poor

  return (
    <Html
      position={position}
      center={false}
      style={{
        pointerEvents: 'none',
        userSelect: 'none',
      }}
    >
      <div
        style={{
          background: 'rgba(0, 0, 0, 0.75)',
          color: '#fff',
          padding: '8px 12px',
          borderRadius: '6px',
          fontFamily: 'monospace',
          fontSize: '12px',
          minWidth: '140px',
          backdropFilter: 'blur(4px)',
        }}
      >
        {/* FPS Display */}
        <div style={{ marginBottom: '4px' }}>
          <span style={{ color: fpsColor, fontWeight: 'bold', fontSize: '16px' }}>
            {metrics.currentFps}
          </span>
          <span style={{ color: '#9ca3af', marginLeft: '4px' }}>FPS</span>
        </div>

        {/* Average FPS */}
        <div style={{ color: '#9ca3af', fontSize: '10px' }}>
          Avg: {metrics.averageFps} | Min: {metrics.minFps} | Max: {metrics.maxFps}
        </div>

        {/* Performance Tier */}
        <div
          style={{
            marginTop: '6px',
            paddingTop: '6px',
            borderTop: '1px solid rgba(255,255,255,0.1)',
          }}
        >
          <span style={{ color: '#9ca3af' }}>Tier: </span>
          <span
            style={{
              color: '#a5b4fc',
              textTransform: 'uppercase',
              fontWeight: 'bold',
            }}
          >
            {tier}
          </span>
          {autoAdjust && (
            <span
              style={{
                marginLeft: '6px',
                fontSize: '9px',
                color: '#22c55e',
              }}
            >
              (auto)
            </span>
          )}
        </div>

        {/* Degraded Warning */}
        {metrics.isDegraded && (
          <div
            style={{
              marginTop: '4px',
              color: '#ef4444',
              fontSize: '10px',
            }}
          >
            ⚠ Performance degraded
          </div>
        )}

        {/* Detailed Stats */}
        {detailed && (
          <>
            {/* Frame Time */}
            <div
              style={{
                marginTop: '6px',
                paddingTop: '6px',
                borderTop: '1px solid rgba(255,255,255,0.1)',
                fontSize: '10px',
                color: '#9ca3af',
              }}
            >
              Frame: {metrics.frameTimeMs.toFixed(2)}ms
              {metrics.memoryUsageMb !== null && (
                <span style={{ marginLeft: '8px' }}>
                  Mem: {metrics.memoryUsageMb}MB
                </span>
              )}
            </div>

            {/* LOD Stats */}
            <div
              style={{
                marginTop: '6px',
                paddingTop: '6px',
                borderTop: '1px solid rgba(255,255,255,0.1)',
                fontSize: '10px',
              }}
            >
              <div style={{ color: '#9ca3af', marginBottom: '2px' }}>
                LOD Objects: {lodStats.objectCount}
              </div>
              <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                <span style={{ color: '#22c55e' }}>
                  H:{lodStats.levelCounts.high}
                </span>
                <span style={{ color: '#eab308' }}>
                  M:{lodStats.levelCounts.medium}
                </span>
                <span style={{ color: '#f97316' }}>
                  L:{lodStats.levelCounts.low}
                </span>
                <span style={{ color: '#ef4444' }}>
                  C:{lodStats.levelCounts.culled}
                </span>
              </div>
            </div>
          </>
        )}
      </div>
    </Html>
  )
}

/**
 * Minimal FPS counter for corner display.
 */
export function MiniFPSCounter() {
  const fps = usePerformanceStore((state) => state.metrics.currentFps)
  const showOverlay = usePerformanceStore((state) => state.config.showOverlay)

  if (!showOverlay) return null

  const color =
    fps >= 55 ? '#22c55e' : fps >= 40 ? '#eab308' : '#ef4444'

  return (
    <Html
      position={[-6, 5, 0]}
      style={{ pointerEvents: 'none', userSelect: 'none' }}
    >
      <div
        style={{
          background: 'rgba(0,0,0,0.6)',
          color,
          padding: '2px 6px',
          borderRadius: '4px',
          fontFamily: 'monospace',
          fontSize: '14px',
          fontWeight: 'bold',
        }}
      >
        {fps}
      </div>
    </Html>
  )
}
