import { describe, expect, it } from 'vitest'

import { HealthScoreSchema } from '../graph.schema'

describe('HealthScoreSchema', () => {
  it('defaults omitted zero-valued protobuf scores', () => {
    const score = HealthScoreSchema.parse({
      nodeId: 'cli:external-tool',
      factors: {},
      messages: [],
    })

    expect(score.score).toBe(0)
  })
})
