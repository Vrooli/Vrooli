/**
 * Tests for graphStore health score fallback chain.
 *
 * Verifies that selectEffectiveHealthScores returns health scores
 * in the correct priority order:
 * 1. healthScoreOverride (config preview)
 * 2. graph.graph.healthScores (full graph)
 * 3. standaloneHealthScores (lightweight fetch)
 * 4. [] (no data)
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useGraphStore, selectEffectiveHealthScores } from './graphStore'
import type { HealthScore } from '@/lib/schemas'

// Mock the graph service
vi.mock('@/services/graphService', () => ({
  getGraph: vi.fn(),
  getGraphHealth: vi.fn(),
  regenerateGraph: vi.fn(),
}))

function makeScore(nodeId: string, score: number): HealthScore {
  return {
    nodeId,
    score,
    factors: {},
    messages: [],
  }
}

const standaloneScores = [makeScore('s1', 0.5)]
const graphScores = [makeScore('g1', 0.8)]
const overrideScores = [makeScore('o1', 0.3)]

describe('selectEffectiveHealthScores', () => {
  beforeEach(() => {
    useGraphStore.setState({
      graph: null,
      standaloneHealthScores: null,
      healthScoreOverride: null,
    })
  })

  it('returns empty array when no health data is available', () => {
    const result = selectEffectiveHealthScores(useGraphStore.getState())
    expect(result).toEqual([])
  })

  it('returns standalone scores when graph is not loaded', () => {
    useGraphStore.setState({ standaloneHealthScores: standaloneScores })

    const result = selectEffectiveHealthScores(useGraphStore.getState())
    expect(result).toEqual(standaloneScores)
  })

  it('returns graph scores over standalone scores', () => {
    useGraphStore.setState({
      standaloneHealthScores: standaloneScores,
      graph: {
        generatedAt: '2026-01-01',
        graph: {
          nodes: [],
          edges: [],
          healthScores: graphScores,
        },
      },
    })

    const result = selectEffectiveHealthScores(useGraphStore.getState())
    expect(result).toEqual(graphScores)
  })

  it('returns override scores over graph and standalone scores', () => {
    useGraphStore.setState({
      standaloneHealthScores: standaloneScores,
      graph: {
        generatedAt: '2026-01-01',
        graph: {
          nodes: [],
          edges: [],
          healthScores: graphScores,
        },
      },
      healthScoreOverride: overrideScores,
    })

    const result = selectEffectiveHealthScores(useGraphStore.getState())
    expect(result).toEqual(overrideScores)
  })
})

describe('fetchHealthScores', () => {
  beforeEach(() => {
    useGraphStore.setState({
      graph: null,
      standaloneHealthScores: null,
      healthScoreOverride: null,
    })
  })

  it('populates standaloneHealthScores from API', async () => {
    const { getGraphHealth } = await import('@/services/graphService')
    const mockGetGraphHealth = vi.mocked(getGraphHealth)
    mockGetGraphHealth.mockResolvedValueOnce(standaloneScores)

    await useGraphStore.getState().fetchHealthScores()

    expect(useGraphStore.getState().standaloneHealthScores).toEqual(standaloneScores)
  })

  it('does not overwrite standaloneHealthScores on null response', async () => {
    useGraphStore.setState({ standaloneHealthScores: standaloneScores })

    const { getGraphHealth } = await import('@/services/graphService')
    const mockGetGraphHealth = vi.mocked(getGraphHealth)
    mockGetGraphHealth.mockResolvedValueOnce(null)

    await useGraphStore.getState().fetchHealthScores()

    expect(useGraphStore.getState().standaloneHealthScores).toEqual(standaloneScores)
  })

  it('does not throw on API error', async () => {
    const { getGraphHealth } = await import('@/services/graphService')
    const mockGetGraphHealth = vi.mocked(getGraphHealth)
    mockGetGraphHealth.mockRejectedValueOnce(new Error('Network error'))

    await expect(useGraphStore.getState().fetchHealthScores()).resolves.toBeUndefined()
  })
})
