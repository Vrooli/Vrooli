import { useEffect, useState } from 'react'
import { timestampDate, type Timestamp } from '@bufbuild/protobuf/wkt'
import { EffortFreshness, type EffortBoardRow, type EffortEnrollment } from '@vrooli/proto-types/agent-manager/v1/domain/effort_pb'
import { effortOwnerPath, LINKED_EFFORT_PAGE_SIZE, useAgentManagerUrl, useEffortObservations } from '@/services/effortService'

interface EffortTeam {
  id: string
  displayName: string
  effortRefs?: string[]
  purpose?: string
  members?: Array<{ agentId: string }>
}

export interface TeamEffortsPanelProps {
  team?: EffortTeam
  teams?: EffortTeam[]
  teamRegistryComplete?: boolean
  leaderEffortRefs?: string[]
  scheduled?: { enabled: boolean; summary?: string }
  onOpenTeam?: (id: string) => void
}

const when = (value?: Timestamp) => value ? timestampDate(value).toLocaleString() : 'unknown'
const freshnessNames: Partial<Record<EffortFreshness, string>> = { [EffortFreshness.FRESH]: 'fresh', [EffortFreshness.STALE]: 'stale', [EffortFreshness.UNAVAILABLE]: 'unavailable' }
const freshness = (value: EffortFreshness) => freshnessNames[value] ?? 'unknown'

function describeEffortOwner(enrollment: EffortEnrollment | undefined): string {
  if (enrollment?.withdrawn) return `Withdrawn: ${enrollment.withdrawalReason || 'reason not reported'}`
  if (!enrollment?.authorizedBy) return 'Observation only; not enrolled by an owner'
  return `Enrolled by ${enrollment.authorizedBy}`
}

