import { act, renderHook, waitFor } from '@testing-library/react'
import { create } from '@bufbuild/protobuf'
import { WorldLayoutSchema, type WorldLayout } from '@vrooli/proto-types/prompt-manager/v1/world/world_pb'
import { describe, expect, it, vi } from 'vitest'
import { useLayoutPersistence } from './layout'
import { createWorldApi, type WorldLayoutInit } from './worldClient'
import type { SceneId } from '../config'

const layout = (scene: string, x: number) => create(WorldLayoutSchema, { scene, overrides: [{ placeId: 'room:team', position: { x, z: 0 } }] })

describe('scene layout persistence', () => {
  it('initializes imported synthetic layout without reading or saving live preferences', () => {
    const api = { ...createWorldApi(), getLayout: vi.fn(), setLayout: vi.fn() }
    const imported = { scene: 'park' as const, overrides: [{ placeId: 'room:demo-team-0', position: [15, 15] as const }] }
    const view = renderHook(() => useLayoutPersistence('park', false, 10, api, imported))
    expect(view.result.current.loaded).toBe(true)
    expect(view.result.current.overrides).toEqual(imported.overrides)
    act(() => view.result.current.save([{ placeId: 'room:demo-team-0', position: [20, 20] }]))
    expect(view.result.current.overrides[0]?.position).toEqual([20, 20])
    view.unmount()
    expect(api.getLayout).not.toHaveBeenCalled()
    expect(api.setLayout).not.toHaveBeenCalled()
  })
  it('keeps failed loads unready and retries them explicitly', async () => {
    const getLayout = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValue(layout('park', 12))
    const api = { ...createWorldApi(), getLayout }
    const view = renderHook(() => useLayoutPersistence('park', true, 10, api))
    await waitFor(() => expect(view.result.current.error).toBe('offline'))
    expect(view.result.current.loaded).toBe(false)
    act(() => view.result.current.retry())
    await waitFor(() => expect(view.result.current.loaded).toBe(true))
    expect(view.result.current.error).toBeNull()
    expect(view.result.current.overrides[0]?.position).toEqual([12, 0])
    view.unmount()
  })

  it('retains failed saves and retries the edited value', async () => {
    const setLayout = vi.fn().mockRejectedValueOnce(new Error('save offline')).mockResolvedValue(layout('park', 42))
    const api = { ...createWorldApi(), getLayout: vi.fn().mockResolvedValue(layout('park', 0)), setLayout }
    const view = renderHook(() => useLayoutPersistence('park', true, 10, api))
    await waitFor(() => expect(view.result.current.loaded).toBe(true))
    act(() => view.result.current.save([{ placeId: 'room:team', position: [42, 0] }]))
    expect(view.result.current.saving).toBe(true)
    await waitFor(() => expect(view.result.current.error).toBe('save offline'))
    expect(view.result.current.saving).toBe(false)
    expect(view.result.current.overrides[0]?.position).toEqual([42, 0])
    act(() => view.result.current.retry())
    await waitFor(() => expect(setLayout).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(view.result.current.saving).toBe(false))
    expect(view.result.current.error).toBeNull()
    expect(setLayout.mock.calls[1]?.[0]).toMatchObject({ overrides: [{ position: { x: 42, z: 0 } }] })
    view.unmount()
  })

  it('ignores an older save failure after a newer edit has been queued', async () => {
    let rejectFirst: ((error: Error) => void) | undefined
    const setLayout = vi.fn().mockImplementationOnce(() => new Promise<WorldLayout>((_resolve, reject) => { rejectFirst = reject })).mockResolvedValue(layout('park', 2))
    const api = { ...createWorldApi(), getLayout: vi.fn().mockResolvedValue(layout('park', 0)), setLayout }
    const view = renderHook(() => useLayoutPersistence('park', true, 10, api))
    await waitFor(() => expect(view.result.current.loaded).toBe(true))
    act(() => view.result.current.save([{ placeId: 'room:team', position: [1, 0] }]))
    await waitFor(() => expect(rejectFirst).toBeDefined())
    act(() => view.result.current.save([{ placeId: 'room:team', position: [2, 0] }]))
    await act(async () => { rejectFirst?.(new Error('old failure')); await Promise.resolve() })
    expect(view.result.current.error).toBeNull()
    await waitFor(() => expect(view.result.current.saving).toBe(false))
    expect(view.result.current.overrides[0]?.position).toEqual([2, 0])
    view.unmount()
  })

  it('discards a departed scene response that arrives after the current scene loaded', async () => {
    let finishPark: ((value: WorldLayout) => void) | undefined
    const api = { ...createWorldApi(), getLayout: vi.fn((scene: string) => scene === 'office' ? Promise.resolve(layout(scene, 20)) : new Promise<WorldLayout>(resolve => { finishPark = resolve })) }
    const view = renderHook(({ scene }: { scene: SceneId }) => useLayoutPersistence(scene, true, 10, api), { initialProps: { scene: 'park' as SceneId } })
    await waitFor(() => expect(finishPark).toBeDefined())
    view.rerender({ scene: 'office' })
    await waitFor(() => expect(view.result.current.loaded).toBe(true))
    await act(async () => { finishPark?.(layout('park', 10)); await Promise.resolve() })
    expect(view.result.current.overrides[0]?.position).toEqual([20, 0])
    view.unmount()
  })

  it('never exposes old scene overrides while loading a replacement', async () => {
    let resolveOffice: ((value: WorldLayout) => void) | undefined
    const api = { ...createWorldApi(), getLayout: vi.fn((scene: string) => scene === 'park' ? Promise.resolve(layout(scene, 10)) : new Promise<WorldLayout>(resolve => { resolveOffice = resolve })) }
    const view = renderHook(({ scene }: { scene: SceneId }) => useLayoutPersistence(scene, true, 10, api), { initialProps: { scene: 'park' as SceneId } })
    await waitFor(() => expect(view.result.current.loaded).toBe(true))
    expect(view.result.current.overrides[0]?.position).toEqual([10, 0])
    view.rerender({ scene: 'office' })
    expect(view.result.current.loaded).toBe(false)
    expect(view.result.current.overrides).toEqual([])
    await waitFor(() => expect(resolveOffice).toBeDefined())
    await act(async () => { resolveOffice?.(layout('office', 20)); await Promise.resolve() })
    expect(view.result.current.overrides[0]?.position).toEqual([20, 0])
    view.unmount()
  })

  it('flushes each scene edit under its original key and serializes writes', async () => {
    let finishFirst: ((value: WorldLayout) => void) | undefined
    const setLayout = vi.fn((value: WorldLayoutInit) => new Promise<WorldLayout>(resolve => {
      if (!finishFirst) finishFirst = resolve
      else resolve(create(WorldLayoutSchema, value))
    }))
    const api = { ...createWorldApi(), getLayout: vi.fn((scene: string) => Promise.resolve(layout(scene, 0))), setLayout }
    const view = renderHook(({ scene }: { scene: SceneId }) => useLayoutPersistence(scene, true, 1000, api), { initialProps: { scene: 'park' as SceneId } })
    await waitFor(() => expect(view.result.current.loaded).toBe(true))
    act(() => view.result.current.save([{ placeId: 'room:team', position: [10, 0] }]))
    act(() => view.result.current.save([{ placeId: 'room:team', position: [11, 0] }]))
    view.rerender({ scene: 'office' })
    await waitFor(() => expect(setLayout).toHaveBeenCalledTimes(1))
    expect(setLayout.mock.calls[0]?.[0]).toMatchObject({ scene: 'park', overrides: [{ position: { x: 11, z: 0 } }] })
    act(() => view.result.current.save([{ placeId: 'room:team', position: [22, 0] }]))
    view.unmount()
    expect(setLayout).toHaveBeenCalledTimes(1)
    await act(async () => { finishFirst?.(layout('park', 11)); await Promise.resolve() })
    await waitFor(() => expect(setLayout).toHaveBeenCalledTimes(2))
    expect(setLayout.mock.calls[1]?.[0]).toMatchObject({ scene: 'office', overrides: [{ position: { x: 22, z: 0 } }] })
  })

  it('does not let a late load overwrite a newer local edit', async () => {
    let finishLoad: ((value: WorldLayout) => void) | undefined
    const api = { ...createWorldApi(), getLayout: vi.fn(() => new Promise<WorldLayout>(resolve => { finishLoad = resolve })), setLayout: vi.fn((value: WorldLayoutInit) => Promise.resolve(create(WorldLayoutSchema, value))) }
    const view = renderHook(() => useLayoutPersistence('park', true, 1000, api))
    await waitFor(() => expect(finishLoad).toBeDefined())
    act(() => view.result.current.save([{ placeId: 'room:team', position: [30, 0] }]))
    await act(async () => { finishLoad?.(layout('park', 1)); await Promise.resolve() })
    expect(view.result.current.overrides[0]?.position).toEqual([30, 0])
    view.unmount()
  })
})
