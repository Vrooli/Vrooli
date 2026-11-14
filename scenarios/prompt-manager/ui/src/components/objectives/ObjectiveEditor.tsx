/**
 * ObjectiveEditor - the reusable objective authority editor surface.
 *
 * The component is presentational: it maps already-built view models into
 * terminal/instrumental forms, relationship context, accessible ordering
 * controls and recovery states, then calls the callbacks it was given. The
 * ObjectiveAuthorityEditor container wires those callbacks to the typed
 * objective service, so the team container and the global settings container
 * mount the same editor instead of implementing another CRUD surface.
 */

import { useEffect, useMemo, useState } from 'react'
import type { ObjectiveAttachmentInput, ObjectiveInput, ObjectiveRelation, ObjectiveValidation } from '@/lib/api'
import {
  type AttachmentViewModel,
  type ObjectiveClass,
  type ObjectiveViewModel,
  coverageLabel,
  moveItem,
  orderObjectives,
  roleLabel,
} from './objectiveViewModel'
import { SortableList, SortableRow } from './sortableRows'

export interface ObjectiveEditorCallbacks {
  onCreateObjective?: (input: ObjectiveInput) => Promise<void>
  onUpdateObjective?: (input: ObjectiveInput, expectedMeaningRevision: string) => Promise<void>
  onDeleteObjective?: (id: string, expectedMeaningRevision: string) => Promise<void>
  onReorderObjectives?: (orderedIds: string[]) => Promise<void>
  onAttach?: (input: ObjectiveAttachmentInput, expectedTeamRevision: string) => Promise<void>
  onUpdateAttachment?: (input: ObjectiveAttachmentInput, expectedTeamRevision: string) => Promise<void>
  onDetach?: (objectiveId: string, expectedTeamRevision: string) => Promise<void>
  onReorderTeamAttachments?: (orderedIds: string[], expectedTeamRevision: string) => Promise<void>
  onAcknowledge?: (objectiveId: string, revision: string) => Promise<void>
  onAddRelation?: (fromObjectiveId: string, toObjectiveId: string) => Promise<void>
  onDeleteRelation?: (fromObjectiveId: string, toObjectiveId: string) => Promise<void>
}

export interface ObjectiveEditorProps {
  scope: 'global' | 'team'
  /** Required for team scope. */
  teamId?: string
  objectives: ObjectiveViewModel[]
  relations?: ObjectiveRelation[]
  validation?: ObjectiveValidation
  attachments?: AttachmentViewModel[]
  attachmentRevision?: string
  loading?: boolean
  error?: string | null
  /** Refetch the authority after a conflict or when the reader asks to recover. */
  onReload?: () => void
  callbacks?: ObjectiveEditorCallbacks
}

interface ObjectiveDraft {
  id: string
  title: string
  class: ObjectiveClass
  evidenceSource: string
  gapMarker: string
}

const emptyObjectiveDraft: ObjectiveDraft = { id: '', title: '', class: 'terminal', evidenceSource: '', gapMarker: '' }

interface AttachmentDraft {
  objectiveId: string
  role: string
  coverage: string
  note: string
}

const emptyAttachmentDraft: AttachmentDraft = { objectiveId: '', role: '', coverage: '', note: '' }

/** messageOf extracts a user-facing message without leaking a stack trace. */
function messageOf(cause: unknown): string {
  return cause instanceof Error && cause.message ? cause.message : 'The objective authority refused the change.'
}

