/**
 * EnvironmentControls - UI for changing the world environment.
 * Allows users to change time of day, scene type, and other environment settings.
 */

import { useState, useCallback } from 'react'
import { Sun, Moon, Sunrise, Sunset, Building2, Trees, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useEnvironmentStore, getTimeOfDayPreset } from '@/stores/environmentStore'
import type { TimeOfDay, DreiEnvironmentPreset } from '@/types/environment'

// Stable icon references
const TIME_OF_DAY_CONFIG: { time: TimeOfDay; icon: React.ReactNode; label: string }[] = [
  { time: 'morning', icon: <Sunrise className="h-4 w-4" />, label: 'Morning' },
  { time: 'noon', icon: <Sun className="h-4 w-4" />, label: 'Noon' },
  { time: 'sunset', icon: <Sunset className="h-4 w-4" />, label: 'Sunset' },
  { time: 'night', icon: <Moon className="h-4 w-4" />, label: 'Night' },
]

// Scene type configurations
type SceneType = 'abstract-space' | 'outdoor-park' | 'indoor-office'

const SCENE_TYPE_CONFIG: { type: SceneType; icon: React.ReactNode; label: string; preset: DreiEnvironmentPreset }[] = [
  { type: 'abstract-space', icon: <Sparkles className="h-4 w-4" />, label: 'Space', preset: 'night' },
  { type: 'outdoor-park', icon: <Trees className="h-4 w-4" />, label: 'Park', preset: 'forest' },
  { type: 'indoor-office', icon: <Building2 className="h-4 w-4" />, label: 'Office', preset: 'apartment' },
]

interface EnvironmentControlsProps {
  className?: string
}

/**
 * Environment controls for changing world appearance.
 */
export function EnvironmentControls({ className }: EnvironmentControlsProps) {
  const preferredTimeOfDay = useEnvironmentStore((state) => state.preferredTimeOfDay)
  const setPreferredTimeOfDay = useEnvironmentStore((state) => state.setPreferredTimeOfDay)
  const setDreiPreset = useEnvironmentStore((state) => state.setDreiPreset)
  const syncWithTheme = useEnvironmentStore((state) => state.syncWithTheme)
  const setSyncWithTheme = useEnvironmentStore((state) => state.setSyncWithTheme)
  const currentEnv = useEnvironmentStore((state) => state.current)

  const [sceneType, setSceneType] = useState<SceneType>(
    (currentEnv?.type as SceneType) ?? 'abstract-space'
  )

  const handleTimeOfDayChange = useCallback(
    (timeOfDay: TimeOfDay) => {
      setPreferredTimeOfDay(timeOfDay)
      // Also update the drei preset to match
      if (!syncWithTheme) {
        setDreiPreset(getTimeOfDayPreset(timeOfDay))
      }
    },
    [setPreferredTimeOfDay, setDreiPreset, syncWithTheme]
  )

  const handleSceneTypeChange = useCallback(
    (type: SceneType, preset: DreiEnvironmentPreset) => {
      setSceneType(type)
      // Update the drei preset based on scene type
      if (!syncWithTheme) {
        setDreiPreset(preset)
      }
    },
    [setDreiPreset, syncWithTheme]
  )

  const handleToggleThemeSync = useCallback(() => {
    setSyncWithTheme(!syncWithTheme)
  }, [syncWithTheme, setSyncWithTheme])

  return (
    <div className={`flex flex-col gap-2 p-2 bg-slate-800/80 border border-slate-700 rounded-lg ${className ?? ''}`}>
      {/* Time of Day Row */}
      <div className="flex items-center gap-1">
        <span className="text-xs text-slate-400 w-12">Time:</span>
        {TIME_OF_DAY_CONFIG.map(({ time, icon, label }) => (
          <Button
            key={time}
            variant="ghost"
            size="sm"
            onClick={() => handleTimeOfDayChange(time)}
            className={`h-7 w-7 p-0 ${
              preferredTimeOfDay === time
                ? 'bg-indigo-500/30 text-indigo-300'
                : 'text-slate-400 hover:text-slate-200'
            }`}
            title={label}
          >
            {icon}
          </Button>
        ))}
      </div>

      {/* Scene Type Row */}
      <div className="flex items-center gap-1">
        <span className="text-xs text-slate-400 w-12">Scene:</span>
        {SCENE_TYPE_CONFIG.map(({ type, icon, label, preset }) => (
          <Button
            key={type}
            variant="ghost"
            size="sm"
            onClick={() => handleSceneTypeChange(type, preset)}
            className={`h-7 w-7 p-0 ${
              sceneType === type
                ? 'bg-indigo-500/30 text-indigo-300'
                : 'text-slate-400 hover:text-slate-200'
            }`}
            title={label}
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
            syncWithTheme ? 'text-amber-400' : 'text-slate-500'
          }`}
          title={syncWithTheme ? 'Environment synced with app theme' : 'Manual environment control'}
        >
          {syncWithTheme ? 'Auto' : 'Manual'}
        </Button>
      </div>
    </div>
  )
}
