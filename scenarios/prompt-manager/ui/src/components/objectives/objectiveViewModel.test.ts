import { describe, expect, it } from 'vitest'
import {
  classLabel,
  coverageLabel,
  moveItem,
  normalizeClass,
  orderObjectives,
  roleLabel,
  toAttachmentViewModel,
  toObjectiveViewModel,
} from './objectiveViewModel'
import type { Objective, ObjectiveAttachment } from '@/lib/api'

const objective: Objective = {
  id: 'win-market',
  title: 'Win the market',
  class: 'terminal',
  evidenceSource: 'revenue',
  hasEvidence: true,
  globalOrder: 0,
  meaningRevision: 'rev-1',
}

const attachment: ObjectiveAttachment = {
  objectiveId: 'win-market',
  teamId: 'marketing-crew',
  role: 'primary',
  coverage: 'partial',
  priority: 0,
  acknowledgedRevision: 'rev-1',
  restatementPending: false,
}

describe('objective view models', () => {
  it('normalizes the owner class and labels', () => {
    expect(normalizeClass('terminal')).toBe('terminal')
    expect(normalizeClass('weird')).toBe('unknown')
    expect(classLabel('instrumental')).toBe('Instrumental means')
    expect(roleLabel('supporting')).toBe('Supporting')
    expect(roleLabel(undefined)).toBe('Role unspecified')
    expect(coverageLabel('full')).toBe('Full coverage')
  })

  it('maps an objective and drops empty optional fields', () => {
    expect(toObjectiveViewModel({ ...objective, evidenceSource: '', gapMarker: '' })).toEqual({
      id: 'win-market',
      title: 'Win the market',
      class: 'terminal',
      classLabel: 'Terminal end',
      evidenceSource: undefined,
      hasEvidence: true,
      gapMarker: undefined,
      globalOrder: 0,
      meaningRevision: 'rev-1',
    })
  })

  it('orders objectives by persisted order with a stable id tiebreak', () => {
    const ordered = orderObjectives([
      toObjectiveViewModel({ ...objective, id: 'z', globalOrder: 1 }),
      toObjectiveViewModel({ ...objective, id: 'a', globalOrder: 1 }),
      toObjectiveViewModel({ ...objective, id: 'first', globalOrder: 0 }),
    ])
    expect(ordered.map(o => o.id)).toEqual(['first', 'a', 'z'])
  })

  it('links an attachment to its objective context', () => {
    const model = toAttachmentViewModel(attachment, new Map([['win-market', toObjectiveViewModel(objective)]]))
    expect(model.roleLabel).toBe('Primary')
    expect(model.coverageLabel).toBe('Partial coverage')
    expect(model.objective?.title).toBe('Win the market')
  })

  it('moves an item and refuses out-of-range moves', () => {
    expect(moveItem(['a', 'b', 'c'], 0, 1)).toEqual(['b', 'a', 'c'])
    expect(moveItem(['a', 'b', 'c'], 2, 1)).toEqual(['a', 'b', 'c'])
    expect(moveItem(['a', 'b', 'c'], 0, -1)).toEqual(['a', 'b', 'c'])
    expect(moveItem(['a', 'b', 'c'], 1, 1)).toEqual(['a', 'c', 'b'])
  })
})
