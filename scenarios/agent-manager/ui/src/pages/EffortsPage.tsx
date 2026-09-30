import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";
import {
  EffortFreshness,
  type EffortBoardRow,
  type EffortEnrollment,
  type EffortUsage,
} from "@vrooli/proto-types/agent-manager/v1/domain/effort_pb";
import { useEffortBoard } from "../features/efforts/api";
import { Button } from "../components/ui/button";

const freshnessNames: Record<number, string> = {
  [EffortFreshness.FRESH]: "fresh",
  [EffortFreshness.STALE]: "stale",
  [EffortFreshness.UNAVAILABLE]: "unavailable",
};
const when = (value?: Timestamp) => value ? timestampDate(value).toISOString() : "unknown";

function Ownership({ enrollment }: { enrollment?: EffortEnrollment }) {
  if (enrollment?.withdrawn) return <>Withdrawn: {enrollment.withdrawalReason || "no reason reported"}</>;
  if (!enrollment?.authorizedBy) return <>Not established; discovery is observation only</>;
  return <>Enrolled by {enrollment.authorizedBy}</>;
}

function References({ values }: { values: string[] }) {
  return values.length ? <ul className="list-disc space-y-1 pl-5 text-xs">{values.map((value, i) => <li className="break-all" key={`${i}-${value}`}>{value}</li>)}</ul> : <p className="text-sm text-muted-foreground">None reported.</p>;
}

function Usage({ usage }: { usage?: EffortUsage }) {
  return <div className="space-y-2 text-sm">
    <dl className="grid gap-3 sm:grid-cols-3">
      <div><dt className="text-muted-foreground">Reported tokens</dt><dd>{usage?.tokens?.toLocaleString() ?? "unknown"}</dd></div>
      <div><dt className="text-muted-foreground">Reported cost (USD)</dt><dd>{usage?.reportedCostUsd === undefined ? "unknown" : usage.reportedCostUsd.toFixed(4)}</dd></div>
      <div><dt className="text-muted-foreground">Summed agent seconds</dt><dd>{usage?.agentSeconds?.toLocaleString() ?? "unknown"}</dd></div>
    </dl>
    <p className="text-xs text-muted-foreground">{usage ? `${usage.observedRuns} observed / ${usage.declaredRuns} declared runs · ${usage.partial ? "partial coverage" : "reported coverage"}` : "Usage coverage unavailable"}. Shared runs must not be summed across efforts.</p>
    {usage?.source ? <p className="break-all text-xs">Source: {usage.source}</p> : null}
    {usage?.limitations.length ? <References values={usage.limitations} /> : null}
  </div>;
}

