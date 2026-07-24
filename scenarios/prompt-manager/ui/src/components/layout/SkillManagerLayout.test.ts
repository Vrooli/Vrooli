import { describe, expect, it } from 'vitest'
import layoutSource from './SkillManagerLayout.tsx?raw'

describe('SkillManagerLayout new-skill lifecycle', () => {
  it('keeps New Skill client-only until an explicit save', () => {
    const createHandler = layoutSource.match(
      /const handleCreateNew = useCallback\([\s\S]*?\n  }, \[isMobile, navigate\]\)/
    )?.[0]
    const saveHandler = layoutSource.match(
      /const handleSaveCurrentSkill = useCallback\([\s\S]*?\n  }, \[creatingSkillDraft, createSkill, formState, navigate, saveCurrentSkill, showSaveResultToast\]\)/
    )?.[0]

    expect(createHandler).toBeDefined()
    expect(createHandler).toContain("navigate(skillDetailPath('new'))")
    expect(createHandler).not.toContain('createSkill(')

    expect(saveHandler).toBeDefined()
    expect(saveHandler).toContain('if (creatingSkillDraft)')
    expect(saveHandler).toContain('await createSkill(')
  })
})