function EffortCard({ row, teams, teamRegistryComplete, ownerUrl, onOpenTeam, leaderBound }: {
  row: EffortBoardRow
  teams: EffortTeam[]
  teamRegistryComplete: boolean
  ownerUrl?: string
  onOpenTeam?: (id: string) => void
  leaderBound: boolean
}) {
  const ref = row.enrollment?.effortRef ?? ''
  const bindings = teams.filter(team => team.effortRefs?.includes(ref))
  const assignments = row.assignments.slice(0, 5)
  return <article className="space-y-3 rounded-lg border border-border bg-muted/15 p-3" aria-label={`Effort ${ref}`}>
    <div>
      <h4 className="font-medium">{row.enrollment?.displayName || ref || 'Unknown effort'}</h4>
      <p className="break-all text-xs text-muted-foreground">{ref}</p>
      {row.enrollment?.withdrawn ? <p className="text-xs text-muted-foreground">Withdrawn enrollment</p> : null}
      <p className="mt-1 text-xs">{bindings.length ? <>Authored team relationship{bindings.length > 1 ? 's' : ''}: {bindings.map((binding, i) => <span key={binding.id}>{i ? ', ' : ''}{onOpenTeam ? <button type="button" className="text-primary underline" onClick={() => onOpenTeam(binding.id)}>{binding.displayName}</button> : binding.displayName}</span>)}</> : teamRegistryComplete ? 'No authored team relationship in the registered team list.' : 'Authored team relationships unknown; registry coverage unavailable.'}</p>
      <p className="text-xs text-muted-foreground">{leaderBound ? 'Finite leader binding for this team, reported by its heartbeat configuration.' : 'Runtime leader binding coverage unknown.'}</p>
    </div>
    <dl className="grid gap-2 text-xs sm:grid-cols-2">
      <div><dt className="text-muted-foreground">Owner execution</dt><dd>{row.runtimeState || 'unknown'}</dd></div>
      <div><dt className="text-muted-foreground">Accepted outcome</dt><dd>{row.outcomeStanding?.state || 'unverified'} · {row.outcomeStanding?.attribution || 'source unknown'}</dd></div>
      <div><dt className="text-muted-foreground">Evidence</dt><dd>{freshness(row.freshness)} · observed {when(row.observedAt)}</dd></div>
      <div><dt className="text-muted-foreground">Enrollment owner</dt><dd>{describeEffortOwner(row.enrollment)}</dd></div>
    </dl>
    {row.enrollment?.authorityRef ? <p className="break-all text-xs text-muted-foreground">Authority reference: {row.enrollment.authorityRef}</p> : null}
    <div className="text-xs"><p><strong>Next action or wait:</strong> {row.nextAction || 'Not reported'}</p>{row.rationale ? <p className="text-muted-foreground">{row.rationale}</p> : null}</div>
    <details className="text-xs"><summary className="cursor-pointer font-medium">Temporary assignments and models ({row.assignments.length} reported)</summary>
      <p className="my-2 text-muted-foreground">Owner assignments are separate from the registered member roster.</p>
      {assignments.length ? <ul className="space-y-2">{assignments.map((assignment, i) => <li className="rounded border border-border p-2" key={`${assignment.subject?.reference}-${i}`}>
        <p>{assignment.subject?.role || 'Role unknown'} · {assignment.runtimeState || 'runtime unknown'}</p>
        <p>{assignment.subject?.assignment || 'Assignment not reported'}</p>
        <p>Requested model: {assignment.requestedModel || 'unknown'} · effective model: {assignment.effectiveModel || 'unknown'}</p>
        <p>Requested reasoning: {assignment.requestedReasoning || 'unknown'} · effective reasoning: {assignment.effectiveReasoning || 'unknown'}</p>
        {ownerUrl && assignment.subject?.runId ? <a className="text-primary underline" href={`${ownerUrl}/runs/${encodeURIComponent(assignment.subject.runId)}`}>Open {assignment.subject.role || 'assignment'} run</a> : null}
        {assignment.unavailableReason ? <p className="text-muted-foreground">{assignment.unavailableReason}</p> : null}
      </li>)}</ul> : <p>No attributed assignments; execution coverage is unknown.</p>}
      {row.assignments.length > assignments.length ? <p className="mt-2">Showing {assignments.length} of {row.assignments.length}; open the effort for all assignments.</p> : null}
    </details>
    {row.limitations.length || row.blockers.length ? <details className="text-xs"><summary className="cursor-pointer">Blockers and coverage limitations</summary><ul className="mt-2 list-disc pl-4">{[...row.blockers, ...row.limitations].map((item, i) => <li key={i}>{item}</li>)}</ul></details> : null}
    {ownerUrl && ref ? <a className="inline-block text-xs text-primary underline" href={effortOwnerPath(ownerUrl, ref)}>Open effort, acceptance evidence and all assignments</a> : <p className="text-xs text-muted-foreground">Agent Manager navigation unavailable.</p>}
  </article>
}

