import { useEffect, useRef, useState } from 'react'
import { Dialog } from '@/components/shared/Dialog'
import { MarkdownRenderer } from '@/components/markdown/MarkdownRenderer'
import type { Team, TeamDetails, UpdateTeamRequest } from '@/types/team'
import { UpdateTeamRequestSchema } from '@/lib/schemas'
import teamModel from '../../../../docs/concepts/SWARM-MODEL.md?raw'
import { useAgentManagerUrl } from '@/services/effortService'

export const teamPurposeLabels = {
  'domain-stewardship': 'Domain stewardship',
  delivery: 'Delivery',
  supervision: 'Supervision',
} as const

export const teamLifetimeLabels = { standing: 'Standing', finite: 'Finite' } as const

export const teamPresets = {
  custom: { label: 'Custom team', purpose: '', lifetime: '' },
  domain: { label: 'Standing domain team', purpose: 'domain-stewardship', lifetime: 'standing' },
  delivery: { label: 'Finite delivery team', purpose: 'delivery', lifetime: 'finite' },
  supervision: { label: 'Standing supervision team', purpose: 'supervision', lifetime: 'standing' },
} as const

export function TeamPurposeBadges({ team }: { team: Pick<Team, 'purpose' | 'lifetime'> }) {
  return <span className="flex flex-wrap gap-1 text-[11px]">
    <span className="rounded border border-border px-1.5 py-0.5 text-muted-foreground">{team.lifetime ? teamLifetimeLabels[team.lifetime] : 'Lifetime unspecified'}</span>
    <span className="rounded bg-primary/10 px-1.5 py-0.5 text-primary">{team.purpose ? teamPurposeLabels[team.purpose] : 'Purpose unspecified'}</span>
  </span>
}

export function TeamModelHelp() {
  const [open, setOpen] = useState(false)
  const ownerUrl = useAgentManagerUrl(open)
  return <>
    <button type="button" className="text-xs text-primary underline underline-offset-2" onClick={() => setOpen(true)}>How teams and efforts work</button>
    <Dialog isOpen={open} onClose={() => setOpen(false)} title="Teams and efforts" maxWidth="max-w-3xl" appearance="theme">
      <div className="space-y-4 text-sm text-muted-foreground">
        <p>A standing team maintains a responsibility over time. A finite delivery team works toward an accepted destination, then retires its recurring execution.</p>
        <p>Effort Supervision is a standing team. Its watches end individually while it remains available for other efforts. Director Swarm owns portfolio priorities; the effort orchestrator owns delivery and worker assignments.</p>
        <p>Authority comes from the operating contract and actual grants. Purpose, lifetime, and an enabled schedule do not grant additional permission.</p>
        <p>Registered members provide durable roles and context. Temporary workers belong to assignments. An effort can use a temporary driver before it has a registered team.</p>
        {ownerUrl.data ? <a href={`${ownerUrl.data}/efforts`} target="_blank" rel="noreferrer" className="inline-block text-primary underline">Open the effort board</a> : <p className="text-xs">{ownerUrl.error ? 'The effort board address is unavailable.' : 'Resolving the effort board address…'}</p>}
        <details className="border-t border-border pt-3">
          <summary className="cursor-pointer text-foreground">Read the canonical team model</summary>
          <MarkdownRenderer content={teamModel} className="mt-4" />
        </details>
      </div>
    </Dialog>
  </>
}

