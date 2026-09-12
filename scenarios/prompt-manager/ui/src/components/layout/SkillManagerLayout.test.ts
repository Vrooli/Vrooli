import { describe, expect, it } from 'vitest'
import layoutSource from './SkillManagerLayout.tsx?raw'

describe('SkillManagerLayout new-skill lifecycle', () => {
  it('keeps New Skill client-only until an explicit save', () => {
    const createHandler = layoutSource.match(
      /const handleCreateNew = useCallback\([\s\S]*?\n {2}}, \[isMobile, navigate\]\)/
    )?.[0]
    const saveHandler = layoutSource.match(
      /const handleSaveCurrentSkill = useCallback\([\s\S]*?\n {2}}, \[creatingSkillDraft, createSkill, formState, navigate, saveCurrentSkill, showSaveResultToast\]\)/
    )?.[0]

    expect(createHandler).toBeDefined()
    expect(createHandler).toContain("navigate(skillDetailPath('new'))")
    expect(createHandler).not.toContain('createSkill(')
    expect(layoutSource).toContain("content: '# New Skill\\n\\nEnter your skill content here...'")

    expect(saveHandler).toBeDefined()
    expect(saveHandler).toContain('if (creatingSkillDraft)')
    expect(saveHandler).toContain('await createSkill(')
    expect(saveHandler).toContain("movePromptState('new', created.id)")
    expect(saveHandler).toContain('markAsSaved(created.id, created)')
    expect(saveHandler).not.toContain("removePrompt('new')")
    expect(layoutSource).toContain('isDirty={isDirty || creatingSkillDraft}')
    expect(layoutSource).toContain('if ((isDirty || creatingSkillDraft) && validation.valid)')
  })
})
