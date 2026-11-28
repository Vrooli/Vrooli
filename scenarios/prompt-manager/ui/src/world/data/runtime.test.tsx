import { act, renderHook, waitFor } from '@/test-utils/renderWithProviders'
import { describe, expect, it, vi } from 'vitest'
import { makeWorldInput } from '../sim/__tests__/fixtures'
import { useWorldRuntime } from './runtime'
import { generateWorld } from '../sim/world'
import { tuning, withTuningOverride } from '../config'
import type { GeneratedWorld } from '../sim/model'
import type { GenerationJob } from './generation'

const worker = vi.hoisted(() => ({ mode: 'worker', generate: vi.fn(), dispose: vi.fn() }))
vi.mock('./generationClient', () => ({ createGenerationClient: () => worker }))
const feed = vi.hoisted(() => ({ start: vi.fn(), stop: vi.fn() }))
vi.mock('./feed', () => ({ startFeed: feed.start }))


describe('runtime structural identity', () => {
  it('keeps the feed attached for presentation edits and replaces it for data edits', async () => {
    worker.generate.mockClear().mockImplementation((job: GenerationJob) => Promise.resolve(generateWorld(job.input, job.tuning)))
    feed.start.mockReset().mockReturnValue({ stop: feed.stop })
    feed.stop.mockReset()
    const first = withTuningOverride({ data: { pollIntervalMs: 1500 } })
    const options = { ...makeWorldInput({ teams: 1, agents: 2 }), tuning: first, live: true, step: false, treeVariants: 0 }
    const view = renderHook(props => useWorldRuntime(props), { initialProps: options })
    await waitFor(() => expect(feed.start).toHaveBeenCalledTimes(1))
    const store = view.result.current.store
    const next = withTuningOverride({ data: { pollIntervalMs: 1500 }, camera: { fov: 65 } }, tuning, first)
    view.rerender({ ...options, tuning: next })
    expect(feed.start).toHaveBeenCalledTimes(1)
    expect(feed.stop).not.toHaveBeenCalled()
    const changed = withTuningOverride({ data: { pollIntervalMs: 1600 }, camera: { fov: 65 } }, tuning, next)
    view.rerender({ ...options, tuning: changed })
    expect(feed.start).toHaveBeenCalledTimes(2)
    expect(feed.stop).toHaveBeenCalledTimes(1)
    expect(view.result.current.store).toBe(store)
    expect(worker.generate).toHaveBeenCalledTimes(1)
    view.unmount()
    expect(feed.stop).toHaveBeenCalledTimes(2)
  })

  it('starts assets and generation concurrently but waits for both before publication', async () => {
    let finishGeneration: ((world: GeneratedWorld) => void) | undefined
    let finishAssets: (() => void) | undefined
    worker.generate.mockImplementation(() => new Promise<GeneratedWorld>(resolve => { finishGeneration = resolve }))
    const prepareAssets = vi.fn(() => new Promise<void>(resolve => { finishAssets = resolve }))
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const view = renderHook(() => useWorldRuntime({ ...input, prepareAssets, live: false, step: false, treeVariants: 0 }))
    await waitFor(() => { expect(finishGeneration).toBeDefined(); expect(finishAssets).toBeDefined() })
    expect(view.result.current.store).toBeNull()
    await act(async () => { finishGeneration?.(generateWorld(input, tuning)); await Promise.resolve() })
    expect(view.result.current.store).toBeNull()
    expect(view.result.current.progress?.stage).toBe('assets')
    await act(async () => { finishAssets?.(); await Promise.resolve() })
    await waitFor(() => expect(view.result.current.store).not.toBeNull())
    view.unmount()
  })

  it('retains the active store through delayed or failed destination preparation and retries', async () => {
    worker.generate.mockImplementation((job: GenerationJob) => Promise.resolve(generateWorld(job.input, job.tuning)))
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const prepareDestination = vi.fn(() => Promise.resolve())
    const options = { ...input, prepareDestination, live: false, step: false, treeVariants: 0 }
    const view = renderHook(props => useWorldRuntime(props), { initialProps: options })
    await waitFor(() => expect(view.result.current.store).not.toBeNull())
    const original = view.result.current.store
    let rejectPreparation: ((error: Error) => void) | undefined
    prepareDestination.mockImplementationOnce(() => new Promise<void>((_resolve, reject) => { rejectPreparation = reject }))
    view.rerender({ ...options, seed: input.seed + 1 })
    await waitFor(() => expect(rejectPreparation).toBeDefined())
    expect(view.result.current.store).toBe(original)
    expect(view.result.current.progress?.stage).toBe('presentation')
    await act(async () => { rejectPreparation?.(new Error('destination mesh failed')); await Promise.resolve() })
    await waitFor(() => expect(view.result.current.error).toBe('destination mesh failed'))
    expect(view.result.current.store).toBe(original)
    expect(window.__worldSim?.preparation().state).toBe('failed')
    act(() => view.result.current.retry())
    await waitFor(() => expect(view.result.current.store).not.toBe(original))
    expect(view.result.current.store?.getState().seed).toBe(input.seed + 1)
    view.unmount()
  })

  it('returns to a cached scene without starting another worker generation', async () => {
    worker.generate.mockClear().mockImplementation((job: GenerationJob) => Promise.resolve(generateWorld(job.input, job.tuning)))
    const options = { ...makeWorldInput({ teams: 1, agents: 2 }), live: false, step: false, treeVariants: 0 }
    const view = renderHook(props => useWorldRuntime(props), { initialProps: options })
    await waitFor(() => expect(view.result.current.store).not.toBeNull())
    const originalTerrain = view.result.current.store?.getState().terrain
    view.rerender({ ...options, scene: 'office' })
    await waitFor(() => expect(view.result.current.store?.getState().scene).toBe('office'))
    view.rerender(options)
    await waitFor(() => expect(view.result.current.store?.getState().scene).toBe('park'))
    expect(view.result.current.store?.getState().terrain).toBe(originalTerrain)
    expect(worker.generate).toHaveBeenCalledTimes(2)
    expect(window.__worldSim?.generation()).toMatchObject({ source: 'cache', cache: { hits: 1, misses: 2, entries: 2 } })
    view.unmount()
  })

  it('adopts roster growth into the existing live store with a new commit epoch', async () => {
    worker.generate.mockImplementation((job: GenerationJob) => Promise.resolve(generateWorld(job.input, job.tuning)))
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const options = { ...input, live: false, step: false, treeVariants: 0 }
    const view = renderHook(props => useWorldRuntime(props), { initialProps: options })
    await waitFor(() => expect(view.result.current.store).not.toBeNull())
    const original = view.result.current.store
    const epoch = view.result.current.epoch
    view.rerender({ ...options, ...makeWorldInput({ teams: 1, agents: 3 }) })
    await waitFor(() => expect(view.result.current.epoch).toBeGreaterThan(epoch))
    expect(view.result.current.store).toBe(original)
    expect(original?.getState().actorOrder).toHaveLength(3)
    view.unmount()
  })

  it('keeps the current scene when a replacement fails and allows retry', async () => {
    worker.generate.mockImplementation((job: GenerationJob) => Promise.resolve(generateWorld(job.input, job.tuning)))
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const options = { ...input, live: false, step: false, treeVariants: 0 }
    const view = renderHook(props => useWorldRuntime(props), { initialProps: options })
    await waitFor(() => expect(view.result.current.store).not.toBeNull())
    const original = view.result.current.store
    worker.generate.mockRejectedValueOnce(new Error('worker failed'))
    view.rerender({ ...options, scene: 'office' })
    await waitFor(() => expect(view.result.current.error).toBe('worker failed'))
    expect(view.result.current.store).toBe(original)
    act(() => view.result.current.retry())
    await waitFor(() => expect(view.result.current.store?.getState().scene).toBe('office'))
    expect(view.result.current.error).toBeNull()
    view.unmount()
  })

  it('discards completion from an obsolete effect even when its transport ignores disposal', async () => {
    const pending: Array<(world: GeneratedWorld) => void> = []
    worker.generate.mockImplementation(() => new Promise<GeneratedWorld>(resolve => pending.push(resolve)))
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const options = { ...input, live: false, step: false, treeVariants: 0 }
    const view = renderHook(props => useWorldRuntime(props), { initialProps: options })
    await waitFor(() => expect(pending).toHaveLength(1))
    view.rerender({ ...options, seed: input.seed + 1 })
    await waitFor(() => expect(pending).toHaveLength(2))
    await act(async () => { pending[0]?.(generateWorld(input, tuning)); await Promise.resolve() })
    expect(view.result.current.store).toBeNull()
    await act(async () => { pending[1]?.(generateWorld({ ...input, seed: input.seed + 1 }, tuning)); await Promise.resolve() })
    expect(view.result.current.store?.getState().seed).toBe(input.seed + 1)
    expect(window.__worldSim?.generation().cache.entries).toBe(1)
    view.unmount()
  })

  it('keeps the running store for presentation changes and regenerates for structural changes', async () => {
    worker.generate.mockImplementation((job: GenerationJob) => Promise.resolve(generateWorld(job.input, job.tuning)))
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const options = { ...input, live: false, step: false, treeVariants: 0 }
    const view = renderHook(props => useWorldRuntime(props), { initialProps: options })
    expect(view.result.current.store).toBeNull()
    await waitFor(() => expect(view.result.current.store).not.toBeNull())
    const original = view.result.current.store
    if (!original) throw new Error('Expected prepared world')
    const terrain = original.getState().terrain
    const originalEpoch = view.result.current.epoch
    expect(window.__worldSim?.generation().count).toBe(1)
    view.rerender({ ...options, agents: options.agents.map(agent => ({ ...agent, name: 'New display name', skillCount: 12 })) })
    expect(view.result.current.store).toBe(original)
    expect(original.getState().terrain).toBe(terrain)
    expect(view.result.current.epoch).toBe(originalEpoch)
    expect(window.__worldSim?.generation().count).toBe(1)
    expect(Object.values(original.getState().actors).every(actor => actor.name === 'New display name')).toBe(true)
    view.rerender({ ...options, seed: options.seed + 1 })
    expect(view.result.current.store).toBe(original)
    expect(view.result.current.preparing).toBe(true)
    await waitFor(() => expect(view.result.current.store).not.toBe(original))
    expect(view.result.current.epoch).toBeGreaterThan(originalEpoch)
    expect(window.__worldSim?.generation()).toMatchObject({ count: 2, reason: 'terrain-input' })
    expect(view.result.current.store?.getState().terrain).not.toBe(terrain)
    view.unmount()
  })
})


it('advances an invited agent even when the demo roster is otherwise frozen', async () => {
  worker.generate.mockImplementation((job: GenerationJob) => Promise.resolve(generateWorld(job.input, job.tuning)))
  const view = renderHook(() => useWorldRuntime({ ...makeWorldInput({ teams: 0, agents: 1 }), live: false, step: false, treeVariants: 0 }))
  try {
    await waitFor(() => expect(view.result.current.store).not.toBeNull())
    const store = view.result.current.store
    if (!store) throw new Error('World did not become ready')
    const id = store.getState().actorOrder[0]
    if (!id) throw new Error('Missing fixture actor')
    const actor = store.getState().actors[id]
    if (!actor) throw new Error('Missing fixture actor')
    const before = [...actor.position]
    store.setVisitorConversation({ agentId: id, position: [actor.position[0], actor.position[1] + 4], yaw: 0 })
    await waitFor(() => expect(store.getState().actors[id]?.position).not.toEqual(before), { timeout: 3000 })
    store.setVisitorConversation(undefined)
    const stopped = store.getState().tick
    await new Promise(resolve => setTimeout(resolve, 150))
    expect(store.getState().tick).toBe(stopped)
  } finally { view.unmount() }
})
