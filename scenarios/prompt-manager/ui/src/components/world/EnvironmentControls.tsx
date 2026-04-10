/**
 * EnvironmentControls - UI for changing the world environment.
 * Allows users to control time (via slider), scene type, and real-time sync.
 */

import { useCallback, useMemo } from 'react'
import { Clock, Building2, Trees, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Slider } from '@/components/ui/slider'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { createEnvironmentConfig, getPresetFromTime } from '@/config/environments'
import { formatTimeFromHour } from '@/lib/sky/sunPosition'
import { useRealTimeClock } from '@/hooks/useRealTimeClock'
import type { SceneType } from '@/types/environment'
import { selectors } from '@/constants/selectors'

// Scene type configurations
const SCENE_TYPE_CONFIG: { type: SceneType; icon: React.ReactNode; label: string; testId: string }[] = [
  { type: 'abstract-space', icon: <Sparkles className="h-4 w-4" />, label: 'Space', testId: selectors.environment.sceneSpace },
  { type: 'outdoor-park', icon: <Trees className="h-4 w-4" />, label: 'Park', testId: selectors.environment.scenePark },
  { type: 'indoor-office', icon: <Building2 className="h-4 w-4" />, label: 'Office', testId: selectors.environment.sceneOffice },
]

interface EnvironmentControlsProps {
  className?: string
}

/**
 * Environment controls for changing world appearance.
 */
export function EnvironmentControls({ className }: EnvironmentControlsProps) {
  // Enable real-time clock sync
  useRealTimeClock()

  const timeValue = useEnvironmentStore((state) => state.timeValue)
  const setTimeValue = useEnvironmentStore((state) => state.setTimeValue)
  const realTimeMode = useEnvironmentStore((state) => state.realTimeMode)
  const setRealTimeMode = useEnvironmentStore((state) => state.setRealTimeMode)
  const setDreiPreset = useEnvironmentStore((state) => state.setDreiPreset)
  const setEnvironment = useEnvironmentStore((state) => state.setEnvironment)
  const syncWithTheme = useEnvironmentStore((state) => state.syncWithTheme)
  const setSyncWithTheme = useEnvironmentStore((state) => state.setSyncWithTheme)
  const currentEnv = useEnvironmentStore((state) => state.current)

  const sceneType: SceneType = currentEnv.type

  // Format the current time for display
  const timeDisplay = useMemo(() => formatTimeFromHour(timeValue), [timeValue])

  const handleTimeChange = useCallback(
    (values: number[]) => {
      const newTime = values[0] ?? timeValue
      setTimeValue(newTime)

      // Disable real-time mode when user manually adjusts
      if (realTimeMode) {
        setRealTimeMode(false)
      }

      // Update environment config with new time
      const newEnv = createEnvironmentConfig(
        `${sceneType}-${newTime}`,
        `${sceneType} environment`,
        { sceneType, timeValue: newTime }
      )
      setEnvironment(newEnv)

      // Update drei preset
      if (!syncWithTheme) {
        setDreiPreset(getPresetFromTime(newTime))
      }
    },
    [timeValue, setTimeValue, realTimeMode, setRealTimeMode, sceneType, setEnvironment, syncWithTheme, setDreiPreset]
  )

  const handleSceneTypeChange = useCallback(
    (type: SceneType) => {
      // Create and set full environment config
      const newEnv = createEnvironmentConfig(
        `${type}-${timeValue}`,
        `${type} environment`,
        { sceneType: type, timeValue }
      )
      setEnvironment(newEnv)
      // Also update drei preset for scene type
      if (!syncWithTheme) {
        setDreiPreset(getPresetFromTime(timeValue))
      }
    },
    [setDreiPreset, setEnvironment, syncWithTheme, timeValue]
  )

  const handleToggleRealTime = useCallback(() => {
    setRealTimeMode(!realTimeMode)
  }, [realTimeMode, setRealTimeMode])

  const handleToggleThemeSync = useCallback(() => {
    setSyncWithTheme(!syncWithTheme)
  }, [syncWithTheme, setSyncWithTheme])

  return (
    <div
      className={`flex flex-col gap-2 p-2 bg-card/80 border border-border rounded-lg ${className ?? ''}`}
      data-testid={selectors.environment.controls}
    >
      {/* Time Slider Row */}
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground w-12">Time:</span>
        <div className="flex-1 flex items-center gap-2">
          <Slider
            value={[timeValue]}
            onValueChange={handleTimeChange}
            min={0}
            max={24}
            step={0.25}
            disabled={realTimeMode}
            aria-label="Time of day"
            data-testid={selectors.environment.timeSlider}
          />
          <span className="text-xs text-foreground w-16 text-right font-mono">
            {timeDisplay}
          </span>
        </div>
        {/* Real-time Toggle */}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleToggleRealTime}
          className={`h-7 w-7 p-0 ${
            realTimeMode ? 'bg-indigo-500/30 text-indigo-300' : 'text-muted-foreground hover:text-foreground'
          }`}
          title={realTimeMode ? 'Real-time mode (synced with system clock)' : 'Manual time control'}
          data-testid={selectors.environment.realTimeToggle}
          aria-pressed={realTimeMode}
        >
          <Clock className="h-4 w-4" />
        </Button>
      </div>

      {/* Scene Type Row */}
      <div className="flex items-center gap-1">
        <span className="text-xs text-muted-foreground w-12">Scene:</span>
        {SCENE_TYPE_CONFIG.map(({ type, icon, label, testId }) => (
          <Button
            key={type}
            variant="ghost"
            size="sm"
            onClick={() => handleSceneTypeChange(type)}
            className={`h-7 w-7 p-0 ${
              sceneType === type
                ? 'bg-indigo-500/30 text-indigo-300'
                : 'text-muted-foreground hover:text-foreground'
            }`}
            title={label}
            data-testid={testId}
            aria-pressed={sceneType === type}
            data-active={sceneType === type}
          >
            {icon}
          </Button>
        ))}

        {/* Theme Sync Toggle */}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleToggleThemeSync}
          className={`h-7 px-2 text-xs ml-1 ${
            syncWithTheme ? 'text-amber-400' : 'text-muted-foreground/60'
          }`}
          title={syncWithTheme ? 'Environment synced with app theme' : 'Manual environment control'}
        >
          {syncWithTheme ? 'Auto' : 'Manual'}
        </Button>
      </div>
    </div>
  )
}
