import { Suspense, StrictMode, type ReactNode } from 'react'
import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { BufferGeometry, Float32BufferAttribute, Group, Mesh, MeshStandardMaterial } from 'three'
import { preloadProp, preloadProps, usePropParts } from './loader'
import { preparedAssets } from './cache'
import { allProps } from './registry'

const source = vi.hoisted(() => ({ scene: null as unknown }))
const loading = vi.hoisted(() => ({ load: vi.fn() }))
vi.mock('three/examples/jsm/loaders/GLTFLoader.js', () => ({ GLTFLoader: class { setMeshoptDecoder() { return this }; loadAsync() { return loading.load() } } }))

describe('prepared asset React lifetimes', () => {
  beforeEach(() => { loading.load.mockImplementation(() => Promise.resolve(source)) })
  it('bounds concurrent decodes and stops admitting records after cancellation', async () => {
    const base = allProps()[0]
    if (!base) throw new Error('Missing fixture')
    const scene = new Group()
    const geometry = new BufferGeometry()
    geometry.setAttribute('position', new Float32BufferAttribute([0, 0, 0, 1, 0, 0, 0, 1, 0], 3))
    scene.add(new Mesh(geometry, new MeshStandardMaterial()))
    const pending: Array<(value: { scene: Group }) => void> = []
    loading.load.mockImplementation(() => new Promise(resolve => pending.push(resolve)))
    const controller = new AbortController()
    const result = preloadProps(Array.from({ length: 8 }, (_, index) => ({ ...base, contentHash: `queue-fixture-${index}` })), controller.signal)
    const rejected = expect(result).rejects.toMatchObject({ name: 'AbortError' })
    await waitFor(() => expect(pending).toHaveLength(4))
    controller.abort()
    for (const resolve of pending) resolve({ scene })
    await rejected
    expect(pending).toHaveLength(4)
  })

  it('does not acquire resources for a suspended render that never commits', async () => {
    preparedAssets.trim(0)
    const base = allProps()[0]
    if (!base) throw new Error('Missing shipped fixture')
    const abandonedScene = new Group()
    const geometry = new BufferGeometry()
    geometry.setAttribute('position', new Float32BufferAttribute([0, 0, 0, 1, 0, 0, 0, 1, 0], 3))
    abandonedScene.add(new Mesh(geometry, new MeshStandardMaterial()))
    source.scene = abandonedScene
    const pending = new Promise<void>(() => undefined)
    await preloadProp({ ...base, contentHash: 'abandoned-fixture' })
    preparedAssets.trim(0)
    const before = preparedAssets.stats()
    const view = renderHook(() => {
      usePropParts({ ...base, contentHash: 'abandoned-fixture' })
      // React's Suspense contract uses a thrown thenable to abandon this render.
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw pending
    }, { wrapper: ({ children }: { children: ReactNode }) => <Suspense fallback={null}>{children}</Suspense> })
    view.unmount()
    expect(preparedAssets.stats()).toEqual(before)
  })

  it('shares committed strict-mode consumers, releases on unmount, and reuses a warm return', async () => {
    preparedAssets.trim(0)
    const scene = new Group()
    const geometry = new BufferGeometry()
    geometry.setAttribute('position', new Float32BufferAttribute([0, 0, 0, 1, 0, 0, 0, 1, 0], 3))
    scene.add(new Mesh(geometry, new MeshStandardMaterial()))
    source.scene = scene
    const base = allProps()[0]
    if (!base) throw new Error('Missing shipped fixture')
    const record = { ...base, contentHash: 'strict-mode-fixture' }
    await preloadProp(record)
    const wrapper = ({ children }: { children: ReactNode }) => <StrictMode>{children}</StrictMode>
    const first = renderHook(() => usePropParts(record), { wrapper })
    const second = renderHook(() => usePropParts(record), { wrapper })
    const parts = first.result.current
    expect(parts).toHaveLength(1)
    expect(second.result.current).toBe(parts)
    expect(preparedAssets.stats().references).toBe(2)
    first.unmount()
    second.unmount()
    expect(preparedAssets.stats().references).toBe(0)
    const returned = renderHook(() => usePropParts(record), { wrapper })
    expect(returned.result.current).toBe(parts)
    returned.unmount()
    preparedAssets.trim(0)
    expect(preparedAssets.stats().residentAssets).toBe(0)
  })
})
