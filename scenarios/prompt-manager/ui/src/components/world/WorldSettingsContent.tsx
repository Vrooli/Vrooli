/**
 * WorldSettingsContent - Settings body for camera, time, scene, and graphics.
 *
 * Rendered inside shared floating panel by ViewOverlay. Contains no panel chrome.
 */

import { useCallback, useMemo } from 'react'
import {
  User,
  Eye,
  Map,
  Clock,
  Building2,
  Trees,
  Sparkles,
  Zap,
  Gauge,
  Rocket,
  Flame,
  Ruler,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Slider } from '@/components/ui/slider'
import { useCameraStore, type CameraMode } from '@/stores/cameraStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { usePerformanceStore } from '@/stores/performanceStore'
import { formatTimeFromHour } from '@/lib/sky/sunPosition'
import { useRealTimeClock } from '@/hooks/useRealTimeClock'
import type { SceneType } from '@/types/environment'
import type { PerformanceTier, GraphicsConfig, MaterialQuality, AntialiasingMethod } from '@/types/graphics'
import { SettingsToggle, SettingsSelect, SettingsSlider } from './settings'
import { useWorldScaleStore } from '@/stores/worldScaleStore'
import { selectors } from '@/constants/selectors'

const CAMERA_MODES: { mode: CameraMode; icon: React.ReactNode; label: string }[] = [
  { mode: 'freeform', icon: <Eye className="h-4 w-4" />, label: 'Default' },
  { mode: 'zoomed-agent', icon: <User className="h-4 w-4" />, label: 'Focus' },
  { mode: 'top-down', icon: <Map className="h-4 w-4" />, label: 'Aerial' },
]

const SCENE_TYPES: { type: SceneType; icon: React.ReactNode; label: string }[] = [
  { type: 'abstract-space', icon: <Sparkles className="h-4 w-4" />, label: 'Space' },
  { type: 'outdoor-park', icon: <Trees className="h-4 w-4" />, label: 'Park' },
  { type: 'indoor-office', icon: <Building2 className="h-4 w-4" />, label: 'Office' },
]

const GRAPHICS_TIERS: { tier: PerformanceTier; icon: React.ReactNode; label: string }[] = [
  { tier: 'low', icon: <Zap className="h-4 w-4" />, label: 'Low' },
  { tier: 'medium', icon: <Gauge className="h-4 w-4" />, label: 'Medium' },
  { tier: 'high', icon: <Rocket className="h-4 w-4" />, label: 'High' },
  { tier: 'ultra', icon: <Flame className="h-4 w-4" />, label: 'Ultra' },
]

const SHADOW_MAP_SIZES = [
  { value: 512, label: '512' },
  { value: 1024, label: '1024' },
  { value: 2048, label: '2048' },
  { value: 4096, label: '4096' },
]

const MATERIAL_OPTIONS: { value: MaterialQuality; label: string }[] = [
  { value: 'basic', label: 'Basic' },
  { value: 'standard', label: 'Standard' },
  { value: 'physical', label: 'Physical' },
]

const ANTIALIASING_OPTIONS: { value: AntialiasingMethod; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'fxaa', label: 'FXAA' },
  { value: 'smaa', label: 'SMAA' },
]

