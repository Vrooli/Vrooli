/**
 * ObjectiveAuthorityEditor - hook-wired container for the reusable editor.
 *
 * It resolves the owner reads into view models once and adapts the typed
 * objective service mutations into the editor callbacks. Team containers pass
 * `scope="team"` with a team id; the global settings container passes
 * `scope="global"`. Neither implements its own objective CRUD surface.
 */

import { useMemo } from 'react'
import {
  useAcknowledgeObjective,
  useAddObjectiveRelation,
  useAttachObjective,
  useDeleteObjective,
  useDeleteObjectiveRelation,
  useDetachObjective,
  useObjectiveRelations,
  useObjectiveValidation,
  useObjectives,
  useReorderObjectives,
  useReorderTeamAttachments,
  useTeamAttachments,
  useUpdateAttachment,
  useUpsertObjective,
} from '@/services/objectiveService'
import { ObjectiveEditor, type ObjectiveEditorCallbacks, type ObjectiveEditorProps } from './ObjectiveEditor'
import { orderObjectives, toAttachmentViewModel, toObjectiveViewModel } from './objectiveViewModel'

export interface ObjectiveAuthorityEditorProps {
  scope?: ObjectiveEditorProps['scope']
  teamId?: string
  className?: string
}

export function ObjectiveAuthorityEditor({ scope = 'global', teamId = '' }: ObjectiveAuthorityEditorProps) {
  const objectivesQuery = useObjectives()
  const relationsQuery = useObjectiveRelations()
  const validationQuery = useObjectiveValidation()
  const attachmentsQuery = useTeamAttachments(teamId)

  const upsert = useUpsertObjective()
  const removeObjective = useDeleteObjective()
  const reorderObjectives = useReorderObjectives()
  const attach = useAttachObjective()
  const updateAttachment = useUpdateAttachment()
  const detach = useDetachObjective()
  const reorderTeam = useReorderTeamAttachments()
  const acknowledge = useAcknowledgeObjective()
  const addRelation = useAddObjectiveRelation()
  const deleteRelation = useDeleteObjectiveRelation()

  const objectives = useMemo(
    () => orderObjectives((objectivesQuery.data ?? []).map(toObjectiveViewModel)),
    [objectivesQuery.data],
  )
  const objectivesById = useMemo(() => new Map(objectives.map(objective => [objective.id, objective])), [objectives])
  const attachments = useMemo(
    () => (attachmentsQuery.data?.attachments ?? []).map(attachment => toAttachmentViewModel(attachment, objectivesById)),
    [attachmentsQuery.data, objectivesById],
  )

  const readingAttachments = scope === 'team' && teamId !== '' && attachmentsQuery.isLoading
  const loading = objectivesQuery.isLoading || readingAttachments
  const error = objectivesQuery.isError
    ? 'the objective list could not be read'
    : scope === 'team' && attachmentsQuery.isError
      ? 'this team’s objective links could not be read'
      : null

  const reload = () => {
    void objectivesQuery.refetch()
    void relationsQuery.refetch()
    void validationQuery.refetch()
    if (scope === 'team' && teamId !== '') void attachmentsQuery.refetch()
  }

  const callbacks: ObjectiveEditorCallbacks = {
    onCreateObjective: async input => { await upsert.mutateAsync({ objective: input }) },
    onUpdateObjective: async (input, expectedMeaningRevision) => { await upsert.mutateAsync({ objective: input, expectedMeaningRevision }) },
    onDeleteObjective: async (id, expectedMeaningRevision) => { await removeObjective.mutateAsync({ id, expectedMeaningRevision }) },
    onReorderObjectives: async orderedIds => { await reorderObjectives.mutateAsync(orderedIds) },
    onAttach: async (input, expectedTeamRevision) => { await attach.mutateAsync({ attachment: input, expectedTeamRevision }) },
    onUpdateAttachment: async (input, expectedTeamRevision) => { await updateAttachment.mutateAsync({ attachment: input, expectedTeamRevision }) },
    onDetach: async (objectiveId, expectedTeamRevision) => { await detach.mutateAsync({ objectiveId, teamId, expectedTeamRevision }) },
    onReorderTeamAttachments: async (orderedIds, expectedTeamRevision) => { await reorderTeam.mutateAsync({ teamId, objectiveIds: orderedIds, expectedTeamRevision }) },
    onAcknowledge: async (objectiveId, revision) => { await acknowledge.mutateAsync({ objectiveId, teamId, revision }) },
    onAddRelation: async (fromObjectiveId, toObjectiveId) => { await addRelation.mutateAsync({ fromObjectiveId, toObjectiveId }) },
    onDeleteRelation: async (fromObjectiveId, toObjectiveId) => { await deleteRelation.mutateAsync({ fromObjectiveId, toObjectiveId }) },
  }

  return <ObjectiveEditor
    scope={scope}
    teamId={teamId}
    objectives={objectives}
    relations={relationsQuery.data ?? []}
    validation={validationQuery.data}
    attachments={attachments}
    attachmentRevision={attachmentsQuery.data?.attachmentRevision ?? ''}
    loading={loading}
    error={error}
    onReload={reload}
    callbacks={callbacks}
  />
}
