import { describe, expect, it, vi } from 'vitest'
import { resolveTerrain, scenes, tuning } from '../../config'
import { runCooperatively, sortSteps } from '../cooperative'
import { generateWorld, generateWorldSteps } from '../world'
import { buildTerrain, buildTerrainSteps } from '../terrain/field'
import { makeWorldInput } from './fixtures'
import { terraceSiteSteps } from '../layout/terrace'

describe('cooperative generation', () => {
  it.each([0, 1, 2, 127, 128, 129, 1025])('sorts %i entries stably without changing the input', async count => {
    const input = Array.from({ length: count }, (_, index) => ({ index, rank: (index * 73) % 17 }))
    const before = [...input]
    const compare = (a: typeof input[number], b: typeof input[number]) => a.rank - b.rank
    const actual = await runCooperatively(sortSteps(input, compare), { yieldTask: () => Promise.resolve() })
    expect(actual).toEqual([...input].sort(compare))
    expect(input).toEqual(before)
  })

  it('allows cancellation within a sort pass and retains its source', async () => {
    const input = Array.from({ length: 512 }, (_, index) => 512 - index)
    const before = [...input]
    const controller = new AbortController()
    const steps = sortSteps(input, (a, b) => a - b)
    await expect(runCooperatively(steps, {
      signal: controller.signal, yieldTask: () => Promise.resolve(),
      onProgress: event => { if (event.completed > input.length) controller.abort(new Error('cancel sort')) },
    })).rejects.toThrow('cancel sort')
    expect(input).toEqual(before)
    expect(steps.next().done).toBe(true)
  })
  it.each([1, 2, 257, 512])('preserves the upper median when terracing %i samples', async count => {
    const resolver = resolveTerrain(scenes.park, tuning)
    const height = Float32Array.from({ length: count }, (_, i) => 10 + ((i * 73) % 113) / 10)
    const expected = [...height].sort((a, b) => a - b)[Math.floor(count / 2)] ?? 0
    const field = { radius: count, cellSize: 1, cols: count, rows: 1, originX: 0, originZ: 0, height, moisture: new Float32Array(count) }
    const site = await runCooperatively(terraceSiteSteps(field, resolver, { position: [0, 0], rotation: 0, size: [count * 2, 2], height: 0 }), { yieldTask: () => Promise.resolve() })
    expect(site.height).toBe(Math.max(expected, resolver.at(0, 0).waterLevel + resolver.at(0, 0).padClearance))
    expect([...height].every(value => value === Math.fround(site.height))).toBe(true)
  })

  it('cancels during median computation before mutating terrain', async () => {
    const resolver = resolveTerrain(scenes.park, tuning)
    const field = buildTerrain({ seed: 7, tuning: resolver })
    const before = field.height.slice()
    const controller = new AbortController()
    const steps = terraceSiteSteps(field, resolver, { position: [0, 0], rotation: 0, size: [field.radius * 2, field.radius * 2], height: 0 })
    await expect(runCooperatively(steps, {
      signal: controller.signal, yieldTask: () => Promise.resolve(),
      onProgress: event => { if (event.operation === 'median' && event.completed >= 128) controller.abort(new Error('cancel median')) },
    })).rejects.toThrow('cancel median')
    expect(field.height).toEqual(before)
    expect(steps.next().done).toBe(true)
  })
  it('batches cheap checkpoints without scheduling a timer for every work item', async () => {
    const clock = vi.spyOn(performance, 'now').mockReturnValue(100)
    const yieldTask = vi.fn(() => Promise.resolve())
    function* cheapWork() { for (let i = 0; i < 300; i++) yield i; return 'complete' }
    try {
      expect(await runCooperatively(cheapWork(), { yieldTask })).toBe('complete')
      expect(yieldTask).toHaveBeenCalledTimes(1)
    } finally { clock.mockRestore() }
  })

  it.each(['park', 'office'] as const)('matches synchronous generated output for %s', async scene => {
    const input = makeWorldInput({ scene, teams: 1, agents: 2 })
    const progress: string[] = []
    const actual = await runCooperatively(generateWorldSteps(input, tuning), {
      yieldTask: () => Promise.resolve(),
      onProgress: event => { progress.push(event.stage); expect(event.completed).toBeLessThanOrEqual(event.total) },
    })
    expect(actual).toEqual(generateWorld(input, tuning))
    expect(progress).toContain('terrain')
    expect(progress).toContain('biomes')
    expect(progress).toContain('layout')
    expect(progress).toContain('indexing')
    expect(progress).toContain('navigation')
    expect(progress).toContain('paths')
  })
  it.each(['park', 'office'] as const)('cancels %s indexing before routing starts', async scene => {
    const controller = new AbortController()
    const steps = generateWorldSteps(makeWorldInput({ scene, teams: 1, agents: 2 }), tuning)
    const stages: string[] = []
    await expect(runCooperatively(steps, {
      signal: controller.signal, yieldTask: () => Promise.resolve(),
      onProgress: event => {
        stages.push(event.stage)
        if (event.stage === 'indexing') controller.abort(new Error('cancel indexing'))
      },
    })).rejects.toThrow('cancel indexing')
    expect(stages).toContain('indexing')
    expect(stages).not.toContain('routing')
    expect(steps.next().done).toBe(true)
  })
  it.each(['park', 'office'] as const)('cancels inside %s layout before dressing or publication', async scene => {
    const controller = new AbortController()
    const steps = generateWorldSteps(makeWorldInput({ scene, teams: 1, agents: 2 }), tuning)
    const stages: string[] = []
    let layoutCheckpoints = 0
    await expect(runCooperatively(steps, {
      signal: controller.signal, yieldTask: () => Promise.resolve(),
      onProgress: event => {
        stages.push(event.stage)
        if (event.stage === 'layout' && ++layoutCheckpoints === 10) controller.abort(new Error('cancel layout'))
      },
    })).rejects.toThrow('cancel layout')
    expect(layoutCheckpoints).toBe(10)
    expect(stages).not.toContain('dressing')
    expect(steps.next().done).toBe(true)
  })

  it('cancels after dressing begins without returning partial placements', async () => {
    const controller = new AbortController()
    const steps = generateWorldSteps(makeWorldInput({ teams: 1, agents: 2 }), tuning)
    let reachedDressing = false
    await expect(runCooperatively(steps, {
      signal: controller.signal, yieldTask: () => Promise.resolve(),
      onProgress: event => {
        if (event.stage === 'dressing' && event.completed > 0) {
          reachedDressing = true
          controller.abort(new Error('cancel dressing'))
        }
      },
    })).rejects.toThrow('cancel dressing')
    expect(reachedDressing).toBe(true)
    expect(steps.next().done).toBe(true)
  })

  it('closes a cancelled terrain iterator without producing a partial world', async () => {
    const input = makeWorldInput({ teams: 1, agents: 2 })
    const controller = new AbortController()
    const steps = generateWorldSteps(input, tuning)
    const progress = vi.fn(() => controller.abort(new Error('superseded')))
    await expect(runCooperatively(steps, { signal: controller.signal, onProgress: progress, yieldTask: () => Promise.resolve() })).rejects.toThrow('superseded')
    expect(progress).toHaveBeenCalledTimes(1)
    expect(steps.next().done).toBe(true)
  })
  it('does no work when already cancelled and closes on scheduler failure', async () => {
    const start = vi.fn()
    const close = vi.fn()
    function* fixture() {
      start()
      try { yield 1; return 2 } finally { close() }
    }
    const aborted = new AbortController()
    aborted.abort(new Error('cancelled'))
    await expect(runCooperatively(fixture(), { signal: aborted.signal })).rejects.toThrow('cancelled')
    expect(start).not.toHaveBeenCalled()
    await expect(runCooperatively(fixture(), { yieldTask: () => Promise.reject(new Error('scheduler failed')) })).rejects.toThrow('scheduler failed')
    expect(close).toHaveBeenCalledOnce()
  })
  it('uses row checkpoints and rejects unsafe terrain allocations before allocation', () => {
    const resolver = resolveTerrain(scenes.park, tuning)
    const steps = buildTerrainSteps({ seed: 1, tuning: resolver })
    const first = steps.next()
    expect(first.done).toBe(false)
    if (!first.done) expect(first.value.completed).toBe(1)
    steps.return(buildTerrain({ seed: 1, tuning: resolver }))
    const base = resolver.base()
    for (const overrides of [{ radius: Infinity }, { cellSize: 0 }, { radius: 1e10 }]) {
      const invalid = { ...resolver, base: () => ({ ...base, ...overrides }) }
      expect(() => buildTerrain({ seed: 1, tuning: invalid })).toThrow(/Terrain/)
    }
  })
})