function EffortDetail({ row }: { row: EffortBoardRow }) {
  const enrollment = row.enrollment;
  return <article className="space-y-5 rounded-md border border-border bg-card p-4" aria-label={`Effort ${enrollment?.effortRef} details`}>
    <header><h2 className="text-lg font-semibold">{enrollment?.displayName || enrollment?.effortRef}</h2><p className="break-all text-xs text-muted-foreground">{enrollment?.effortRef} · enrollment r{enrollment?.revision.toString() ?? "unknown"}</p></header>
    <dl className="grid gap-3 text-sm sm:grid-cols-2">
      <div><dt className="text-muted-foreground">Runtime activity</dt><dd>{row.runtimeState || "unknown"}</dd></div>
      <div><dt className="text-muted-foreground">Outcome standing</dt><dd>{row.outcomeStanding?.state || "unverified"} · {row.outcomeStanding?.attribution || "source unknown"}</dd></div>
      <div><dt className="text-muted-foreground">Evidence freshness</dt><dd>{freshnessNames[row.freshness] || "unknown"} · {when(row.observedAt)}</dd></div>
      <div><dt className="text-muted-foreground">Work shape</dt><dd>{enrollment?.workShape || "unknown"}</dd></div>
      <div><dt className="text-muted-foreground">Accepted destination reference</dt><dd className="break-all">{enrollment?.destinationRef || "unknown"}</dd></div>
      <div><dt className="text-muted-foreground">Target revision</dt><dd className="break-all">{enrollment?.targetRevision || "unknown"}</dd></div>
      <div><dt className="text-muted-foreground">Enrollment owner</dt><dd><Ownership enrollment={enrollment} /></dd></div>
      {enrollment?.authorityRef ? <div><dt className="text-muted-foreground">Authority reference</dt><dd className="break-all">{enrollment.authorityRef}</dd></div> : null}
    </dl>
    <section><h3 className="mb-1 font-semibold">Next action and rationale</h3><p className="text-sm">{row.nextAction || "No next action reported."}</p><p className="text-sm text-muted-foreground">{row.rationale || "No rationale reported."}</p></section>
    <section><h3 className="mb-1 font-semibold">Blockers</h3><References values={row.blockers} /></section>
    <section><h3 className="mb-1 font-semibold">Pending owner operations</h3><References values={row.pendingOperations} /></section>
    <section><h3 className="mb-2 font-semibold">Resource attribution</h3><Usage usage={row.usage} /></section>
    <section><h3 className="mb-2 font-semibold">Assignments and configuration</h3>
      <p className="mb-2 text-xs text-muted-foreground">These are owner assignments, separate from a Prompt Manager team’s registered members. Role and model settings grant no authority.</p>
      {row.assignments.length ? <div className="space-y-3">{row.assignments.map((assignment, i) => <div key={`${assignment.subject?.reference}-${i}`} className="space-y-2 rounded bg-muted/30 p-3 text-xs">
        <p>{assignment.subject?.role || "role unknown"} · {assignment.runtimeState || "runtime unknown"} · {assignment.subject?.assignment || "assignment unknown"}</p>
        {assignment.subject?.runId ? <Link className="text-primary underline" to={`/runs/${encodeURIComponent(assignment.subject.runId)}`}>Run {assignment.subject.runId}</Link> : <p className="break-all">{assignment.subject?.reference || "No owner run reference"}</p>}
        <div className="overflow-x-auto"><table className="w-full text-left"><thead><tr><th scope="col">Setting</th><th scope="col">Requested</th><th scope="col">Effective</th></tr></thead><tbody>
          <tr><th scope="row">Model</th><td>{assignment.requestedModel || "unknown"}</td><td>{assignment.effectiveModel || "unknown"}</td></tr>
          <tr><th scope="row">Runner</th><td>{assignment.requestedRunner || "unknown"}</td><td>{assignment.effectiveRunner || "unknown"}</td></tr>
          <tr><th scope="row">Reasoning</th><td>{assignment.requestedReasoning || "unknown"}</td><td>{assignment.effectiveReasoning || "unknown"}</td></tr>
        </tbody></table></div>
        <p>Observed: {when(assignment.observedAt)}</p>
        {assignment.unavailableReason ? <p className="text-warning">{assignment.unavailableReason}</p> : null}
        <Usage usage={assignment.usage} />
      </div>)}</div> : <p className="text-sm text-muted-foreground">No attributed assignments; this does not mean no agents are running.</p>}
    </section>
    <section><h3 className="mb-1 font-semibold">Evidence references</h3><References values={[...row.evidenceRefs, ...(row.outcomeStanding?.evidenceRefs || [])]} /></section>
    <section><h3 className="mb-1 font-semibold">Coverage and limitations</h3><References values={[...row.limitations, ...(row.outcomeStanding?.limitations || [])]} /></section>
  </article>;
}

