import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { fetchReleaseLadder, setReleaseRank } from "../api/offers";

type NodeLike = { id: string; name: string; releaseRank?: number; status?: string | number };
type EntryLike = { deliverable?: NodeLike; unlockedRamps?: NodeLike[]; unlockedStreams?: NodeLike[]; audiences?: NodeLike[]; cumulativeRamps?: NodeLike[] };

const names = (nodes: NodeLike[] | undefined) => (nodes ?? []).map((node) => node.name).join(", ") || "None recorded";

export function ReleaseLadderPage() {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["release-ladder"], queryFn: () => fetchReleaseLadder(), retry: false });
  const [selected, setSelected] = useState("");
  const [rank, setRank] = useState("");
  const mutation = useMutation({
    mutationFn: () => setReleaseRank({ nodeId: selected, releaseRank: Number(rank) }),
    onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ["release-ladder"] }); setSelected(""); setRank(""); },
  });
  const entries = useMemo(() => (query.data?.entries ?? []) as EntryLike[], [query.data?.entries]);
  const state = useSurfaceState({ query: { isLoading: query.isLoading, isFetching: query.isFetching, isError: query.isError, error: query.error }, empty: Boolean(query.data && entries.length === 0) });

  return <ExperienceSurface surfaceId="release-ladder" state={state.state} statusMessage={state.reason} data-testid="page-release-ladder" aria-labelledby="release-ladder-heading" className="flex flex-col gap-4">
    <h2 id="release-ladder-heading" className="text-2xl font-semibold">Release ladder</h2>
    <p className="text-app-muted-foreground">Operator-owned deliverable order, unlocked ramps, streams, and audiences.</p>
    <Card>
      <CardHeader><CardTitle>Schedule a deliverable</CardTitle></CardHeader>
      <CardContent><form className="flex flex-wrap gap-2" onSubmit={(event) => { event.preventDefault(); if (selected && rank) mutation.mutate(); }}>
        <select aria-label="Deliverable" value={selected} onChange={(event) => setSelected(event.target.value)}><option value="">Select deliverable</option>{entries.map((entry: EntryLike) => <option key={entry.deliverable?.id} value={entry.deliverable?.id}>{entry.deliverable?.name}</option>)}</select>
        <input aria-label="Release rank" type="number" min="0" value={rank} onChange={(event) => setRank(event.target.value)} placeholder="Rank" />
        <button type="submit" disabled={mutation.isPending || !selected || !rank}>Save rank</button>
      </form>{mutation.isError && <p role="alert">{mutation.error instanceof Error ? mutation.error.message : "Could not save release rank."}</p>}</CardContent>
    </Card>
    <Card><CardContent><div className="overflow-x-auto"><table className="w-full text-left text-sm"><caption className="sr-only">Release ladder entries</caption><thead><tr><th>Rank</th><th>Deliverable</th><th>Unlocks</th><th>Serves</th><th>Cumulative ramps</th></tr></thead><tbody>{entries.map((entry) => <tr key={entry.deliverable?.id} className="border-t"><td>{entry.deliverable?.releaseRank ?? "—"}</td><th scope="row">{entry.deliverable?.name}</th><td>{names([...(entry.unlockedRamps ?? []), ...(entry.unlockedStreams ?? [])])}</td><td>{names(entry.audiences)}</td><td>{names(entry.cumulativeRamps)}</td></tr>)}</tbody></table></div>{query.data && !entries.length && <p role="note">No deliverables have a release rank yet.</p>}</CardContent></Card>
  </ExperienceSurface>;
}