export function ObjectiveEditor(props: ObjectiveEditorProps) {
  const {
    scope, teamId, relations = [], validation, attachments = [], attachmentRevision = '',
    loading = false, error = null, onReload, callbacks = {},
  } = props
  const objectives = useMemo(() => orderObjectives(props.objectives), [props.objectives])
  const byId = useMemo(() => new Map(objectives.map(objective => [objective.id, objective])), [objectives])

  const [objectiveDraft, setObjectiveDraft] = useState<ObjectiveDraft | null>(null)
  const [editingObjectiveId, setEditingObjectiveId] = useState<string | null>(null)
  const [attachmentDraft, setAttachmentDraft] = useState<AttachmentDraft | null>(null)
  const [editingAttachmentId, setEditingAttachmentId] = useState<string | null>(null)
  const [relationDraft, setRelationDraft] = useState({ fromObjectiveId: '', toObjectiveId: '' })
  const [busy, setBusy] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [conflict, setConflict] = useState(false)
  const [status, setStatus] = useState<string | null>(null)

  // A background owner refresh replaces view models. Ending a half-finished
  // create/edit form on refresh would discard the operator's unsaved input, so
  // the form is only reset by an explicit save, cancel or scope change.
  useEffect(() => {
    setObjectiveDraft(null)
    setEditingObjectiveId(null)
    setAttachmentDraft(null)
    setEditingAttachmentId(null)
    setFormError(null)
    setConflict(false)
    setStatus(null)
  }, [scope, teamId])

  const attachedIds = useMemo(() => new Set(attachments.map(attachment => attachment.objectiveId)), [attachments])
  const availableObjectives = useMemo(() => objectives.filter(objective => !attachedIds.has(objective.id)), [objectives, attachedIds])
  const instrumentalObjectives = useMemo(() => objectives.filter(objective => objective.class === 'instrumental'), [objectives])

  const run = async (action: () => Promise<void>, successMessage: string): Promise<boolean> => {
    setBusy(true)
    setFormError(null)
    setConflict(false)
    setStatus(null)
    try {
      await action()
      setStatus(successMessage)
      return true
    } catch (cause) {
      const message = messageOf(cause)
      setFormError(message)
      // A revision conflict is recoverable: the owner's current state can be
      // fetched and the operator can retry without losing the draft.
      if (/conflict|revision/i.test(message)) setConflict(true)
      return false
    } finally {
      setBusy(false)
    }
  }

  const startCreate = () => {
    setEditingObjectiveId(null)
    setObjectiveDraft({ ...emptyObjectiveDraft })
    setFormError(null)
    setConflict(false)
    setStatus(null)
  }

  const startEdit = (objective: ObjectiveViewModel) => {
    setEditingObjectiveId(objective.id)
    setObjectiveDraft({
      id: objective.id,
      title: objective.title,
      class: objective.class,
      evidenceSource: objective.evidenceSource ?? '',
      gapMarker: objective.gapMarker ?? '',
    })
    setFormError(null)
    setConflict(false)
    setStatus(null)
  }

  const submitObjective = () => {
    if (!objectiveDraft) return
    const input: ObjectiveInput = {
      id: objectiveDraft.id.trim(),
      title: objectiveDraft.title.trim(),
      class: objectiveDraft.class,
      evidenceSource: objectiveDraft.evidenceSource.trim() || undefined,
      gapMarker: objectiveDraft.gapMarker.trim() || undefined,
    }
    if (editingObjectiveId) {
      const current = byId.get(editingObjectiveId)
      void run(() => callbacks.onUpdateObjective?.(input, current?.meaningRevision ?? '') ?? Promise.resolve(), 'Objective saved.').then(saved => {
        if (!saved) return
        setObjectiveDraft(null)
        setEditingObjectiveId(null)
      })
    } else {
      void run(() => callbacks.onCreateObjective?.(input) ?? Promise.resolve(), 'Objective created.').then(saved => {
        if (!saved) return
        setObjectiveDraft(null)
      })
    }
  }

  const submitAttachment = () => {
    if (!attachmentDraft || !teamId) return
    const input: ObjectiveAttachmentInput = {
      objectiveId: attachmentDraft.objectiveId,
      teamId,
      role: attachmentDraft.role || undefined,
      coverage: attachmentDraft.coverage || undefined,
      note: attachmentDraft.note.trim() || undefined,
    }
    if (editingAttachmentId) {
      void run(() => callbacks.onUpdateAttachment?.(input, attachmentRevision) ?? Promise.resolve(), 'Objective link saved.').then(saved => {
        if (!saved) return
        setAttachmentDraft(null)
        setEditingAttachmentId(null)
      })
    } else {
      void run(() => callbacks.onAttach?.(input, attachmentRevision) ?? Promise.resolve(), 'Objective attached.').then(saved => {
        if (!saved) return
        setAttachmentDraft(null)
      })
    }
  }

  const moveObjective = (index: number, delta: number) => {
    const next = moveItem(objectives, index, delta)
    if (next === objectives) return
    void run(() => callbacks.onReorderObjectives?.(next.map(objective => objective.id)) ?? Promise.resolve(), 'Objective order saved.')
  }

  const moveAttachment = (index: number, delta: number) => {
    const next = moveItem(attachments, index, delta)
    if (next === attachments) return
    void run(() => callbacks.onReorderTeamAttachments?.(next.map(attachment => attachment.objectiveId), attachmentRevision) ?? Promise.resolve(), 'Objective priority saved.')
  }

  if (loading) {
    return <section aria-label="Objective authority" className="space-y-3 rounded-lg border border-border p-4">
      <p className="text-sm text-muted-foreground">Loading objective authority…</p>
    </section>
  }

  if (error) {
    return <section aria-label="Objective authority" className="space-y-3 rounded-lg border border-border p-4">
      <p role="alert" className="text-sm text-destructive">Objective authority is unavailable: {error}</p>
      {onReload ? <button type="button" onClick={onReload} className="rounded border border-border px-3 py-1.5 text-sm">Retry</button> : null}
    </section>
  }

  return <section aria-label="Objective authority" className="space-y-4 rounded-lg border border-border p-4">
    <header className="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h3 className="text-sm font-semibold">{scope === 'team' ? 'Objective commitments' : 'Objectives'}</h3>
        <p className="text-xs text-muted-foreground">
          {scope === 'team'
            ? 'Terminal ends and instrumental means this team commits to, in the priority its prompt uses.'
            : 'Operator ends and means, with global priority and support relationships.'}
        </p>
      </div>
      <div className="flex items-center gap-2">
        {validation ? <span className="text-xs text-muted-foreground" role="status">{validation.errors} error(s), {validation.warnings} warning(s)</span> : null}
        {scope === 'global' ? <button type="button" onClick={startCreate} className="rounded bg-primary px-3 py-1.5 text-sm text-primary-foreground">New objective</button> : null}
        {scope === 'team' && availableObjectives.length ? <button type="button" onClick={() => { setEditingAttachmentId(null); setAttachmentDraft({ ...emptyAttachmentDraft }); setFormError(null) }} className="rounded bg-primary px-3 py-1.5 text-sm text-primary-foreground">Link objective</button> : null}
      </div>
    </header>

    {formError ? <div role="alert" className="space-y-2 rounded border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
      <p>{formError}</p>
      {conflict ? <p>The owner state changed since this view was read. Reload to pick up the current revision, then retry. Your unsaved fields stay in the form.</p> : null}
      {conflict && onReload ? <button type="button" onClick={onReload} className="rounded border border-destructive/40 px-2 py-1 text-xs">Reload authority</button> : null}
    </div> : null}
    {status ? <p role="status" className="text-sm text-muted-foreground">{status}</p> : null}

    {objectiveDraft ? <ObjectiveForm
      draft={objectiveDraft}
      editing={editingObjectiveId !== null}
      busy={busy}
      onChange={setObjectiveDraft}
      onSubmit={submitObjective}
      onCancel={() => { setObjectiveDraft(null); setEditingObjectiveId(null) }}
    /> : null}

    {scope === 'global' ? <ObjectiveList
      objectives={objectives}
      busy={busy}
      onMove={moveObjective}
      onDragReorder={(from, to) => moveObjective(from, to - from)}
      onEdit={startEdit}
      onDelete={objective => void run(() => callbacks.onDeleteObjective?.(objective.id, objective.meaningRevision) ?? Promise.resolve(), 'Objective deleted.')}
    /> : null}

    {scope === 'team' ? <AttachmentList
      attachments={attachments}
      busy={busy}
      onMove={moveAttachment}
      onDragReorder={(from, to) => moveAttachment(from, to - from)}
      onEdit={attachment => {
        setEditingAttachmentId(attachment.objectiveId)
        setAttachmentDraft({
          objectiveId: attachment.objectiveId,
          role: attachment.role ?? '',
          coverage: attachment.coverage ?? '',
          note: attachment.note ?? '',
        })
        setFormError(null)
      }}
      onDetach={attachment => void run(() => callbacks.onDetach?.(attachment.objectiveId, attachmentRevision) ?? Promise.resolve(), 'Objective detached.')}
      onAcknowledge={attachment => void run(() => callbacks.onAcknowledge?.(attachment.objectiveId, attachment.objective?.meaningRevision ?? '') ?? Promise.resolve(), 'Objective acknowledged.')}
    /> : null}

    {scope === 'team' && attachmentDraft ? <AttachmentForm
      draft={attachmentDraft}
      candidates={editingAttachmentId ? objectives : availableObjectives}
      editing={editingAttachmentId !== null}
      busy={busy}
      onChange={setAttachmentDraft}
      onSubmit={submitAttachment}
      onCancel={() => { setAttachmentDraft(null); setEditingAttachmentId(null) }}
    /> : null}

    {scope === 'global' ? <RelationPanel
      relations={relations}
      instruments={instrumentalObjectives}
      objectives={objectives}
      draft={relationDraft}
      busy={busy}
      onDraftChange={setRelationDraft}
      onAdd={() => void run(async () => {
        await callbacks.onAddRelation?.(relationDraft.fromObjectiveId, relationDraft.toObjectiveId)
      }, 'Support relationship saved.').then(saved => {
        if (saved) setRelationDraft({ fromObjectiveId: '', toObjectiveId: '' })
      })}
      onDelete={relation => void run(() => callbacks.onDeleteRelation?.(relation.fromObjectiveId, relation.toObjectiveId) ?? Promise.resolve(), 'Support relationship removed.')}
    /> : null}

    {scope === 'global' && validation?.findings.length ? <div className="space-y-1 border-t border-border pt-3">
      <h4 className="text-sm font-semibold">Owner validation</h4>
      <ul className="space-y-1 text-xs">
        {validation.findings.map((finding, index) => <li key={`${finding.rule}-${finding.objectiveId ?? ''}-${finding.teamId ?? ''}-${index}`} className={finding.severity === 'error' ? 'text-destructive' : 'text-muted-foreground'}>
          <span className="font-medium">{finding.severity}</span> · {finding.objectiveId ?? 'authority'}{finding.teamId ? ` · ${finding.teamId}` : ''} · {finding.detail}
        </li>)}
      </ul>
    </div> : null}
  </section>
}

