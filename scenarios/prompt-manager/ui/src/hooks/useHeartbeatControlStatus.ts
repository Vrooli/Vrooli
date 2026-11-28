/**
 * useHeartbeatControlStatus - Shared polling hook for heartbeat auto-pause state.
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import {
  getHeartbeatControlStatus,
  pauseHeartbeatControl,
  resumeHeartbeatControl,
  type HeartbeatControlStatus,
} from '@/services/heartbeatService'

const POLL_INTERVAL_MS = 45_000

interface SharedHeartbeatControlState {
  status: HeartbeatControlStatus | null
  isLoading: boolean
}

export interface UseHeartbeatControlStatusResult {
  status: HeartbeatControlStatus | null
  isLoading: boolean
  refresh: () => Promise<void>
  pause: (reason?: string) => Promise<void>
  resume: () => Promise<void>
}

const INITIAL_STATE: SharedHeartbeatControlState = {
  status: null,
  isLoading: true,
}

let sharedState: SharedHeartbeatControlState = INITIAL_STATE
let pollTimeoutId: ReturnType<typeof setTimeout> | null = null
let isPolling = false
const subscribers = new Set<(state: SharedHeartbeatControlState) => void>()

export function resetHeartbeatControlPollingForTests() {
  sharedState = INITIAL_STATE
  clearPollTimeout()
  isPolling = false
  subscribers.clear()
}

function clearPollTimeout() {
  if (pollTimeoutId !== null) {
    clearTimeout(pollTimeoutId)
    pollTimeoutId = null
  }
}

function emitState() {
  for (const listener of subscribers) {
    listener(sharedState)
  }
}

function setSharedState(next: SharedHeartbeatControlState) {
  sharedState = next
  emitState()
}

function schedulePoll() {
  if (pollTimeoutId !== null || subscribers.size === 0) return
  pollTimeoutId = setTimeout(() => {
    pollTimeoutId = null
    void poll()
  }, POLL_INTERVAL_MS)
}

async function poll() {
  if (isPolling || subscribers.size === 0) return
  isPolling = true
  try {
    if (document.hidden) {
      schedulePoll()
      return
    }
    const status = await getHeartbeatControlStatus()
    setSharedState({ status, isLoading: false })
  } catch {
    if (sharedState.isLoading) {
      setSharedState({ ...sharedState, isLoading: false })
    }
  } finally {
    isPolling = false
    schedulePoll()
  }
}

function subscribe(listener: (state: SharedHeartbeatControlState) => void): () => void {
  subscribers.add(listener)
  listener(sharedState)
  if (subscribers.size === 1) {
    clearPollTimeout()
    void poll()
  }
  return () => {
    subscribers.delete(listener)
    if (subscribers.size === 0) clearPollTimeout()
  }
}

export async function refreshHeartbeatControlStatus() {
  const status = await getHeartbeatControlStatus()
  setSharedState({ status, isLoading: false })
}

export function useHeartbeatControlStatus(): UseHeartbeatControlStatusResult {
  const [state, setState] = useState<SharedHeartbeatControlState>(() => sharedState)
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    const unsubscribe = subscribe((next) => {
      if (mountedRef.current) setState(next)
    })
    return () => {
      mountedRef.current = false
      unsubscribe()
    }
  }, [])

  const refresh = useCallback(async () => {
    await refreshHeartbeatControlStatus()
  }, [])

  const pause = useCallback(async (reason?: string) => {
    const status = await pauseHeartbeatControl({ reason })
    setSharedState({ status, isLoading: false })
  }, [])

  const resume = useCallback(async () => {
    const status = await resumeHeartbeatControl()
    setSharedState({ status, isLoading: false })
  }, [])

  return {
    status: state.status,
    isLoading: state.isLoading,
    refresh,
    pause,
    resume,
  }
}
