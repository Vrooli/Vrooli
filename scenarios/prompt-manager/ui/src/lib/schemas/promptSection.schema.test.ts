import { describe, expect, it } from 'vitest'
import { PromptSectionSchema } from './agent.schema'

describe('PromptSectionSchema', () => {
  it('accepts current backend section kinds', () => {
    const section = PromptSectionSchema.parse({
      kind: 'active-task-brief',
      label: 'Active Task Brief',
      content: '# Active Task Brief\n\nYou are running one prompt-manager heartbeat as `agent-1`.',
    })

    expect(section.kind).toBe('active-task-brief')
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
