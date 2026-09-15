import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import type { NavGrid, WorldBounds, Place } from '../model'
import { PathCache, findPath, findPathSteps, lineOfSight, smoothPath } from '../nav/astar'
import { buildNavGrid, buildNavGridSteps, cellToWorld, isWalkable, nearestWalkable, worldToCell } from '../nav/grid'
import { moveAlongPath, turnToward, wrapAngle } from '../motion/move'
import { runCooperatively } from '../cooperative'
import { spacePoint } from '../layout/spaces'
import { makeWorld } from './fixtures'

function openGrid(cols: number, rows: number, cellSize = 1): NavGrid {
  return { cellSize, cols, rows, originX: 0, originZ: 0, walkable: new Uint8Array(cols * rows).fill(1) }
}

describe('nav grid', () => {
  it('shrinks path capacity immediately while preserving the most recently used paths', () => {
    const cache = new PathCache(4)
    for (const key of ['a', 'b', 'c', 'd']) cache.set(key, [[1, 2]])
    const retained = cache.get('a')
    cache.resize(2)
    expect(cache.size).toBe(2)
    expect(cache.get('b')).toBeUndefined()
    expect(cache.get('c')).toBeUndefined()
    expect(cache.get('a')).toBe(retained)
    expect(cache.get('d')).toEqual([[1, 2]])
    cache.resize(3)
    cache.set('e', [[3, 4]])
    expect(cache.size).toBe(3)
    cache.resize(0)
    expect(cache.size).toBe(0)
    cache.set('f', [[1, 2]])
    expect(cache.size).toBe(0)
    expect(() => cache.resize(-1)).toThrow(RangeError)
    expect(() => cache.resize(.5)).toThrow(RangeError)
  })
  it('yields during a long search and never caches a cancelled path', async () => {
    const grid = openGrid(300, 300)
    const cache = new PathCache(2)
    const steps = findPathSteps(grid, [0.5, 0.5], [299.5, 299.5], cache)
    const controller = new AbortController()
    await expect(runCooperatively(steps, {
      signal: controller.signal, yieldTask: () => Promise.resolve(),
      onProgress: () => controller.abort(new Error('superseded path')),
    })).rejects.toThrow('superseded path')
    expect(cache.size).toBe(0)
    expect(steps.next().done).toBe(true)
    const path = await runCooperatively(findPathSteps(grid, [0.5, 0.5], [299.5, 299.5], cache), { yieldTask: () => Promise.resolve() })
    expect(path).toEqual(findPath(grid, [0.5, 0.5], [299.5, 299.5]))
    expect(cache.size).toBe(1)
  })

  it('yields inside one large obstacle before marking that obstacle complete', () => {
    const bounds: WorldBounds = { width: 32, depth: 32, center: [0, 0], footprint: { width: 32, depth: 32, center: [0, 0] }, outline: [] }
    const board: Place = { id: 'board', kind: 'board', position: [0, 0], size: [32, 32], rotation: 0, seats: [], label: 'Board' }
    const steps = buildNavGridSteps(bounds, [board], [], 1, 0.2)
    const first = steps.next()
    expect(first.done).toBe(false)
    if (!first.done) expect(first.value.completed).toBe(0)
    let result = steps.next()
    while (!result.done) result = steps.next()
    expect(result.value.walkable).toEqual(new Uint8Array(1024))
  })

  it('cancels between obstacles and rejects unsafe grid sizes before allocation', async () => {
    const bounds: WorldBounds = { width: 4, depth: 4, center: [0, 0], footprint: { width: 4, depth: 4, center: [0, 0] }, outline: [] }
    const desk: Place = { id: 'desk', kind: 'desk', position: [0, 0], size: [1, 1], rotation: 0, seats: [], label: 'Desk' }
    const steps = buildNavGridSteps(bounds, [desk], [], 1, 0.2)
    const controller = new AbortController()
    let checkpoints = 0
    await expect(runCooperatively(steps, {
      signal: controller.signal, yieldTask: () => Promise.resolve(),
      onProgress: () => { checkpoints++; controller.abort(new Error('changed destination')) },
    })).rejects.toThrow('changed destination')
    expect(checkpoints).toBe(1)
    expect(steps.next().done).toBe(true)
    expect(() => buildNavGrid(bounds, [], [], 0, 0.2)).toThrow(/Navigation/)
    expect(() => buildNavGrid({ ...bounds, width: 1e12 }, [], [], 1, 0.2)).toThrow(/allocation/)
    const distant = { ...desk, position: [1e9, 1e9] as const, size: [1e6, 1e6] as const }
    expect(buildNavGrid(bounds, [distant], [], 1, 0.2).walkable).toEqual(new Uint8Array(16).fill(1))
  })

  it('blocks campsite furniture and shelter walls while keeping the site entrance open', () => {
    const s = makeWorld({ teams: 1, agents: 3, treeVariants: 3 })
    const room = Object.values(s.places).find((p) => p.kind === 'room')
    const desk = Object.values(s.places).find((p) => p.kind === 'desk')
    const fire = s.places.hearth
    expect(room && desk && fire).toBeTruthy()
    if (!room || !desk || !fire) return
    expect(isWalkable(s.nav, desk.position)).toBe(false)
    expect(isWalkable(s.nav, fire.position)).toBe(false)
    const localPoint = (z: number): [number, number] => [room.position[0] + z * Math.sin(room.rotation), room.position[1] + z * Math.cos(room.rotation)]
    const shelter = room.space?.shelters[0]
    if (!shelter) throw new Error('Missing campsite shelter')
    expect(isWalkable(s.nav, spacePoint(room, [shelter.position[0], shelter.position[1] - shelter.size[1] / 2]))).toBe(false)
    expect(isWalkable(s.nav, localPoint(room.size[1] / 2))).toBe(true)
    const tree = s.decor.find((spot) => spot.kind === 'tree')
    if (tree) expect(isWalkable(s.nav, tree.position)).toBe(false)
    expect(isWalkable(s.nav, desk.seats[0]?.position ?? [0, 0])).toBe(true)
  })

  it('round-trips world and cell coordinates', () => {
    const grid = openGrid(10, 10, 0.5)
    const [c, r] = worldToCell(grid, [2.3, 4.9])
    expect([c, r]).toEqual([4, 9])
    expect(cellToWorld(grid, c, r)).toEqual([2.25, 4.75])
  })

  it('nearestWalkable finds a free cell next to a blocked one', () => {
    const grid = openGrid(5, 5)
    grid.walkable[12] = 0
    expect(nearestWalkable(grid, [2.5, 2.5], 3)).not.toEqual([2.5, 2.5])
    expect(isWalkable(grid, nearestWalkable(grid, [2.5, 2.5], 3) ?? [0, 0])).toBe(true)
    expect(nearestWalkable(grid, [1.5, 1.5], 3)).toEqual([1.5, 1.5])
  })
})

