/**
 * Hook to sync environment time with system clock.
 * When realTimeMode is enabled, updates timeValue every minute.
 */

import { useEffect } from 'react'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { getCurrentTimeAsHour } from '@/lib/sky/sunPosition'

/**
 * Syncs the environment store's timeValue with the system clock
 * when realTimeMode is enabled.
 *
 * Updates once immediately and then every minute.
 */
export function useRealTimeClock(): void {
  const realTimeMode = useEnvironmentStore((state) => state.realTimeMode)
  const setTimeValue = useEnvironmentStore((state) => state.setTimeValue)

  useEffect(() => {
    if (!realTimeMode) {
      return
    }

    // Update immediately
    setTimeValue(getCurrentTimeAsHour())

    // Update every minute
    const interval = setInterval(() => {
      setTimeValue(getCurrentTimeAsHour())
    }, 60_000)

    return () => clearInterval(interval)
  }, [realTimeMode, setTimeValue])
}

/**
 * Hook that returns whether real-time mode is active.
 */
export function useIsRealTimeMode(): boolean {
  return useEnvironmentStore((state) => state.realTimeMode)
}

/**
 * Hook that returns the current time value.
 */
export function useTimeValue(): number {
  return useEnvironmentStore((state) => state.timeValue)
}