export function TeamPurposePanel({ team, onUpdate }: { team: TeamDetails; onUpdate: (updates: UpdateTeamRequest) => Promise<void> }) {
  const [purpose, setPurpose] = useState(team.purpose ?? '')
  const [lifetime, setLifetime] = useState(team.lifetime ?? '')
  const [effortRefs, setEffortRefs] = useState((team.effortRefs ?? []).join('\n'))
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const metadataKey = JSON.stringify([team.purpose, team.lifetime, team.effortRefs])
  const observedMetadata = useRef(metadataKey)

  useEffect(() => {
    // A successful mutation can resolve before the refreshed query arrives.
    // Only new server metadata may replace the draft, not the dirty flag alone.
    if (observedMetadata.current === metadataKey) return
    observedMetadata.current = metadataKey
    if (dirty) return
    setPurpose(team.purpose ?? '')
    setLifetime(team.lifetime ?? '')
    setEffortRefs((team.effortRefs ?? []).join('\n'))
  }, [metadataKey, team.purpose, team.lifetime, team.effortRefs, dirty])

  const changed = () => { setDirty(true); setSaved(false); setError(null) }
  const save = async () => {
    setSaving(true)
    setError(null)
    try {
      const updates = UpdateTeamRequestSchema.parse({
        purpose, lifetime,
        effortRefs: [...new Set(effortRefs.split('\n').map(ref => ref.trim()).filter(Boolean))],
      })
      await onUpdate(updates)
      setDirty(false)
      setSaved(true)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Team details could not be saved.')
    } finally { setSaving(false) }
  }

  return <section aria-label="Team purpose and authority" className="space-y-4 rounded-lg border border-border p-4">
    <div className="flex flex-wrap items-center justify-between gap-3"><TeamPurposeBadges team={team} /><TeamModelHelp /></div>
    <p className="text-sm text-muted-foreground">Purpose describes the responsibility. Lifetime describes when it ends. Execution and permissions have their own controls.</p>
    <div className="grid gap-4 sm:grid-cols-2">
      <label className="space-y-1 text-sm">Purpose
        <select aria-label="Team purpose" value={purpose} disabled={saving} onChange={event => { setPurpose(event.target.value as typeof purpose); changed() }} className="block w-full rounded border border-border bg-background p-2">
          <option value="">Unspecified</option>
          {Object.entries(teamPurposeLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>
      <label className="space-y-1 text-sm">Lifetime
        <select aria-label="Team lifetime" value={lifetime} disabled={saving} onChange={event => { setLifetime(event.target.value as typeof lifetime); changed() }} className="block w-full rounded border border-border bg-background p-2">
          <option value="">Unspecified</option>
          {Object.entries(teamLifetimeLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>
    </div>
    <p className="text-xs text-muted-foreground">{lifetime === 'finite' ? 'Finite teams stop recurring work after evidenced completion or withdrawal. Selecting this label does not activate or retire a schedule.' : lifetime === 'standing' ? 'Standing teams continue serving their responsibility. Completing a run or one effort does not retire the team.' : 'Choose a lifetime when the operating contract establishes one.'}</p>
    <details>
      <summary className="cursor-pointer text-sm">Linked effort references ({team.effortRefs?.length ?? 0})</summary>
      <label className="mt-2 block space-y-1 text-sm">One canonical effort reference per line
        <textarea aria-label="Linked effort references" value={effortRefs} disabled={saving} onChange={event => { setEffortRefs(event.target.value); changed() }} rows={3} className="block w-full rounded border border-border bg-background p-2 font-mono text-xs" />
      </label>
      <p className="mt-1 text-xs text-muted-foreground">Link delivery or contribution to existing efforts. This does not enroll work or grant authority. A supervisor's observed efforts are supplied by its runtime.</p>
    </details>
    <div className="flex items-center gap-3">
      <button type="button" disabled={!dirty || saving} onClick={() => void save()} className="rounded bg-primary px-3 py-1.5 text-sm text-primary-foreground disabled:opacity-50">{saving ? 'Saving…' : 'Save team details'}</button>
      {saved ? <span role="status" className="text-sm text-muted-foreground">Team details saved.</span> : null}
    </div>
    {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
    <div className="space-y-2 border-t border-border pt-3">
      <h3 className="text-sm font-semibold">Objectives served</h3>
      {team.objectivesServed?.length ? <ul className="space-y-1 text-sm">{team.objectivesServed.map(objective => <li key={objective.id}>
        <span className="font-medium">{objective.id}</span>{objective.role ? ` · ${objective.role}` : ''}{objective.coverage ? ` · ${objective.coverage} coverage` : ''}
        {objective.note ? <p className="text-xs text-muted-foreground">{objective.note}</p> : null}
        {objective.acknowledgedRevision ? <p className="text-xs text-muted-foreground">Acknowledged revision: {objective.acknowledgedRevision}</p> : null}
      </li>)}</ul> : <p className="text-sm text-muted-foreground">No objective relationships declared.</p>}
    </div>
    <details className="border-t border-border pt-3">
      <summary className="cursor-pointer text-sm font-semibold">Declared authority and source contracts</summary>
      <p className="my-2 text-xs text-muted-foreground">These are the member's declared work boundaries. Current effort action grants appear with the effort observation below.</p>
      {Object.entries(team.operatingContract.members).length ? Object.entries(team.operatingContract.members).map(([id, member]) => <div key={id} className="my-3 space-y-1 rounded bg-muted p-3 text-sm">
        <h4 className="font-medium">{team.members.find(item => item.agentId === id)?.displayName ?? id}</h4>
        <p>{member.lane}</p>
        <p className="text-xs">Allowed writes: {member.allowedWrites?.length ? member.allowedWrites.map(write => [write.kind, write.base, write.path].filter(Boolean).join(': ')).join('; ') : 'No write surfaces declared.'}</p>
        {member.forbiddenWrites?.length ? <p className="text-xs">Prohibited writes: {member.forbiddenWrites.map(write => [write.kind, write.base, write.path].filter(Boolean).join(': ')).join('; ')}</p> : null}
        {member.safetyCriticalRules?.length ? <ul className="list-disc space-y-1 pl-4 text-xs text-muted-foreground">{member.safetyCriticalRules.map(rule => <li key={rule}>{rule}</li>)}</ul> : null}
      </div>) : <p className="text-sm text-muted-foreground">No member authority declared. Do not infer permission from the team label.</p>}
      {team.operatingContract.documents.planOfRecord.map(source => <div key={source.id} className="my-2 text-xs">
        <p className="font-medium">{source.id} · {source.writePolicy}</p>
        {source.paths.map((path, index) => <p key={index} className="break-all text-muted-foreground">{[path.base, path.path].filter(Boolean).join(': ')}</p>)}
      </div>)}
    </details>
  </section>
}