describe('A*', () => {
  it('routes around a wall and never crosses a blocked cell', () => {
    const grid = openGrid(20, 20)
    for (let r = 2; r < 18; r += 1) grid.walkable[r * 20 + 10] = 0
    const path = findPath(grid, [2.5, 10.5], [17.5, 10.5])
    expect(path).not.toBeNull()
    if (!path) return
    expect(path[path.length - 1]).toEqual([17.5, 10.5])
    for (let i = 1; i < path.length; i += 1) {
      const a = path[i - 1]
      const b = path[i]
      if (a && b) expect(lineOfSight(grid, a, b)).toBe(true)
    }
    expect(path.length).toBeGreaterThan(2)
  })

  it('returns null for an unreachable goal and a direct path for a trivial one', () => {
    const grid = openGrid(10, 10)
    for (let c = 0; c < 10; c += 1) grid.walkable[5 * 10 + c] = 0
    expect(findPath(grid, [1.5, 1.5], [1.5, 8.5])).toBeNull()
    expect(findPath(grid, [1.5, 1.5], [1.7, 1.6])).toEqual([[1.7, 1.6]])
  })

  it('smoothing keeps only the corners that need turning', () => {
    const grid = openGrid(10, 10)
    const straight = smoothPath(grid, [[0.5, 0.5], [1.5, 0.5], [2.5, 0.5], [3.5, 0.5]])
    expect(straight).toEqual([[0.5, 0.5], [3.5, 0.5]])
  })

  it('the LRU cache bounds itself and serves repeats', () => {
    const cache = new PathCache(2)
    const grid = openGrid(6, 6)
    findPath(grid, [0.5, 0.5], [4.5, 4.5], cache)
    findPath(grid, [0.5, 0.5], [4.5, 0.5], cache)
    findPath(grid, [0.5, 4.5], [4.5, 4.5], cache)
    expect(cache.size).toBe(2)
    const before = cache.size
    findPath(grid, [0.5, 4.5], [4.5, 4.5], cache)
    expect(cache.size).toBe(before)
  })
})

describe('motion', () => {
  it('ramps speed over accelSeconds and never exceeds the target speed', () => {
    const s = makeWorld({ teams: 1, agents: 1, treeVariants: 3 })
    const base = s.actors['agent-0-0']
    if (!base) throw new Error('missing')
    const a = { ...base }
    a.path = [[a.position[0] + 50, a.position[1]]]
    a.speed = 0
    const dt = tuning.sim.tickSeconds
    let maxSpeed = 0
    let ramped = 0
    for (let i = 0; i < 200; i += 1) {
      moveAlongPath(a, dt, tuning.sim)
      maxSpeed = Math.max(maxSpeed, a.speed)
      if (a.speed < tuning.sim.walkSpeed) ramped += 1
    }
    expect(maxSpeed).toBeLessThanOrEqual(tuning.sim.walkSpeed + 1e-9)
    expect(ramped * dt).toBeGreaterThanOrEqual(tuning.sim.accelSeconds - dt)
  })

  it('arrives exactly on the last waypoint and reports it once', () => {
    const s = makeWorld({ teams: 1, agents: 1, treeVariants: 3 })
    const base = s.actors['agent-0-0']
    if (!base) throw new Error('missing')
    const a = { ...base }
    const goal: [number, number] = [a.position[0] + 3, a.position[1] + 1]
    a.path = [goal]
    let arrivals = 0
    for (let i = 0; i < 500; i += 1) if (moveAlongPath(a, tuning.sim.tickSeconds, tuning.sim)) arrivals += 1
    expect(arrivals).toBe(1)
    expect(a.position).toEqual(goal)
    expect(a.speed).toBe(0)
  })

  it('turns toward the heading at most turnRate per second', () => {
    expect(turnToward(0, Math.PI / 2, 1, 0.5)).toBeCloseTo(0.5, 9)
    expect(turnToward(0, 0.1, 1, 0.5)).toBeCloseTo(0.1, 9)
    expect(wrapAngle(Math.PI * 3)).toBeCloseTo(Math.PI, 9)
  })
})