export function TeamEffortsPanel({ team, teams = [], teamRegistryComplete = false, leaderEffortRefs = [], scheduled, onOpenTeam }: TeamEffortsPanelProps) {
  const [pages, setPages] = useState([''])
  const [referencePage, setReferencePage] = useState(0)
  const refs = team ? [...new Set([...(team.effortRefs ?? []), ...leaderEffortRefs])].sort() : undefined
  const refsKey = refs?.join('\u0000')
  useEffect(() => { setPages(['']); setReferencePage(0) }, [team?.id, refsKey])
  const pageRefs = refs?.slice(referencePage * LINKED_EFFORT_PAGE_SIZE, (referencePage + 1) * LINKED_EFFORT_PAGE_SIZE)
  const result = useEffortObservations(pages[pages.length - 1] ?? '', pageRefs)
  const owner = useAgentManagerUrl()
  const observations = result.data
  const boards = observations?.boards ?? []
  const firstBoard = boards[0]
  const rows = boards.flatMap(board => board.rows)
  const available = boards.length > 0
  const noRefs = refs?.length === 0
  const unavailable = observations?.unavailable ?? []
  const partial = boards.some(board => board.partial || board.discovery?.partial) || unavailable.length > 0
  const nextToken = refs === undefined ? boards[0]?.nextPageToken : undefined
  const canNext = refs ? (referencePage + 1) * LINKED_EFFORT_PAGE_SIZE < refs.length : !!nextToken
  const registry = team && !teams.some(item => item.id === team.id) ? [...teams, team] : teams
  return <section aria-label={team ? 'Linked efforts' : 'Discovered efforts'} className="space-y-3 rounded-xl border border-border p-4">
    <div className="flex flex-wrap items-center justify-between gap-2"><h3 className="font-semibold">{team ? 'Linked efforts' : 'Discovered efforts'}</h3><button type="button" className="text-xs text-primary underline disabled:opacity-50" disabled={result.isFetching || noRefs} onClick={() => void result.refetch()}>Refresh work observations</button></div>
    <p className="text-xs text-muted-foreground">{team ? 'Authored effort references and finite leader bindings connect this team to work.' : 'Work can exist before a Prompt Manager team is registered.'} Owner activity, scheduling and accepted outcomes are separate.</p>
    {team ? <p className="text-xs"><strong>Team scheduling:</strong> {scheduled ? `${scheduled.enabled ? 'Enabled' : 'Not enabled'}${scheduled.summary ? ` · ${scheduled.summary}` : ''}` : 'Unknown'}. Scheduling does not prove current execution.</p> : null}
    {noRefs ? <p className="text-sm text-muted-foreground">No authored references or finite leader bindings available. This does not establish that the team has no work.</p> : null}
    {!noRefs && result.isPending ? <p role="status" className="text-sm">Loading owner effort observations…</p> : null}
    {result.error ? <p role="alert" className="text-sm">Owner observation unavailable: {result.error.message}. {available ? 'Retained observations may be stale.' : 'Effort count and state are unknown.'}</p> : null}
    {unavailable.length ? <div role="alert" className="text-xs"><p>Some linked effort observations are unavailable; their state is unknown.</p><ul className="list-disc pl-4">{unavailable.map(item => <li className="break-all" key={item.effortRef}>{item.effortRef}: {item.reason}</li>)}</ul></div> : null}
    {available ? <>
      <p className="text-xs text-muted-foreground">{partial ? 'Partial owner coverage' : 'Reported owner coverage'} · {rows.length} effort{rows.length === 1 ? '' : 's'} on this page · observed {when(firstBoard?.observedAt)}. Refresh reads the owner; it does not scan or start work.</p>
      {firstBoard ? <p className="text-xs text-muted-foreground">{firstBoard.activeCount} active enrollments reported across the owner registry; this page may also include withdrawn work.</p> : null}
      <p className="text-xs text-muted-foreground">Discovery: {firstBoard?.discovery ? `last successful scan ${when(firstBoard.discovery.lastSuccessfulScanAt)} · ${firstBoard.discovery.scannedCount} scanned / ${firstBoard.discovery.scanLimit} cap` : 'coverage unknown'}.</p>
      {boards.some(board => board.limitations.length || board.discovery?.findings.length) ? <details className="text-xs"><summary className="cursor-pointer">Owner discovery limitations</summary><ul className="list-disc pl-4">{[...new Set(boards.flatMap(board => [...board.limitations, ...(board.discovery?.findings.map(finding => `${finding.code}: ${finding.reason}`) ?? [])]))].map(item => <li key={item}>{item}</li>)}</ul></details> : null}
      {!rows.length ? <p className="text-sm">No efforts returned on this page. Check discovery coverage before concluding that no work exists.</p> : null}
      <div className="space-y-3">{rows.map((row, i) => <EffortCard key={row.enrollment?.effortRef || i} row={row} teams={registry} teamRegistryComplete={teamRegistryComplete} ownerUrl={owner.data} onOpenTeam={onOpenTeam} leaderBound={!!team && leaderEffortRefs.includes(row.enrollment?.effortRef || '')} />)}</div>
    </> : null}
    {!noRefs && (available || refs) ? <nav aria-label="Work observation pages" className="flex items-center gap-3 text-xs"><button type="button" className="text-primary underline disabled:opacity-50" disabled={result.isFetching || (refs ? referencePage === 0 : pages.length === 1)} onClick={() => refs ? setReferencePage(page => page - 1) : setPages(old => old.slice(0, -1))}>Previous efforts</button><span>Page {refs ? referencePage + 1 : pages.length}</span><button type="button" className="text-primary underline disabled:opacity-50" disabled={result.isFetching || !canNext} onClick={() => refs ? setReferencePage(page => page + 1) : setPages(old => nextToken ? [...old, nextToken] : old)}>Next efforts</button></nav> : null}
  </section>
}
