import { describe, expect, it } from 'vitest'
import { PromptSectionSchema } from './agent.schema'

describe('PromptSectionSchema', () => {
  it('accepts current backend section kinds', () => {
    const section = PromptSectionSchema.parse({
      kind: 'execution-brief',
      label: 'Execution Brief',
      content: '# Execution Brief\n\nMember: `agent-1`',
    })

    expect(section.kind).toBe('execution-brief')
    expect(section.sourcePath).toBe('')
  })

  it('accepts future backend section kinds', () => {
    const section = PromptSectionSchema.parse({
      kind: 'future-section-kind',
      label: 'Future Section',
      content: 'content',
    })

    expect(section.kind).toBe('future-section-kind')
  })
})
