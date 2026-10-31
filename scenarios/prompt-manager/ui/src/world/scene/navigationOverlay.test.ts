import { describe, expect, it, vi } from 'vitest'
import { navigationOverlaySteps } from './navigationOverlay'
import { runCooperatively } from '../sim/cooperative'

describe('navigation diagnostic markers', () => {
  it('represents every cell at its world center with terrain height and walkability', async () => {
    const nav = { cols: 2, rows: 1, originX: 10, originZ: 20, cellSize: 2, walkable: new Uint8Array([1, 0]) }
    const result = await runCooperatively(navigationOverlaySteps(nav, (x, z) => x + z))
    expect(result.count).toBe(2)
    expect(result.walkable).toBe(1)
    expect(result.blocked).toBe(1)
    expect([...result.positions].filter((_, i) => i % 3 !== 1)).toEqual([11, 21, 13, 21])
    expect(result.positions[1]).toBeCloseTo(32.05)
    expect(result.positions[4]).toBeCloseTo(34.05)
    expect(result.colors[1]).toBe(1)
    expect(result.colors[3]).toBe(1)
    expect(result.bytes).toBe(48)
    expect([...nav.walkable]).toEqual([1, 0])
  })
  it('cancels before publishing markers and rejects malformed grids', async () => {
    const nav = { cols: 128, rows: 1, originX: 0, originZ: 0, cellSize: 1, walkable: new Uint8Array(128) }
    const controller = new AbortController()
    const height = vi.fn(() => 0)
    await expect(runCooperatively(navigationOverlaySteps(nav, height), {
      signal: controller.signal, yieldTask: () => { controller.abort(); return Promise.resolve() },
    })).rejects.toMatchObject({ name: 'AbortError' })
    expect(height).not.toHaveBeenCalled()
    await expect(runCooperatively(navigationOverlaySteps({ ...nav, rows: 2 }, height))).rejects.toThrow('Invalid navigation overlay grid')
  })
})
