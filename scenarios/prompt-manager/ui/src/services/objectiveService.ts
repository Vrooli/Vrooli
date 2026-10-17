/**
 * Objective Service - reads and mutations for the objective authority.
 *
 * The objective authority (objectives/v1) is the single owner of objective
 * definitions, team attachments, ordering, support relations and
 * acknowledgements. The team.json::objectivesServed declaration is an authored
 * import input, not a second authority, so readers must not resolve coverage
 * from it.
 *
 * All hooks share the owner validation and revision scheme. Mutations
 * invalidate the objective cache so the editor and any team reader converge on
 * one current state.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  api,
  type Objective,
  type ObjectiveAttachment,
  type ObjectiveAttachmentInput,
  type ObjectiveInput,
  type ObjectiveRelation,
  type ObjectiveValidation,
  type TeamObjectiveAttachment,
  type TeamObjectiveAttachments,
} from '@/lib/api'

export type {
  Objective,
  ObjectiveAttachment,
  ObjectiveAttachmentInput,
  ObjectiveInput,
  ObjectiveRelation,
  ObjectiveValidation,
  TeamObjectiveAttachment,
  TeamObjectiveAttachments,
}

/** Query keys for the objective authority, shared by every consumer. */
export const objectiveKeys = {
  all: ['pm-objectives'] as const,
  list: () => [...objectiveKeys.all, 'list'] as const,
  relations: () => [...objectiveKeys.all, 'relations'] as const,
  validation: () => [...objectiveKeys.all, 'validation'] as const,
  team: (teamId: string) => [...objectiveKeys.all, 'team', teamId] as const,
}

const readOptions = {
  staleTime: 30_000,
  retry: false,
  refetchOnWindowFocus: false,
} as const

/** List every objective in persisted global order. */
export function useObjectives() {
  return useQuery({
    queryKey: objectiveKeys.list(),
    queryFn: () => api.listObjectives(),
    ...readOptions,
  })
}

/** List every objective support edge. */
export function useObjectiveRelations() {
  return useQuery({
    queryKey: objectiveKeys.relations(),
    queryFn: () => api.listObjectiveRelations(),
    ...readOptions,
  })
}

/** Read the owner's validation findings for current objective state. */
export function useObjectiveValidation() {
  return useQuery({
    queryKey: objectiveKeys.validation(),
    queryFn: () => api.validateObjectives(),
    ...readOptions,
  })
}

/** Read a team's ordered attachments plus its links/order revision. */
export function useTeamAttachments(teamId: string) {
  return useQuery({
    queryKey: objectiveKeys.team(teamId),
    queryFn: () => api.listTeamAttachments(teamId),
    enabled: teamId.length > 0,
    ...readOptions,
  })
}

/**
 * Read a team's objective attachments for display.
 *
 * Kept as the team-page display read; the full attachment model (priority,
 * restatement state, revision) is available through useTeamAttachments.
 *
 * @param teamId - Team ID
 * @returns Query result for the team's authority attachments
 */
export function useTeamObjectiveAttachments(teamId: string) {
  return useQuery({
    queryKey: objectiveKeys.team(teamId),
    queryFn: () => api.listTeamAttachments(teamId),
    select: (data: TeamObjectiveAttachments): TeamObjectiveAttachment[] =>
      data.attachments.map(attachment => ({
        id: attachment.objectiveId,
        role: attachment.role,
        coverage: attachment.coverage,
        note: attachment.note,
        acknowledgedRevision: attachment.acknowledgedRevision,
      })),
    enabled: teamId.length > 0,
    ...readOptions,
  })
}

function useInvalidateObjectives() {
  const queryClient = useQueryClient()
  return () => { void queryClient.invalidateQueries({ queryKey: objectiveKeys.all }) }
}

/** Create or update one objective with the caller's expected meaning revision. */
export function useUpsertObjective() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { objective: ObjectiveInput; expectedMeaningRevision?: string }) =>
      api.upsertObjective(vars.objective, vars.expectedMeaningRevision ?? ''),
    onSuccess: invalidate,
  })
}

/** Delete an objective that no team or relation still references. */
export function useDeleteObjective() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { id: string; expectedMeaningRevision?: string }) =>
      api.deleteObjective(vars.id, vars.expectedMeaningRevision ?? ''),
    onSuccess: invalidate,
  })
}

/** Rewrite global objective order. */
export function useReorderObjectives() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (objectiveIds: string[]) => api.reorderObjectives(objectiveIds),
    onSuccess: invalidate,
  })
}

/** Link a team to an objective. */
export function useAttachObjective() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { attachment: ObjectiveAttachmentInput; expectedTeamRevision?: string }) =>
      api.attachObjective(vars.attachment, vars.expectedTeamRevision ?? ''),
    onSuccess: invalidate,
  })
}

/** Edit role, coverage or note on an existing team attachment. */
export function useUpdateAttachment() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { attachment: ObjectiveAttachmentInput; expectedTeamRevision?: string }) =>
      api.updateAttachment(vars.attachment, vars.expectedTeamRevision ?? ''),
    onSuccess: invalidate,
  })
}

/** Remove a team's link to an objective. */
export function useDetachObjective() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { objectiveId: string; teamId: string; expectedTeamRevision?: string }) =>
      api.detachObjective(vars.objectiveId, vars.teamId, vars.expectedTeamRevision ?? ''),
    onSuccess: invalidate,
  })
}

/** Rewrite a team's objective priority. */
export function useReorderTeamAttachments() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { teamId: string; objectiveIds: string[]; expectedTeamRevision?: string }) =>
      api.reorderTeamAttachments(vars.teamId, vars.objectiveIds, vars.expectedTeamRevision ?? ''),
    onSuccess: invalidate,
  })
}

/** Confirm a team has reconsidered an objective's current meaning revision. */
export function useAcknowledgeObjective() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { objectiveId: string; teamId: string; revision: string }) =>
      api.acknowledgeObjective(vars.objectiveId, vars.teamId, vars.revision),
    onSuccess: invalidate,
  })
}

/** Add an instrumental support edge. */
export function useAddObjectiveRelation() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { fromObjectiveId: string; toObjectiveId: string }) =>
      api.addObjectiveRelation(vars.fromObjectiveId, vars.toObjectiveId),
    onSuccess: invalidate,
  })
}

/** Remove a support edge. */
export function useDeleteObjectiveRelation() {
  const invalidate = useInvalidateObjectives()
  return useMutation({
    mutationFn: (vars: { fromObjectiveId: string; toObjectiveId: string }) =>
      api.deleteObjectiveRelation(vars.fromObjectiveId, vars.toObjectiveId),
    onSuccess: invalidate,
  })
}