export function EffortsPage() {
  const [pages, setPages] = useState([""]);
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedRef = searchParams.get("effortRef") || "";
  useEffect(() => { setPages([""]); }, [selectedRef]);
  const result = useEffortBoard(selectedRef ? "" : pages[pages.length - 1] ?? "", selectedRef);
  const board = result.data;
  const selected = selectedRef ? board?.rows.find(row => row.enrollment?.effortRef === selectedRef) : board?.rows[0];
  const discovery = board?.discovery;
  return <section className="h-full overflow-auto p-4 sm:p-6" aria-labelledby="effort-board-title"><div className="mx-auto max-w-7xl space-y-4">
    <header className="flex flex-wrap items-start justify-between gap-3"><div><h1 id="effort-board-title" className="text-xl font-semibold">Effort supervision</h1><p className="mt-1 text-sm text-muted-foreground">Delivery evidence and resource use across work shapes.</p></div><Button variant="outline" onClick={() => void result.refetch()} disabled={result.isFetching}>Refresh observation</Button></header>
    <p className="rounded border border-border bg-muted/30 p-3 text-xs">Read-only owner projection. Activity is not acceptance; missing cost is not zero. Refresh does not scan or start supervision.</p>
    {selectedRef ? <div className="flex flex-wrap items-center gap-3 text-sm"><p className="break-all">Exact effort: {selectedRef}</p><Button variant="outline" onClick={() => setSearchParams(params => { params.delete("effortRef"); return params; })}>Show all efforts</Button></div> : null}
    {result.error ? <div role="alert" className="rounded border border-destructive/40 p-3 text-sm">Owner observation unavailable: {result.error.message}. {board ? "Retained results may be stale." : "Effort count and state are unknown."}</div> : null}
    {result.isPending ? <p role="status">Loading effort observation…</p> : null}
    {board ? <>
      <section aria-label="Discovery coverage" className="space-y-2 rounded border border-border p-3 text-sm">
        <p>{board.activeCount} active enrollments · {board.partial ? "partial board coverage" : "reported board coverage"} · observed {when(board.observedAt)}</p>
        <p>Discovery generation {discovery?.generation.toString() ?? "unknown"} · last scan {when(discovery?.lastScanAt)} · last successful scan {when(discovery?.lastSuccessfulScanAt)}</p>
        <p>{discovery ? `${discovery.scannedCount} scanned / ${discovery.scanLimit} scan cap · ${discovery.partial ? "partial discovery" : "reported discovery coverage"}` : "Discovery coverage unknown"}</p>
        {discovery?.findings.length ? <References values={discovery.findings.map(f => `${f.code}: ${f.reason} (${f.source})`)} /> : null}
        {board.limitations.length ? <References values={board.limitations} /> : null}
      </section>
      {selectedRef && !selected ? <p role="alert" className="rounded border border-border p-3 text-sm">The exact requested effort was not returned. Its observation is unavailable; no other effort has been selected.</p> : null}
      {!board.rows.length ? <p className="rounded border border-dashed border-border p-6 text-sm">No enrollments in this observation. Check discovery coverage before concluding that no work exists.</p> : <div className="grid items-start gap-4 lg:grid-cols-[minmax(16rem,0.7fr)_minmax(0,1.7fr)]">
        <nav aria-label="Observed efforts" className="space-y-2">{board.rows.map((row, i) => <button type="button" className={`block w-full rounded border border-border p-3 text-left text-sm ${selected === row ? "bg-accent" : "bg-card"}`} aria-pressed={selected === row} key={row.enrollment?.effortRef || i} onClick={() => setSearchParams(params => { params.set("effortRef", row.enrollment?.effortRef || ""); return params; })}>
          <span className="block break-all font-semibold">{row.enrollment?.displayName || row.enrollment?.effortRef || "Unknown enrollment"}</span>
          <span className="block">Activity: {row.runtimeState || "unknown"}</span><span className="block">Outcome: {row.outcomeStanding?.state || "unverified"}</span><span className="text-xs text-muted-foreground">Evidence: {freshnessNames[row.freshness] || "unknown"}</span>
        </button>)}</nav>
        {selected ? <EffortDetail row={selected} /> : null}
      </div>}
      {!selectedRef ? <nav aria-label="Effort pages" className="flex items-center gap-3"><Button variant="outline" disabled={pages.length <= 1 || result.isFetching} onClick={() => setPages(old => old.slice(0, -1))}>Previous</Button><span className="text-sm">Page {pages.length}</span><Button variant="outline" disabled={!board.nextPageToken || result.isFetching} onClick={() => setPages(old => [...old, board.nextPageToken])}>Next</Button></nav> : null}
    </> : null}
  </div></section>;
}
