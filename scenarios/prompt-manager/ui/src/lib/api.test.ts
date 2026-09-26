import { describe, expect, it } from 'vitest'

import { toProtoJson } from './api'

describe('toProtoJson', () => {
  it('omits undefined fields throughout protobuf JSON request objects', () => {
    expect(toProtoJson({
      agent: {
        id: undefined,
        displayName: 'Workflow Agent',
        appearance: { avatar: undefined, color: '#123456' },
        capabilities: undefined,
        connectors: [],
      },
    })).toEqual({
      agent: {
        displayName: 'Workflow Agent',
        appearance: { color: '#123456' },
        connectors: [],
      },
    })
  })

  it('rejects undefined array entries instead of silently changing repeated fields', () => {
    expect(() => toProtoJson({ tags: ['stable', undefined] })).toThrow(
      'Undefined protobuf JSON array entry at $.tags[1]',
    )
  })
})