export function WorldSettingsContent() {
  useRealTimeClock()

  // Camera store
  const cameraMode = useCameraStore((state) => state.mode)
  const setFreeform = useCameraStore((state) => state.setFreeform)
  const setTopDown = useCameraStore((state) => state.setTopDown)
  const requestZoomToAgent = useCameraStore((state) => state.requestZoomToAgent)

  // Environment store
  const timeValue = useEnvironmentStore((state) => state.timeValue)
  const setTimeValue = useEnvironmentStore((state) => state.setTimeValue)
  const realTimeMode = useEnvironmentStore((state) => state.realTimeMode)
  const setRealTimeMode = useEnvironmentStore((state) => state.setRealTimeMode)
  const currentEnv = useEnvironmentStore((state) => state.current)
  const setSceneType = useEnvironmentStore((state) => state.setSceneType)

  const sceneType = currentEnv.type

  // Graphics store
  const graphicsTier = useGraphicsStore((state) => state.tier)
  const graphicsConfig = useGraphicsStore((state) => state.config)
  const graphicsOverrides = useGraphicsStore((state) => state.overrides)
  const setTier = useGraphicsStore((state) => state.setTier)
  const setOverride = useGraphicsStore((state) => state.setOverride)
  const clearOverrides = useGraphicsStore((state) => state.clearOverrides)

  const hasCustomOverrides = Object.keys(graphicsOverrides).length > 0

  // World scale store
  const agentScale = useWorldScaleStore((state) => state.agent)
  const furnitureScale = useWorldScaleStore((state) => state.furniture)
  const decorationScale = useWorldScaleStore((state) => state.decoration)
  const overlayScale = useWorldScaleStore((state) => state.overlay)
  const setScale = useWorldScaleStore((state) => state.setScale)
  const resetAllScales = useWorldScaleStore((state) => state.resetAll)

  // Performance store
  const autoAdjust = usePerformanceStore((state) => state.config.autoAdjust)

  const timeDisplay = useMemo(() => formatTimeFromHour(timeValue), [timeValue])

  const handleCameraModeChange = useCallback(
    (mode: CameraMode) => {
      if (mode === 'freeform') {
        setFreeform()
      } else if (mode === 'top-down') {
        setTopDown()
      } else {
        requestZoomToAgent()
      }
    },
    [setFreeform, setTopDown, requestZoomToAgent]
  )

  const handleTimeChange = useCallback(
    (values: number[]) => {
      const newTime = values[0] ?? timeValue
      setTimeValue(newTime)
      if (realTimeMode) {
        setRealTimeMode(false)
      }
    },
    [timeValue, setTimeValue, realTimeMode, setRealTimeMode]
  )

  const setAutoDetect = useGraphicsStore((state) => state.setAutoDetect)
  const handleTierChange = useCallback(
    (tier: PerformanceTier) => {
      clearOverrides()
      setTier(tier)
      usePerformanceStore.getState().setConfig({ autoAdjust: false })
      setAutoDetect(false)
    },
    [setTier, clearOverrides, setAutoDetect]
  )

  return (
    <div className="space-y-6">
      {/* Camera View Section */}
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">Camera View</h3>
        <div className="flex gap-2">
          {CAMERA_MODES.map(({ mode, icon, label }) => (
            <button
              key={mode}
              type="button"
              onClick={() => handleCameraModeChange(mode)}
              className={cn(
                'flex-1 flex flex-col items-center gap-1.5 px-3 py-2 rounded-lg border transition-all',
                cameraMode === mode
                  ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                  : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600'
              )}
              data-testid={`${selectors.settings.camera}-${mode}`}
            >
              {icon}
              <span className="text-xs">{label}</span>
            </button>
          ))}
        </div>
      </section>

      <div className="border-t border-white/10" />

      {/* Time of Day Section */}
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">Time of Day</h3>
        <div className="flex items-center gap-3">
          <Slider
            value={[timeValue]}
            onValueChange={handleTimeChange}
            min={0}
            max={24}
            step={0.25}
            disabled={realTimeMode}
            className="flex-1"
            aria-label="Time of day"
            data-testid={selectors.settings.timeSlider}
          />
          <span className="text-xs text-slate-300 w-16 text-right font-mono">
            {timeDisplay}
          </span>
          <button
            type="button"
            onClick={() => setRealTimeMode(!realTimeMode)}
            className={cn(
              'p-1.5 rounded transition-colors',
              realTimeMode
                ? 'bg-indigo-500/30 text-indigo-300'
                : 'text-slate-400 hover:text-slate-200 hover:bg-white/10'
            )}
            title={realTimeMode ? 'Real-time sync enabled' : 'Enable real-time sync'}
            data-testid={selectors.settings.realTimeToggle}
          >
            <Clock className="h-4 w-4" />
          </button>
        </div>
      </section>

      <div className="border-t border-white/10" />

      {/* Scene Section */}
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">Scene</h3>
        <div className="flex gap-2">
          {SCENE_TYPES.map(({ type, icon, label }) => (
            <button
              key={type}
              type="button"
              onClick={() => setSceneType(type)}
              className={cn(
                'flex-1 flex flex-col items-center gap-1.5 px-3 py-2 rounded-lg border transition-all',
                sceneType === type
                  ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                  : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600'
              )}
              data-testid={`${selectors.settings.scene}-${type}`}
            >
              {icon}
              <span className="text-xs">{label}</span>
            </button>
          ))}
        </div>
      </section>

      <div className="border-t border-white/10" />

      {/* Object Scales Section */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-indigo-400 flex items-center gap-1.5">
            <Ruler className="h-4 w-4" />
            Object Scales
          </h3>
          <button
            type="button"
            onClick={resetAllScales}
            className="text-xs text-slate-500 hover:text-slate-300 transition-colors"
          >
            Reset
          </button>
        </div>
        <div className="space-y-2.5">
          <SettingsSlider label="Agent" value={agentScale} onChange={(v) => setScale('agent', v)} min={0.1} max={3} step={0.05} />
          <SettingsSlider label="Furniture" value={furnitureScale} onChange={(v) => setScale('furniture', v)} min={0.1} max={3} step={0.05} />
          <SettingsSlider label="Decoration" value={decorationScale} onChange={(v) => setScale('decoration', v)} min={0.1} max={3} step={0.05} />
          <SettingsSlider label="Overlay" value={overlayScale} onChange={(v) => setScale('overlay', v)} min={0.1} max={3} step={0.05} />
        </div>
      </section>

      <div className="border-t border-white/10" />

      {/* Graphics Section */}
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">Graphics</h3>
        <div className="flex gap-2 mb-4">
          {GRAPHICS_TIERS.map(({ tier, icon, label }) => (
            <button
              key={tier}
              type="button"
              onClick={() => handleTierChange(tier)}
              className={cn(
                'flex-1 flex flex-col items-center gap-1.5 px-3 py-2 rounded-lg border transition-all',
                graphicsTier === tier && !hasCustomOverrides
                  ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                  : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600'
              )}
              data-testid={`${selectors.settings.graphics}-${tier}`}
            >
              {icon}
              <span className="text-xs">{label}</span>
            </button>
          ))}
        </div>

        <SettingsToggle
          label="Auto-adjust quality"
          value={autoAdjust}
          onChange={(v) => {
            usePerformanceStore.getState().setConfig({ autoAdjust: v })
            if (v) {
              setAutoDetect(true)
            }
          }}
        />

        <label className="flex items-center gap-2 mb-4 mt-3 cursor-pointer">
          <input
            type="checkbox"
            checked={hasCustomOverrides}
            onChange={(e) => {
              if (e.target.checked) {
                const currentConfig = useGraphicsStore.getState().config
                for (const [key, value] of Object.entries(currentConfig)) {
                  setOverride(key as keyof GraphicsConfig, value as GraphicsConfig[keyof GraphicsConfig])
                }
              } else {
                clearOverrides()
              }
            }}
            className="h-4 w-4 rounded border-slate-600 bg-slate-700 text-indigo-500 focus:ring-indigo-500 focus:ring-offset-0"
            data-testid={selectors.settings.customToggle}
          />
          <span className="text-xs text-slate-300">Custom</span>
        </label>

        {hasCustomOverrides && (
          <div className="p-3 rounded-lg bg-slate-800/50 border border-slate-700 space-y-3">
            <SettingsToggle label="Shadows" value={graphicsConfig.shadows} onChange={(v) => setOverride('shadows', v)} />
            <SettingsSelect label="Shadow Quality" value={graphicsConfig.shadowMapSize} options={SHADOW_MAP_SIZES} onChange={(v) => setOverride('shadowMapSize', v)} disabled={!graphicsConfig.shadows} />
            <SettingsToggle label="Post Processing" value={graphicsConfig.postProcessing} onChange={(v) => setOverride('postProcessing', v)} />
            <SettingsSelect label="Material" value={graphicsConfig.materialQuality} options={MATERIAL_OPTIONS} onChange={(v) => setOverride('materialQuality', v)} />
            <SettingsToggle label="Bloom" value={graphicsConfig.bloom} onChange={(v) => setOverride('bloom', v)} disabled={!graphicsConfig.postProcessing} />
            <SettingsToggle label="SSAO" value={graphicsConfig.ssao} onChange={(v) => setOverride('ssao', v)} disabled={!graphicsConfig.postProcessing} />
            <SettingsSelect label="Antialiasing" value={graphicsConfig.antialiasing} options={ANTIALIASING_OPTIONS} onChange={(v) => setOverride('antialiasing', v)} />
            <SettingsToggle label="Vignette" value={graphicsConfig.vignette} onChange={(v) => setOverride('vignette', v)} disabled={!graphicsConfig.postProcessing} />
            <SettingsToggle label="Contact Shadows" value={graphicsConfig.contactShadows} onChange={(v) => setOverride('contactShadows', v)} disabled={!graphicsConfig.shadows} />
            <SettingsToggle label="Environment Map" value={graphicsConfig.envMap} onChange={(v) => setOverride('envMap', v)} />
            <SettingsToggle label="Agent Wobble" value={graphicsConfig.agentWobble} onChange={(v) => setOverride('agentWobble', v)} />
          </div>
        )}
      </section>
    </div>
  )
}