function ObjectiveForm(props: {
  draft: ObjectiveDraft
  editing: boolean
  busy: boolean
  onChange: (draft: ObjectiveDraft) => void
  onSubmit: () => void
  onCancel: () => void
}) {
  const { draft, editing, busy, onChange, onSubmit, onCancel } = props
  return <form className="space-y-3 rounded border border-border bg-muted/40 p-3" onSubmit={event => { event.preventDefault(); onSubmit() }}>
    <div className="grid gap-3 sm:grid-cols-2">
      <label className="space-y-1 text-sm">Objective ID
        <input aria-label="Objective ID" value={draft.id} disabled={editing || busy} onChange={event => onChange({ ...draft, id: event.target.value })} className="block w-full rounded border border-border bg-background p-2" />
      </label>
      <label className="space-y-1 text-sm">Title
        <input aria-label="Objective title" value={draft.title} disabled={busy} onChange={event => onChange({ ...draft, title: event.target.value })} className="block w-full rounded border border-border bg-background p-2" />
      </label>
      <label className="space-y-1 text-sm">Class
        <select aria-label="Objective class" value={draft.class} disabled={busy} onChange={event => onChange({ ...draft, class: event.target.value as ObjectiveClass })} className="block w-full rounded border border-border bg-background p-2">
          <option value="terminal">Terminal end</option>
          <option value="instrumental">Instrumental means</option>
        </select>
      </label>
      <label className="space-y-1 text-sm">Evidence source
        <input aria-label="Evidence source" value={draft.evidenceSource} disabled={busy} onChange={event => onChange({ ...draft, evidenceSource: event.target.value })} className="block w-full rounded border border-border bg-background p-2" />
      </label>
      <label className="space-y-1 text-sm sm:col-span-2">Gap marker
        <input aria-label="Gap marker" value={draft.gapMarker} disabled={busy} onChange={event => onChange({ ...draft, gapMarker: event.target.value })} className="block w-full rounded border border-border bg-background p-2" />
      </label>
    </div>
    <div className="flex items-center gap-2">
      <button type="submit" disabled={busy} className="rounded bg-primary px-3 py-1.5 text-sm text-primary-foreground disabled:opacity-50">{busy ? 'Saving…' : editing ? 'Save objective' : 'Create objective'}</button>
      <button type="button" onClick={onCancel} disabled={busy} className="rounded border border-border px-3 py-1.5 text-sm">Cancel</button>
    </div>
  </form>
}

