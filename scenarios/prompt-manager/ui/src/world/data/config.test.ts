import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { Code, ConnectError } from '@connectrpc/connect'
import { DEFAULT_PREFERENCES, preferencesFromWire, useWorldPreferences, type WorldPreferences } from './config'
import type { WorldApi } from './worldClient'

const SAVE_DEBOUNCE_MS = 600

function fakeApi(setConfig: WorldApi['setConfig']): WorldApi {
  return {
    getConfig: vi.fn().mockResolvedValue(DEFAULT_PREFERENCES),
    setConfig,
    getLayout: vi.fn(),
    setLayout: vi.fn(),
    streamFeed: vi.fn(),
  } as unknown as WorldApi
}

describe('camera preference compatibility', () => {
  it('preserves explicit ambient off and defaults absent preferences to enabled', () => {
    const { ambientLife: _ambientLife, ...legacy } = DEFAULT_PREFERENCES
    expect(preferencesFromWire(legacy).ambientLife).toBe(true)
    expect(preferencesFromWire({ ...legacy, ambientLife: false }).ambientLife).toBe(false)
    expect(preferencesFromWire({ ...legacy, ambientLife: true }).ambientLife).toBe(true)
  })
  it('defaults old configurations to cursor zoom and accepts persisted center zoom', () => {
    const { zoomTarget: _zoomTarget, ...legacy } = DEFAULT_PREFERENCES
    expect(preferencesFromWire(legacy).zoomTarget).toBe('cursor')
    expect(preferencesFromWire({ ...legacy, zoomTarget: '' }).zoomTarget).toBe('cursor')
    expect(preferencesFromWire({ ...legacy, zoomTarget: 'center' }).zoomTarget).toBe('center')
  })
})

describe('world preference persistence state', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  async function settle() {
    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })
  }

  async function advanceSave() {
    await act(async () => {
      vi.advanceTimersByTime(SAVE_DEBOUNCE_MS)
      await Promise.resolve()
      await Promise.resolve()
    })
  }

  it('reports dirty then saved after the debounce while holding the edit', async () => {
    const setConfig = vi.fn().mockResolvedValue(DEFAULT_PREFERENCES)
    const { result } = renderHook(() => useWorldPreferences(fakeApi(setConfig), true))
    await settle()

    act(() => result.current.update({ showDiagnostics: true }))
    expect(result.current.saveState.status).toBe('dirty')
    expect(result.current.preferences.showDiagnostics).toBe(true)

    await advanceSave()
    expect(setConfig).toHaveBeenCalledTimes(1)
    expect(setConfig.mock.calls[0][0]).toMatchObject({ showDiagnostics: true })
    expect(result.current.saveState.status).toBe('saved')
  })

  it.each([
    [Code.Unavailable, 'unavailable'],
    [Code.AlreadyExists, 'conflict'],
    [Code.Internal, 'error'],
  ])('classifies save failure %s and keeps edits recoverable', async (code, expected) => {
    const setConfig = vi.fn().mockRejectedValue(new ConnectError('save failed', code))
    const { result } = renderHook(() => useWorldPreferences(fakeApi(setConfig), true))
    await settle()

    act(() => result.current.update({ showDiagnostics: true, zoomTarget: 'center' }))
    await advanceSave()

    expect(result.current.saveState.status).toBe(expected)
    expect(result.current.saveState.message).toBeTruthy()
    expect(result.current.preferences.showDiagnostics).toBe(true)
    expect(result.current.preferences.zoomTarget).toBe('center')
  })

  it('retry re-sends the latest edits after a failed save', async () => {
    const setConfig = vi
      .fn()
      .mockRejectedValueOnce(new ConnectError('down', Code.Unavailable))
      .mockResolvedValueOnce(DEFAULT_PREFERENCES)
    const { result } = renderHook(() => useWorldPreferences(fakeApi(setConfig), true))
    await settle()

    act(() => result.current.update({ ambientLife: false }))
    await advanceSave()
    expect(result.current.saveState.status).toBe('unavailable')

    await act(async () => {
      result.current.retry()
      await Promise.resolve()
      await Promise.resolve()
    })
    expect(setConfig).toHaveBeenCalledTimes(2)
    expect((setConfig.mock.calls[1][0] as WorldPreferences).ambientLife).toBe(false)
    expect(result.current.saveState.status).toBe('saved')
  })
})
