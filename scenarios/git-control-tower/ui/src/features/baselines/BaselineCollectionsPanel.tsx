import { useState } from "react";
import { Layers, RefreshCw } from "lucide-react";
import { Button } from "../../components/ui/button";
import { diffBaselineCollection, getBaselineCollection } from "../../lib/api-baseline-collections";
import type {
  BaselineCollection,
  GetCollectionDiffStatusResponse,
} from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

interface BaselineCollectionsPanelProps {
  repoId?: string | null;
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Collection request failed";
}

// Baseline collections are usually created by a Plan Manager execution. This
// compact inspector makes their aggregate coverage and optional source evidence
// visible without reimplementing plan-specific target policy in GCT's UI.
export function BaselineCollectionsPanel({ repoId }: BaselineCollectionsPanelProps) {
  const [name, setName] = useState("");
  const [branch, setBranch] = useState("");
  const [collection, setCollection] = useState<BaselineCollection | undefined>();
  const [diff, setDiff] = useState<GetCollectionDiffStatusResponse | undefined>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const load = async () => {
    const trimmed = name.trim();
    if (!trimmed) {
      setError("Enter a collection name");
      return;
    }
    setLoading(true);
    setError("");
    setDiff(undefined);
    try {
      const result = await getBaselineCollection(trimmed, branch.trim(), repoId);
      setCollection(result);
      if (!result) setError("Collection was not found");
    } catch (cause) {
      setCollection(undefined);
      setError(errorMessage(cause));
    } finally {
      setLoading(false);
    }
  };

  const compare = async () => {
    if (!collection) return;
    setLoading(true);
    setError("");
    try {
      const result = await diffBaselineCollection(collection.name, collection.branch, repoId);
      setDiff(result);
      setCollection(result.collection ?? collection);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="rounded-lg border border-slate-800 bg-slate-950/30 p-3 space-y-3">
      <div className="flex items-center gap-2">
        <Layers className="h-4 w-4 text-sky-400" />
        <div>
          <h3 className="text-sm font-semibold text-slate-200">Baseline collection</h3>
          <p className="text-xs text-slate-500">Inspect plan-created coverage; source evidence stays informational.</p>
        </div>
      </div>
      <div className="grid gap-2 sm:grid-cols-[1fr_10rem_auto]">
        <input aria-label="Collection name" value={name} onChange={(event) => setName(event.target.value)} placeholder="Collection name" className="h-8 rounded border border-slate-700 bg-slate-900 px-2 text-sm text-slate-100" />
        <input aria-label="Collection branch" value={branch} onChange={(event) => setBranch(event.target.value)} placeholder="Branch (current)" className="h-8 rounded border border-slate-700 bg-slate-900 px-2 text-sm text-slate-100" />
        <Button size="sm" className="h-8" disabled={loading} onClick={() => void load()}>
          <RefreshCw className="mr-1 h-3.5 w-3.5" /> Load
        </Button>
      </div>
      {error && <p role="alert" className="text-xs text-rose-300">{error}</p>}
      {collection && (
        <div className="space-y-2 text-xs">
          <div className="flex flex-wrap items-center justify-between gap-2 text-slate-300">
            <span>{collection.name} · {collection.branch}</span>
            <span>coverage: {collection.coverage ? `${collection.coverage.ready}/${collection.coverage.required} ready` : "unavailable"}</span>
            <Button size="sm" variant="outline" className="h-7" disabled={loading} onClick={() => void compare()}>Diff complete set</Button>
          </div>
          <div className="grid gap-1 rounded bg-slate-900/70 p-2 text-slate-400 sm:grid-cols-2">
            {collection.members.map((member) => <span key={member.scenario}>{member.scenario}: {member.status}{member.required ? " (required)" : ""}</span>)}
          </div>
          {collection.pathSnapshots.length > 0 && <p className="text-amber-200">Informational source evidence: {collection.pathSnapshots.map((snapshot) => snapshot.name).join(", ")}</p>}
          {diff && <p className="text-slate-300">Collection verdict: <span className="font-medium">{diff.classification}</span> ({diff.members.length} members)</p>}
        </div>
      )}
    </section>
  );
}