function ObjectiveList(props: {
  objectives: ObjectiveViewModel[]
  busy: boolean
  onMove: (index: number, delta: number) => void
  onDragReorder: (from: number, to: number) => void
  onEdit: (objective: ObjectiveViewModel) => void
  onDelete: (objective: ObjectiveViewModel) => void
}) {
  const { objectives, busy, onMove, onDragReorder, onEdit, onDelete } = props
  if (!objectives.length) return <p className="text-sm text-muted-foreground">No objectives are defined yet. Create the first terminal end or instrumental means.</p>
  return <SortableList ids={objectives.map(objective => objective.id)} onReorder={onDragReorder}>
    <ol className="space-y-2">
      {objectives.map((objective, index) => <SortableRow key={objective.id} id={objective.id} className="rounded border border-border p-3" render={handle => (
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="flex items-start gap-2">
            <button type="button" {...handle.attributes} {...handle.listeners} aria-label={`Reorder ${objective.title}`} title="Drag or use arrow keys to reorder" className="cursor-grab rounded border border-border px-1.5 py-1 text-xs text-muted-foreground">⠿</button>
            <div>
              <p className="text-sm font-medium">{objective.title}</p>
              <p className="text-xs text-muted-foreground">{objective.id} · {objective.classLabel}{objective.hasEvidence ? ` · evidence: ${objective.evidenceSource}` : ' · no evidence source'}{objective.gapMarker ? ` · gap: ${objective.gapMarker}` : ''}</p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <button type="button" aria-label={`Move ${objective.title} up`} disabled={busy || index === 0} onClick={() => onMove(index, -1)} className="rounded border border-border px-2 py-1 text-xs disabled:opacity-40">Move up</button>
            <button type="button" aria-label={`Move ${objective.title} down`} disabled={busy || index === objectives.length - 1} onClick={() => onMove(index, 1)} className="rounded border border-border px-2 py-1 text-xs disabled:opacity-40">Move down</button>
            <button type="button" onClick={() => onEdit(objective)} disabled={busy} className="rounded border border-border px-2 py-1 text-xs">Edit</button>
            <button type="button" onClick={() => onDelete(objective)} disabled={busy} className="rounded border border-border px-2 py-1 text-xs text-destructive">Delete</button>
          </div>
        </div>
      )} />)}
    </ol>
  </SortableList>
}

