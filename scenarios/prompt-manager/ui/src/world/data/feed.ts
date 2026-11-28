/**
 * The live feed: a server stream with reconnect and backoff, and a poll
 * fallback that starts when the stream is silent or failing. Both produce sim
 * signals; the consumer never knows which path delivered them.
 */
import type { DataTuning } from '../config'
import type { Signal } from '../sim'
import { diffRuns, eventToSignals, initialAdapterState, type AdapterState, type KnownRun } from './adapters'
import type { WorldApi } from './worldClient'

export type FeedMode = 'connecting' | 'stream' | 'polling' | 'stopped' | 'snapshot'

export interface FeedStatus {
  mode: FeedMode
  lastEventAt: number | null
  reconnects: number
  lastError: string | null
}

/** Whatever the host's setTimeout returns; never null. */
type TimerHandle = NonNullable<unknown>

export interface FeedDeps {
  api: Pick<WorldApi, 'streamFeed'>
  /** Fallback source: the existing running-agents endpoint. */
  pollRunning: () => Promise<KnownRun[]>
  onSignals: (signals: Signal[]) => void
  onStatus?: (status: FeedStatus) => void
  tuning: DataTuning
  now?: () => number
  setTimer?: (fn: () => void, ms: number) => TimerHandle
  clearTimer?: (handle: TimerHandle) => void
}

export interface FeedHandle {
  stop(): void
  status(): FeedStatus
}

/** Exponential backoff bounded by tuning. */
export function backoffMs(attempt: number, tuning: DataTuning): number {
  return Math.min(tuning.reconnectMaxMs, tuning.reconnectBaseMs * 2 ** Math.max(0, attempt))
}

export function startFeed(deps: FeedDeps): FeedHandle {
  const now = deps.now ?? (() => Date.now() / 1000)
  const setTimer: (fn: () => void, ms: number) => TimerHandle = deps.setTimer ?? ((fn, ms) => setTimeout(fn, ms))
  const clearTimer: (handle: TimerHandle) => void = deps.clearTimer ?? ((handle) => clearTimeout(handle as ReturnType<typeof setTimeout>))
  const adapter: AdapterState = initialAdapterState()
  const status: FeedStatus = { mode: 'connecting', lastEventAt: null, reconnects: 0, lastError: null }
  let stopped = false
  let controller: AbortController | null = null
  let reconnectTimer: TimerHandle | null = null
  let pollTimer: TimerHandle | null = null
  let silenceTimer: TimerHandle | null = null
  let attempt = 0

  const publish = () => deps.onStatus?.({ ...status })

  const setMode = (mode: FeedMode) => {
    if (status.mode !== mode) {
      status.mode = mode
      publish()
    }
  }

  const stopPolling = () => {
    if (pollTimer !== null) {
      clearTimer(pollTimer)
      pollTimer = null
    }
  }

  const pollOnce = async () => {
    if (stopped) return
    try {
      const fresh = await deps.pollRunning()
      const { signals, next } = diffRuns(adapter.known, fresh, now())
      adapter.known = next
      if (signals.length > 0) deps.onSignals(signals)
      status.lastEventAt = now()
      status.lastError = null
    } catch (error) {
      status.lastError = error instanceof Error ? error.message : String(error)
      publish()
    }
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- mutated by sibling closures (stop, setMode)
    if (!stopped && status.mode === 'polling') pollTimer = setTimer(() => void pollOnce(), deps.tuning.pollIntervalMs)
  }

  const startPolling = () => {
    if (stopped || status.mode === 'polling') return
    setMode('polling')
    void pollOnce()
  }

  const armSilenceTimer = () => {
    if (silenceTimer !== null) clearTimer(silenceTimer)
    silenceTimer = setTimer(() => {
      if (status.mode !== 'stream') startPolling()
    }, deps.tuning.fallbackAfterMs)
  }

  const scheduleReconnect = () => {
    if (stopped) return
    const delay = backoffMs(attempt, deps.tuning)
    attempt += 1
    status.reconnects += 1
    publish()
    reconnectTimer = setTimer(() => void connect(), delay)
  }

  async function connect(): Promise<void> {
    if (stopped) return
    controller = new AbortController()
    if (status.mode !== 'polling') setMode('connecting')
    armSilenceTimer()
    try {
      for await (const event of deps.api.streamFeed(adapter.lastSeq, controller.signal)) {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- mutated by stop()
        if (stopped) return
        if (silenceTimer !== null) {
          clearTimer(silenceTimer)
          silenceTimer = null
        }
        stopPolling()
        setMode('stream')
        attempt = 0
        status.lastEventAt = now()
        status.lastError = null
        const signals = eventToSignals(event, adapter, now())
        if (signals.length > 0) deps.onSignals(signals)
      }
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- mutated by stop()
      if (!stopped) scheduleReconnect()
    } catch (error) {
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- mutated by stop()
      if (stopped) return
      status.lastError = error instanceof Error ? error.message : String(error)
      startPolling()
      scheduleReconnect()
    }
  }

  void connect()

  return {
    stop: () => {
      stopped = true
      controller?.abort()
      if (reconnectTimer !== null) clearTimer(reconnectTimer)
      if (silenceTimer !== null) clearTimer(silenceTimer)
      stopPolling()
      setMode('stopped')
    },
    status: () => ({ ...status }),
  }
}
