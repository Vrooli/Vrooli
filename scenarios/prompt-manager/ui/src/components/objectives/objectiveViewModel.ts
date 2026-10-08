/**
 * Objective view models.
 *
 * The editor maps the generated owner messages into these view models once, so
 * components never re-derive labels, ordering or relationship context from raw
 * wire fields. Keeping the mapping here also makes the editor reusable from the
 * team container and the global container without either one owning a second
 * model.
 */

import type { Objective, ObjectiveAttachment } from '@/lib/api'

/** ObjectiveClass is the editor's normalized view of the owner class field. */
export type ObjectiveClass = 'terminal' | 'instrumental' | 'unknown'

/** ObjectiveViewModel is one objective prepared for display and editing. */
export interface ObjectiveViewModel {
  id: string
  title: string
  class: ObjectiveClass
  classLabel: string
  evidenceSource?: string
  hasEvidence: boolean
  gapMarker?: string
  globalOrder: number
  meaningRevision: string
}

/** AttachmentViewModel is one team attachment prepared for display and editing. */
export interface AttachmentViewModel {
  objectiveId: string
  teamId: string
  role?: string
  roleLabel: string
  coverage?: string
  coverageLabel: string
  note?: string
  priority: number
  acknowledgedRevision?: string
  attachmentRevision: string
  restatementPending: boolean
  objective?: ObjectiveViewModel
}

const classLabels: Record<ObjectiveClass, string> = {
  terminal: 'Terminal end',
  instrumental: 'Instrumental means',
  unknown: 'Unclassified',
}

/** normalizeClass narrows the owner class string into an editor class. */
export function normalizeClass(value: string | undefined): ObjectiveClass {
  return value === 'terminal' || value === 'instrumental' ? value : 'unknown'
}

/** classLabel returns the plain-language label for an objective class. */
export function classLabel(value: string | undefined): string {
  return classLabels[normalizeClass(value)]
}

/** roleLabel returns the plain-language label for an attachment role. */
export function roleLabel(role: string | undefined): string {
  switch (role) {
    case 'primary':
      return 'Primary'
    case 'supporting':
      return 'Supporting'
    default:
      return 'Role unspecified'
  }
}

/** coverageLabel returns the plain-language label for an attachment coverage. */
export function coverageLabel(coverage: string | undefined): string {
  switch (coverage) {
    case 'full':
      return 'Full coverage'
    case 'partial':
      return 'Partial coverage'
    default:
      return 'Coverage unspecified'
  }
}

/** toObjectiveViewModel maps one owner objective into its editor view model. */
export function toObjectiveViewModel(objective: Objective): ObjectiveViewModel {
  return {
    id: objective.id,
    title: objective.title,
    class: normalizeClass(objective.class),
    classLabel: classLabel(objective.class),
    evidenceSource: objective.evidenceSource || undefined,
    hasEvidence: objective.hasEvidence,
    gapMarker: objective.gapMarker || undefined,
    globalOrder: objective.globalOrder,
    meaningRevision: objective.meaningRevision,
  }
}

/** orderObjectives returns a sorted copy in persisted global order. */
export function orderObjectives(objectives: ObjectiveViewModel[]): ObjectiveViewModel[] {
  return [...objectives].sort((a, b) => (a.globalOrder !== b.globalOrder ? a.globalOrder - b.globalOrder : a.id.localeCompare(b.id)))
}

/**
 * toAttachmentViewModel maps one owner attachment, linking it to its objective
 * so the editor can show relationship, class and evidence context without a
 * second lookup at render time.
 */
export function toAttachmentViewModel(
  attachment: ObjectiveAttachment,
  objectivesById: Map<string, ObjectiveViewModel>,
): AttachmentViewModel {
  return {
    objectiveId: attachment.objectiveId,
    teamId: attachment.teamId,
    role: attachment.role,
    roleLabel: roleLabel(attachment.role),
    coverage: attachment.coverage,
    coverageLabel: coverageLabel(attachment.coverage),
    note: attachment.note,
    priority: attachment.priority,
    acknowledgedRevision: attachment.acknowledgedRevision,
    attachmentRevision: attachment.attachmentRevision ?? '',
    restatementPending: attachment.restatementPending,
    objective: objectivesById.get(attachment.objectiveId),
  }
}

/**
 * moveItem returns a new array with the item at `index` moved by `delta`
 * positions. It returns the original array unchanged when the move would fall
 * outside the list, so a caller can schedule a no-op without a special case.
 */
export function moveItem<T>(items: T[], index: number, delta: number): T[] {
  const target = index + delta
  if (index < 0 || index >= items.length || target < 0 || target >= items.length) return items
  const next = [...items]
  const [moved] = next.splice(index, 1)
  if (moved === undefined) return items
  next.splice(target, 0, moved)
  return next
}