function AttachmentList(props: {
  attachments: AttachmentViewModel[]
  busy: boolean
  onMove: (index: number, delta: number) => void
  onDragReorder: (from: number, to: number) => void
  onEdit: (attachment: AttachmentViewModel) => void
  onDetach: (attachment: AttachmentViewModel) => void
  onAcknowledge: (attachment: AttachmentViewModel) => void
}) {
  const { attachments, busy, onMove, onDragReorder, onEdit, onDetach, onAcknowledge } = props
  if (!attachments.length) return <p className="text-sm text-muted-foreground">This team has no objective commitments yet. Link a terminal end or instrumental means.</p>
  return <SortableList ids={attachments.map(attachment => attachment.objectiveId)} onReorder={onDragReorder}>
    <ol className="space-y-2">
      {attachments.map((attachment, index) => {
        const label = attachment.objective?.title ?? attachment.objectiveId
        return <SortableRow key={attachment.objectiveId} id={attachment.objectiveId} className="rounded border border-border p-3" render={handle => (
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div className="flex items-start gap-2">
              <button type="button" {...handle.attributes} {...handle.listeners} aria-label={`Reorder ${label}`} title="Drag or use arrow keys to reorder" className="cursor-grab rounded border border-border px-1.5 py-1 text-xs text-muted-foreground">⠿</button>
              <div>
                <p className="text-sm font-medium">{label}</p>
                <p className="text-xs text-muted-foreground">
                  {attachment.objectiveId}{attachment.objective ? ` · ${attachment.objective.classLabel}` : ''} · {roleLabel(attachment.role)} · {coverageLabel(attachment.coverage)}
                </p>
                {attachment.note ? <p className="text-xs text-muted-foreground">{attachment.note}</p> : null}
                {attachment.restatementPending ? <p className="text-xs text-destructive">Meaning changed; acknowledgement pending.</p> : null}
              </div>
            </div>
            <div className="flex items-center gap-1">
              <button type="button" aria-label={`Move ${label} up`} disabled={busy || index === 0} onClick={() => onMove(index, -1)} className="rounded border border-border px-2 py-1 text-xs disabled:opacity-40">Move up</button>
              <button type="button" aria-label={`Move ${label} down`} disabled={busy || index === attachments.length - 1} onClick={() => onMove(index, 1)} className="rounded border border-border px-2 py-1 text-xs disabled:opacity-40">Move down</button>
              <button type="button" onClick={() => onEdit(attachment)} disabled={busy} className="rounded border border-border px-2 py-1 text-xs">Edit</button>
              {attachment.restatementPending ? <button type="button" onClick={() => onAcknowledge(attachment)} disabled={busy} className="rounded border border-border px-2 py-1 text-xs">Acknowledge</button> : null}
              <button type="button" onClick={() => onDetach(attachment)} disabled={busy} className="rounded border border-border px-2 py-1 text-xs text-destructive">Detach</button>
            </div>
          </div>
        )} />
      })}
    </ol>
  </SortableList>
}

