import { describe, expect, it } from 'vitest'
import { clusterLabels, labelWorldSize } from './clusters'
import { resolveCollisions, type LabelRect } from './collision'

function rect(id: string, x: number, y: number, priority = 1, distance = 10): LabelRect {
  return { id, x, y, width: 60, height: 16, priority, distance }
}

describe('resolveCollisions', () => {
  it('keeps only the top-priority label per overlap group', () => {
    const rects: LabelRect[] = []
    for (let i = 0; i < 50; i += 1) rects.push(rect(`a${i}`, 100 + (i % 5), 100 + (i % 3), i === 7 ? 5 : 1))
    const visible = resolveCollisions(rects, { paddingPx: 4, budget: 100 })
    expect([...visible]).toEqual(['a7'])
  })

  it('shows non-overlapping labels and respects the budget by priority', () => {
    const rects = [rect('far', 0, 0, 1), rect('mid', 300, 0, 3), rect('near', 600, 0, 5), rect('x', 900, 0, 2)]
    expect([...resolveCollisions(rects, { paddingPx: 0, budget: 10 })].sort()).toEqual(['far', 'mid', 'near', 'x'])
    expect([...resolveCollisions(rects, { paddingPx: 0, budget: 2 })].sort()).toEqual(['mid', 'near'])
  })

  it('pinned labels always show and win overlaps', () => {
    const rects = [rect('a', 0, 0, 9), rect('b', 2, 2, 1)]
    const visible = resolveCollisions(rects, { paddingPx: 0, budget: 0, pinned: new Set(['b']) })
    expect([...visible]).toEqual(['b'])
  })

  it('nearer labels win ties', () => {
    const visible = resolveCollisions([rect('far', 0, 0, 1, 50), rect('near', 1, 1, 1, 5)], { paddingPx: 0, budget: 5 })
    expect([...visible]).toEqual(['near'])
  })
})

describe('clusterLabels', () => {
  const members = [
    { id: 'a', roomId: 'r1', x: 0, z: 0 },
    { id: 'b', roomId: 'r1', x: 2, z: 0 },
    { id: 'c', roomId: 'r2', x: 10, z: 0 },
    { id: 'd', x: 5, z: 5 },
  ]

  it('keeps individual labels below the collapse distance', () => {
    expect(clusterLabels(members, 10, 30)).toEqual({ individual: ['a', 'b', 'c', 'd'], clusters: [] })
  })

  it('collapses rooms with several members past the distance, leaving singles and the unassigned alone', () => {
    const result = clusterLabels(members, 40, 30)
    expect(result.individual.sort()).toEqual(['c', 'd'])
    expect(result.clusters).toEqual([{ roomId: 'r1', count: 2, x: 1, z: 0 }])
  })

  it('with 100 actors in one room only one cluster label remains', () => {
    const many = Array.from({ length: 100 }, (_, i) => ({ id: `m${i}`, roomId: 'big', x: i, z: 0 }))
    const result = clusterLabels(many, 100, 30)
    expect(result.individual).toEqual([])
    expect(result.clusters[0]?.count).toBe(100)
  })
})

describe('labelWorldSize', () => {
  it('scales with distance and clamps the pixel size', () => {
    const near = labelWorldSize(10, 32, 1000, 14, 11, 20)
    const far = labelWorldSize(20, 32, 1000, 14, 11, 20)
    expect(far).toBeCloseTo(near * 2, 9)
    expect(labelWorldSize(10, 32, 1000, 40, 11, 20)).toBeCloseTo(labelWorldSize(10, 32, 1000, 20, 11, 20), 9)
    expect(labelWorldSize(10, 32, 1000, 2, 11, 20)).toBeCloseTo(labelWorldSize(10, 32, 1000, 11, 11, 20), 9)
  })
})
