import { describe, expect, it, vi } from 'vitest'
import { tuning } from '../config'
import { prepareTerrainMesh, type TerrainMeshInput } from '../sim/terrain/mesh'
import { prepareTerrainMeshAsync, TerrainMeshCache } from './terrainMesh'

function meshInput(): TerrainMeshInput {
    const cells = 257
    const input: TerrainMeshInput = {
      field: { cols: cells, rows: 1, radius: cells, cellSize: 1, originX: 0, originZ: 0, height: new Float32Array(cells), moisture: new Float32Array(cells) },
      surface: { normals: Float32Array.from({ length: cells * 3 }, (_, index) => index % 3 === 1 ? 1 : 0), colours: new Float64Array(cells * 3).fill(0.5), aoStrength: new Float64Array(cells).fill(1), wetShade: new Float64Array(cells).fill(1) },
      innerRadiusSetting: 10, profile: { terrainInnerRadius: 10, terrainCellScale: 1 }, visual: tuning.visual.terrain,
      weather: { terrainTintMix: 0, terrainTintVariation: 0 }, tint: [1, 1, 1], shadowTint: [0, 0, 0],
    }
    return input
}

describe('terrain mesh transport', () => {
  it('reuses matching inputs, evicts within budget and never detaches a borrowed mesh', async () => {
    const input = meshInput()
    const producer = vi.fn((value: TerrainMeshInput) => Promise.resolve(prepareTerrainMesh(value)))
    const cache = new TerrainMeshCache({ maxEntries: 1, prepare: producer })
    const first = await cache.prepare(input)
    const modes: string[] = []
    expect(await cache.prepare({ ...input }, { onMode: mode => modes.push(mode) })).toBe(first)
    expect(modes).toEqual(['cache'])
    expect(producer).toHaveBeenCalledTimes(1)
    await cache.prepare({ ...input, weather: { ...input.weather, terrainTintMix: 0.5 } })
    expect(cache.stats().evictions).toBe(1)
    expect(first.vertices.byteLength).toBeGreaterThan(0)
    await cache.prepare(input)
    expect(producer).toHaveBeenCalledTimes(3)
    expect(cache.stats().estimatedBytes).toBeLessThanOrEqual(cache.stats().budgetBytes)
    const tiny = new TerrainMeshCache({ budgetBytes: 1, prepare: producer })
    await tiny.prepare(input)
    expect(tiny.stats().entries).toBe(0)
  })

  it('does not admit a result cancelled before publication', async () => {
    const controller = new AbortController()
    const cache = new TerrainMeshCache({ prepare: input => {
      const mesh = prepareTerrainMesh(input)
      controller.abort(new Error('superseded mesh'))
      return Promise.resolve(mesh)
    } })
    await expect(cache.prepare(meshInput(), { signal: controller.signal })).rejects.toThrow('superseded mesh')
    expect(cache.stats().entries).toBe(0)
  })

  it('supports fallback and releases the worker without transferring caller arrays', async () => {
    const input = meshInput()
    const mesh = prepareTerrainMesh(input)
    vi.stubGlobal('Worker', undefined)
    try {
      const modes: string[] = []
      const fallback = await prepareTerrainMeshAsync(input, { onMode: mode => modes.push(mode) })
      expect(fallback).toEqual(mesh)
      expect(modes).toEqual(['fallback'])
      const cancelled = new AbortController()
      cancelled.abort(new Error('cancel before mesh worker'))
      await expect(prepareTerrainMeshAsync(input, { signal: cancelled.signal })).rejects.toThrow('cancel before mesh worker')
    } finally { vi.unstubAllGlobals() }
    const terminated = vi.fn()
    const posted = vi.fn()
    vi.stubGlobal('Worker', class {
      onmessage: ((event: { data: { kind: string; mesh: typeof mesh } }) => void) | null = null
      onerror = null
      terminate = terminated
      postMessage(value: TerrainMeshInput, transfers?: unknown) {
        posted(value, transfers)
        queueMicrotask(() => this.onmessage?.({ data: { kind: 'complete', mesh } }))
      }
    })
    try {
      expect(await prepareTerrainMeshAsync(input)).toBe(mesh)
      expect(posted).toHaveBeenCalledWith(input, undefined)
      expect(terminated).toHaveBeenCalledOnce()
    } finally { vi.unstubAllGlobals() }
    const failedTermination = vi.fn()
    vi.stubGlobal('Worker', class {
      onmessage = null
      onerror: ((event: { preventDefault(): void }) => void) | null = null
      terminate = failedTermination
      postMessage() { queueMicrotask(() => this.onerror?.({ preventDefault() {} })) }
    })
    try {
      const modes: string[] = []
      expect(await prepareTerrainMeshAsync(input, { onMode: mode => modes.push(mode) })).toEqual(mesh)
      expect(modes).toEqual(['worker', 'fallback'])
      expect(failedTermination).toHaveBeenCalledOnce()
      const controller = new AbortController()
      await expect(prepareTerrainMeshAsync(input, {
        signal: controller.signal,
        onMode: mode => { if (mode === 'fallback') controller.abort(new Error('cancel recovery')) },
      })).rejects.toThrow('cancel recovery')
      expect(failedTermination).toHaveBeenCalledTimes(2)
    } finally { vi.unstubAllGlobals() }
  })
})