function AttachmentForm(props: {
  draft: AttachmentDraft
  candidates: ObjectiveViewModel[]
  editing: boolean
  busy: boolean
  onChange: (draft: AttachmentDraft) => void
  onSubmit: () => void
  onCancel: () => void
}) {
  const { draft, candidates, editing, busy, onChange, onSubmit, onCancel } = props
  return <form className="space-y-3 rounded border border-border bg-muted/40 p-3" onSubmit={event => { event.preventDefault(); onSubmit() }}>
    <div className="grid gap-3 sm:grid-cols-2">
      <label className="space-y-1 text-sm">Objective
        <select aria-label="Attachment objective" value={draft.objectiveId} disabled={editing || busy} onChange={event => onChange({ ...draft, objectiveId: event.target.value })} className="block w-full rounded border border-border bg-background p-2">
          <option value="">Select an objective</option>
          {candidates.map(objective => <option key={objective.id} value={objective.id}>{objective.title} ({objective.classLabel})</option>)}
        </select>
      </label>
      <label className="space-y-1 text-sm">Role
        <select aria-label="Attachment role" value={draft.role} disabled={busy} onChange={event => onChange({ ...draft, role: event.target.value })} className="block w-full rounded border border-border bg-background p-2">
          <option value="">Role unspecified</option>
          <option value="primary">Primary</option>
          <option value="supporting">Supporting</option>
        </select>
      </label>
      <label className="space-y-1 text-sm">Coverage
        <select aria-label="Attachment coverage" value={draft.coverage} disabled={busy} onChange={event => onChange({ ...draft, coverage: event.target.value })} className="block w-full rounded border border-border bg-background p-2">
          <option value="">Coverage unspecified</option>
          <option value="full">Full coverage</option>
          <option value="partial">Partial coverage</option>
        </select>
      </label>
      <label className="space-y-1 text-sm">Note
        <input aria-label="Attachment note" value={draft.note} disabled={busy} onChange={event => onChange({ ...draft, note: event.target.value })} className="block w-full rounded border border-border bg-background p-2" />
      </label>
    </div>
    <div className="flex items-center gap-2">
      <button type="submit" disabled={busy || !draft.objectiveId} className="rounded bg-primary px-3 py-1.5 text-sm text-primary-foreground disabled:opacity-50">{busy ? 'Saving…' : editing ? 'Save link' : 'Attach objective'}</button>
      <button type="button" onClick={onCancel} disabled={busy} className="rounded border border-border px-3 py-1.5 text-sm">Cancel</button>
    </div>
  </form>
}

