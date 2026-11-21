import type { Skill } from '@/types'

/** Imported skills are third-party material, never Vrooli-authored guidance. */
export function isThirdPartySkill(skill: Pick<Skill, 'origin'>): boolean {
  return skill.origin?.kind === 'imported'
}
