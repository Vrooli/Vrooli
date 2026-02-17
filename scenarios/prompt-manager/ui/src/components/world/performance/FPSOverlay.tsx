/**
 * FPSOverlay - Displays FPS and performance metrics in the 3D scene.
 * Shows current FPS, average FPS, and performance tier information.
 */
// AI_CHECK: FPS_TRACE_OVERLAY_RENDER=1 | LAST: 2026-02-17

import { useEffect, useRef } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { usePerformanceStore, selectTraceData } from '@/stores/performanceStore'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { useLODStore } from '@/stores/lodStore'
import type { PerformanceTraceMarker, PerformanceTraceSample } from '@/types/performance'

interface FPSOverlayProps {
  /** Whether to show detailed stats */
  detailed?: boolean
}

const TRACE_CANVAS_WIDTH = 220
const TRACE_CANVAS_HEIGHT = 64

function markerColor(type: PerformanceTraceMarker['type']): string {
  switch (type) {
    case 'tier-adjust':
      return '#60a5fa'
    case 'degraded':
      return '#ef4444'
    case 'recovered':
      return '#22c55e'
    case 'hidden':
      return '#f59e0b'
    case 'visible':
      return '#10b981'
  }
}

function drawLine(
  ctx: CanvasRenderingContext2D,
  samples: PerformanceTraceSample[],
  valueForSample: (sample: PerformanceTraceSample) => number,
  minValue: number,
  maxValue: number,
  color: string,
  width: number,
  height: number,
) {
  if (samples.length === 0) return

  const range = Math.max(1, maxValue - minValue)
  const stepX = samples.length <= 1 ? 0 : width / (samples.length - 1)

  ctx.beginPath()
  for (let i = 0; i < samples.length; i++) {
    const sample = samples[i]
    if (!sample) continue
    const v = valueForSample(sample)
    const normalized = (v - minValue) / range
    const x = i * stepX
    const y = height - normalized * height
    if (i === 0) {
      ctx.moveTo(x, y)
    } else {
      ctx.lineTo(x, y)
    }
  }
  ctx.lineWidth = 1.5
  ctx.strokeStyle = color
  ctx.stroke()
}

function drawTraceCanvas(
  canvas: HTMLCanvasElement,
  samples: PerformanceTraceSample[],
  markers: PerformanceTraceMarker[],
) {
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const width = canvas.width
  const height = canvas.height
  ctx.clearRect(0, 0, width, height)
  ctx.fillStyle = 'rgba(2,6,23,0.75)'
  ctx.fillRect(0, 0, width, height)

  if (samples.length === 0) {
    ctx.fillStyle = '#94a3b8'
    ctx.font = '10px monospace'
    ctx.fillText('collecting trace...', 8, Math.round(height / 2))
    return
  }

  const fpsValues = samples.map((s) => s.fps)
  const frameMsValues = samples.map((s) => s.frameTimeMs)
  const minFps = Math.max(0, Math.min(...fpsValues) - 5)
  const maxFps = Math.max(60, Math.max(...fpsValues) + 5)
  const minFrameMs = 0
  const maxFrameMs = Math.max(40, Math.max(...frameMsValues) + 2)

  drawLine(ctx, samples, (s) => s.fps, minFps, maxFps, '#22c55e', width, height)
  drawLine(ctx, samples, (s) => s.frameTimeMs, minFrameMs, maxFrameMs, '#f59e0b', width, height)

  const startTs = samples[0]?.timestamp ?? 0
  const endTs = samples[samples.length - 1]?.timestamp ?? startTs
  const tsRange = Math.max(1, endTs - startTs)
  for (const marker of markers) {
    if (marker.timestamp < startTs || marker.timestamp > endTs) continue
    const normalized = (marker.timestamp - startTs) / tsRange
    const x = Math.round(normalized * width)
    ctx.beginPath()
    ctx.moveTo(x, 0)
    ctx.lineTo(x, height)
    ctx.strokeStyle = markerColor(marker.type)
    ctx.lineWidth = 1
    ctx.stroke()
  }
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
  detailed = false,
}: FPSOverlayProps) {
  // Performance metrics
  const metrics = usePerformanceStore((state) => state.metrics)
  const autoAdjust = usePerformanceStore((state) => state.config.autoAdjust)
  const showTraceCharts = usePerformanceStore((state) => state.config.showTraceCharts)
  const traceData = usePerformanceStore(useShallow(selectTraceData))
  const traceCanvasRef = useRef<HTMLCanvasElement | null>(null)

  // Graphics tier
  const tier = useGraphicsStore((state) => state.tier)

  // LOD stats — useShallow prevents infinite re-render from the inline object literal
  const lodStats = useLODStore(useShallow((state) => ({
    objectCount: state.objectCount,
    levelCounts: state.levelCounts,
  })))

  useEffect(() => {
    const canvas = traceCanvasRef.current
    if (!canvas) return
    drawTraceCanvas(canvas, traceData.samples, traceData.markers)
  }, [traceData.version, traceData.samples, traceData.markers])

  // Color based on FPS
  const fpsColor =
    metrics.averageFps >= 55
      ? '#22c55e' // Green - good
      : metrics.averageFps >= 40
        ? '#eab308' // Yellow - acceptable
        : '#ef4444' // Red - poor

  return (
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
            {showTraceCharts && (
              <div
                style={{
                  marginTop: '6px',
                  paddingTop: '6px',
                  borderTop: '1px solid rgba(255,255,255,0.1)',
                }}
              >
                <div
                  style={{
                    marginBottom: '4px',
                    color: '#9ca3af',
                    fontSize: '10px',
                    display: 'flex',
                    gap: '10px',
                  }}
                >
                  <span style={{ color: '#22c55e' }}>FPS</span>
                  <span style={{ color: '#f59e0b' }}>Frame ms</span>
                  <span style={{ color: '#60a5fa' }}>Tier event</span>
                </div>
                <canvas
                  ref={traceCanvasRef}
                  width={TRACE_CANVAS_WIDTH}
                  height={TRACE_CANVAS_HEIGHT}
                  style={{
                    width: `${TRACE_CANVAS_WIDTH}px`,
                    height: `${TRACE_CANVAS_HEIGHT}px`,
                    borderRadius: '4px',
                    border: '1px solid rgba(255,255,255,0.08)',
                  }}
                />
              </div>
            )}

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
  )
}

/**
 * Minimal FPS counter for corner display.
 */
export function MiniFPSCounter() {
  const fps = usePerformanceStore((state) => state.metrics.currentFps)

  const color =
    fps >= 55 ? '#22c55e' : fps >= 40 ? '#eab308' : '#ef4444'

  return (
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
  )
}