function RelationPanel(props: {
  relations: ObjectiveRelation[]
  instruments: ObjectiveViewModel[]
  objectives: ObjectiveViewModel[]
  draft: { fromObjectiveId: string; toObjectiveId: string }
  busy: boolean
  onDraftChange: (draft: { fromObjectiveId: string; toObjectiveId: string }) => void
  onAdd: () => void
  onDelete: (relation: ObjectiveRelation) => void
}) {
  const { relations, instruments, objectives, draft, busy, onDraftChange, onAdd, onDelete } = props
  const byId = new Map(objectives.map(objective => [objective.id, objective]))
  const canAdd = draft.fromObjectiveId !== '' && draft.toObjectiveId !== '' && draft.fromObjectiveId !== draft.toObjectiveId
  return <div className="space-y-3 border-t border-border pt-3">
    <div>
      <h4 className="text-sm font-semibold">Support relationships</h4>
      <p className="text-xs text-muted-foreground">An instrumental means supports another objective. Support edges stay acyclic.</p>
    </div>
    {relations.length ? <ul className="space-y-1 text-sm">
      {relations.map(relation => <li key={`${relation.fromObjectiveId}->${relation.toObjectiveId}`} className="flex items-center justify-between gap-2 rounded border border-border px-3 py-2">
        <span>{byId.get(relation.fromObjectiveId)?.title ?? relation.fromObjectiveId} <span className="text-muted-foreground">supports</span> {byId.get(relation.toObjectiveId)?.title ?? relation.toObjectiveId}</span>
        <button type="button" onClick={() => onDelete(relation)} disabled={busy} className="rounded border border-border px-2 py-1 text-xs text-destructive">Remove</button>
      </li>)}
    </ul> : <p className="text-sm text-muted-foreground">No support relationships declared.</p>}
    <div className="flex flex-wrap items-end gap-2">
      <label className="space-y-1 text-sm">Instrumental means
        <select aria-label="Relation source" value={draft.fromObjectiveId} disabled={busy} onChange={event => onDraftChange({ ...draft, fromObjectiveId: event.target.value })} className="block rounded border border-border bg-background p-2">
          <option value="">Select a means</option>
          {instruments.map(objective => <option key={objective.id} value={objective.id}>{objective.title}</option>)}
        </select>
      </label>
      <label className="space-y-1 text-sm">Supports
        <select aria-label="Relation target" value={draft.toObjectiveId} disabled={busy} onChange={event => onDraftChange({ ...draft, toObjectiveId: event.target.value })} className="block rounded border border-border bg-background p-2">
          <option value="">Select a target</option>
          {objectives.map(objective => <option key={objective.id} value={objective.id}>{objective.title}</option>)}
        </select>
      </label>
      <button type="button" onClick={onAdd} disabled={busy || !canAdd} className="rounded bg-primary px-3 py-1.5 text-sm text-primary-foreground disabled:opacity-50">Add relationship</button>
    </div>
  </div>
}

