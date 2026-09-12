import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { fetchPrerequisites, fetchReleaseLadder, setReleaseRank } from "../api/offers";

type NodeLike = { id: string; name: string; releaseRank?: number; status?: string | number; deliverableClass?: string | number; finishBar?: string | number };
type GoalImpactLike = { goalName?: string; goalTitle?: string; deliverableName?: string; projectedPriority?: number };
type PrerequisiteLike = { node?: NodeLike; depth?: number; derivedUrgency?: number; path?: string[] };
type EntryLike = { deliverable?: NodeLike; unlockedRamps?: NodeLike[]; unlockedStreams?: NodeLike[]; audiences?: NodeLike[]; cumulativeRamps?: NodeLike[]; goalImpacts?: GoalImpactLike[]; readinessGoalExists?: boolean; readinessGoalClosed?: boolean; readinessApprovedCommit?: string };

const names = (nodes: NodeLike[] | undefined) => (nodes ?? []).map((node) => node.name).join(", ") || "None recorded";

export function ReleaseLadderPage() {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["release-ladder"], queryFn: () => fetchReleaseLadder(), retry: false });
  const [selected, setSelected] = useState("");
  const [stream, setStream] = useState("");
  const [rank, setRank] = useState("");
  const [view, setView] = useState<"schedule" | "reverse">("schedule");
  const mutation = useMutation({
    mutationFn: () => setReleaseRank({ nodeId: selected, releaseRank: Number(rank) }),
    onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ["release-ladder"] }); setSelected(""); setRank(""); },
  });
  const entries = useMemo(() => (query.data?.entries ?? []) as EntryLike[], [query.data?.entries]);
  const enabling = useMemo(() => [...((query.data?.enabling ?? []) as PrerequisiteLike[])].sort((a, b) => {
    const left = a.derivedUrgency ?? 0; const right = b.derivedUrgency ?? 0;
    if (left === 0 && right !== 0) return 1; if (right === 0 && left !== 0) return -1; return left - right;
  }), [query.data?.enabling]);
  const unscheduled = (((query.data as unknown as { unscheduled?: NodeLike[] } | undefined)?.unscheduled) ?? []) as NodeLike[];
  const streams = useMemo(() => (query.data?.streams ?? []) as NodeLike[], [query.data?.streams]);
  const selectedStream = stream || streams[0]?.id || "";
  const prerequisites = useQuery({ queryKey: ["release-prerequisites", selectedStream], queryFn: () => fetchPrerequisites(selectedStream), enabled: Boolean(selectedStream), retry: false });
  const prerequisiteTree = ((prerequisites.data as unknown as { tree?: PrerequisiteLike[] } | undefined)?.tree ?? []);
  const reverseRows = useMemo(() => entries.flatMap((entry) => (entry.goalImpacts ?? []).map((impact) => ({ impact, deliverable: entry.deliverable }))), [entries]);
  const goalSourceIssue = (query.data as unknown as { availability?: Array<{ source?: string; reason?: string }> } | undefined)?.availability?.find((item) => item.source === "swarm-manager.goals");
  const state = useSurfaceState({ query: { isLoading: query.isLoading, isFetching: query.isFetching, isError: query.isError, error: query.error }, empty: Boolean(query.data && entries.length === 0) });

  return <ExperienceSurface surfaceId="release-ladder" state={state.state} statusMessage={state.reason} data-testid="page-release-ladder" aria-labelledby="release-ladder-heading" className="flex flex-col gap-4">
    <h2 id="release-ladder-heading" className="text-2xl font-semibold">Release ladder</h2>
    <p className="text-app-muted-foreground">Operator-owned deliverable order, unlocked ramps, streams, and audiences.</p>
    <Card data-testid="release-ladder-readiness"><CardHeader><CardTitle>Readiness by deliverable</CardTitle></CardHeader><CardContent>{entries.map((entry) => <p key={`readiness-${entry.deliverable?.id}`} data-testid={`readiness-${entry.deliverable?.id}`}>{entry.deliverable?.name}: {entry.readinessGoalExists ? (entry.readinessGoalClosed ? `closed (${entry.readinessApprovedCommit || "commit unknown"})` : "open") : "no goal"}</p>)}</CardContent></Card>
    <div className="flex gap-2" role="group" aria-label="Release ladder view"><button type="button" aria-pressed={view === "schedule"} onClick={() => setView("schedule")}>Schedule</button><button type="button" aria-pressed={view === "reverse"} onClick={() => setView("reverse")}>What moves</button></div>
    <Card><CardHeader><CardTitle>Schedule a deliverable</CardTitle></CardHeader><CardContent><form className="flex flex-wrap gap-2" onSubmit={(event) => { event.preventDefault(); if (selected && rank) mutation.mutate(); }}><select aria-label="Deliverable" value={selected} onChange={(event) => setSelected(event.target.value)}><option value="">Select deliverable</option>{entries.map((entry) => <option key={entry.deliverable?.id} value={entry.deliverable?.id}>{entry.deliverable?.name}</option>)}</select><input aria-label="Release rank" type="number" min="0" value={rank} onChange={(event) => setRank(event.target.value)} placeholder="Rank" /><button type="submit" disabled={mutation.isPending || !selected || !rank}>Save rank</button></form>{mutation.isError && <p role="alert">{mutation.error instanceof Error ? mutation.error.message : "Could not save release rank."}</p>}</CardContent></Card>
    {view === "schedule" ? <Card data-testid="release-ladder-schedule"><CardContent><div className="overflow-x-auto"><table className="w-full text-left text-sm"><caption className="sr-only">Release ladder entries</caption><thead><tr><th>Rank</th><th>Deliverable</th><th>Class</th><th>Finish bar</th><th>Unlocks</th><th>Serves</th><th>Cumulative ramps</th><th>Goals</th></tr></thead><tbody>{entries.map((entry) => <tr key={entry.deliverable?.id} className="border-t"><td>{entry.deliverable?.releaseRank ?? "—"}</td><th scope="row">{entry.deliverable?.name}</th><td>{String(entry.deliverable?.deliverableClass ?? "MARKETED")}</td><td>{String(entry.deliverable?.finishBar ?? "UNSPECIFIED")}</td><td>{names([...(entry.unlockedRamps ?? []), ...(entry.unlockedStreams ?? [])])}</td><td>{names(entry.audiences)}</td><td>{names(entry.cumulativeRamps)}</td><td>{(entry.goalImpacts ?? []).map((impact) => impact.goalName || impact.goalTitle).filter(Boolean).join(", ") || "None recorded"}</td></tr>)}</tbody></table></div>{query.data && !entries.length && <p role="note">No deliverables have a release rank yet.</p>}</CardContent></Card> : <Card data-testid="release-ladder-goals"><CardHeader><CardTitle>What changes when rank changes</CardTitle></CardHeader><CardContent>{goalSourceIssue && <p data-testid="release-ladder-source-degraded" role="alert">Goals unavailable: {goalSourceIssue.source} — {goalSourceIssue.reason}</p>}{!goalSourceIssue && !reverseRows.length && <p role="note">No goals are attached to scheduled deliverables.</p>}{reverseRows.map((row, index) => <div className="border-t py-2" key={`${row.impact.goalName}-${row.deliverable?.id}-${index}`}><span className="font-medium">{row.impact.goalName || row.impact.goalTitle}</span><span className="mx-2">→</span><span>{row.impact.deliverableName || row.deliverable?.name}</span><span className="ml-2 text-xs text-app-muted-foreground">projected priority {row.impact.projectedPriority ?? "—"}</span></div>)}</CardContent></Card>}
    <Card data-testid="release-ladder-enabling"><CardHeader><CardTitle>Enabling work</CardTitle></CardHeader><CardContent>{enabling.length === 0 && <p role="note">No enabling deliverables are recorded.</p>}{enabling.map((item) => <div className="border-t py-2" key={item.node?.id}><span className="font-medium">{item.node?.name}</span><span className="ml-2 text-xs text-app-muted-foreground">urgency {item.derivedUrgency ?? 0} · {String(item.node?.finishBar ?? "UNSPECIFIED")} · {String(item.node?.status ?? "UNKNOWN")}</span></div>)}{enabling.some((item) => (item.derivedUrgency ?? 0) === 0) && <p className="mt-3 text-sm text-app-muted-foreground">Enables nothing scheduled</p>}</CardContent></Card>
    {unscheduled.length > 0 && <Card data-testid="release-ladder-unscheduled"><CardHeader><CardTitle>Unscheduled marketed deliverables</CardTitle></CardHeader><CardContent>{unscheduled.map((node) => <p key={node.id} role="alert">{node.name}</p>)}</CardContent></Card>}
    <Card><CardHeader><CardTitle>Prerequisites by stream</CardTitle></CardHeader><CardContent data-testid="release-ladder-prerequisites">{query.data && streams.length === 0 && <p role="note">No streams are recorded.</p>}{streams.length > 0 && <select data-testid="release-ladder-stream" aria-label="Revenue stream" value={selectedStream} onChange={(event) => setStream(event.target.value)}>{streams.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select>}{prerequisites.isError && <p role="alert">Prerequisites unavailable: {prerequisites.error instanceof Error ? prerequisites.error.message : "source error"}</p>}{prerequisites.data && prerequisiteTree.length === 0 && <p role="note">No prerequisites are recorded for this stream.</p>}{prerequisiteTree.map((item: PrerequisiteLike) => <div key={item.node?.id} className={`mt-2 rounded border p-2 ${String(item.node?.deliverableClass) === "ENABLING" || item.node?.deliverableClass === 2 ? "border-app-warning" : "border-app-border"}`} style={{ marginLeft: `${(item.depth ?? 0) * 1.25}rem` }}><span className="font-medium">{item.node?.name}</span><span className="ml-2 text-xs text-app-muted-foreground">{String(item.node?.deliverableClass ?? "—")} · {String(item.node?.finishBar ?? "—")} · urgency {item.derivedUrgency ?? 0} · depth {item.depth ?? 0}</span><div className="text-xs text-app-muted-foreground">{(item.path ?? []).join(" → ")}</div></div>)}</CardContent></Card>
  </ExperienceSurface>;
}
