import { describe, expect, it } from 'vitest'
import { PromptSectionSchema } from './agent.schema'

describe('PromptSectionSchema', () => {
  it('accepts current backend section kinds', () => {
    const sectionKinds = [
      ['active-task-brief', 'Active Task Brief'],
      ['team-operating-policy', 'Operating Policy'],
      ['task-reminder', 'Task Reminder'],
    ] as const

    for (const [kind, label] of sectionKinds) {
      const section = PromptSectionSchema.parse({
        kind,
        label,
        content: `# ${label}\n\nPrompt section content.`,
      })

      expect(section.kind).toBe(kind)
      expect(section.sourcePath).toBe('')
    }
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
