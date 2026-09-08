import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import type { PushSafetyReport } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { inspectPushSafety } from "../lib/api-push-safety";

type Safety = { report?: PushSafetyReport; label: string; review?: () => void; checking?: boolean };
const Context = createContext<Safety>({ label: "Push safety unchecked" });
export const usePushSafety = () => useContext(Context);

export function PushSafetyProvider({ repoId, revision, review, children }: { repoId?: string; revision: string; review: () => void; children: ReactNode }) {
  const [settled, setSettled] = useState(revision);
  const [now, setNow] = useState(Date.now());
  useEffect(() => { const timer = setTimeout(() => setSettled(revision), 1000); return () => clearTimeout(timer); }, [revision]);
  useEffect(() => { const timer = setInterval(() => setNow(Date.now()), 5000); return () => clearInterval(timer); }, []);
  const query = useQuery({ queryKey: ["push-safety-indicators", repoId, settled], queryFn: ({ signal }) => inspectPushSafety(repoId, signal), retry: false, staleTime: 60000, refetchInterval: 60000, gcTime: 60000 });
  const stale = revision !== settled || now - query.dataUpdatedAt >= 60000;
  const report = !stale && !query.isFetching && !query.isError ? query.data : undefined;
  const count = new Set(report?.files.filter(f => f.blocked).flatMap(f => f.paths)).size;
  const label = stale && query.data ? "Push safety check stale" : query.isFetching ? "Checking push safety…" : !report?.complete || report.state === "unknown" ? "Push safety unverified" : report.state === "blocked" ? `Push blocked: ${count} oversized file${count === 1 ? "" : "s"}` : "File-size check passed for this snapshot";
  return <Context.Provider value={{ report, label, review, checking: query.isFetching }}><div className="sr-only" role="status">{label}</div>{children}</Context.Provider>;
}

export function PushSafetyNotice() {
  const { label, review } = usePushSafety();
  if (!review) return null;
  return <button type="button" onClick={review} className="block max-w-[16rem] whitespace-normal text-xs text-amber-200 text-left" title="Read-only review. Nothing is rewritten or pushed by opening this screen.">{label} · Review recovery options</button>;
}

export function StagedSafetyNotice() {
  const { report, review } = usePushSafety();
  if (!review) return null;
  return <div className="px-3 py-2 text-xs border-b border-slate-800" aria-label="Staged file-size check">
    <p>{!report?.stagedComplete ? "Staged file-size check unverified" : report.limit === 0n ? "Staged sizes checked; destination limit unknown" : "Staged file sizes checked for this snapshot"}</p>
    {report?.stagedReason && <p className="text-slate-400">{report.stagedReason}</p>}
    {report?.stagedFiles?.map(file => <p key={file.oid + file.paths.join()} className="text-amber-200 break-all">{file.paths.join(", ")} · {(Number(file.bytes) / 1048576).toFixed(2)} MiB · {file.blocked ? "Would block push if committed" : "Large staged file — review destination policy"}</p>)}
    {!!report?.stagedFiles?.length && <p>These are staged bytes, which may differ from your working file. You can commit locally. Committing does not fix a push blocker.</p>}
    <button type="button" className="underline text-amber-200" onClick={review}>Review checks and recovery options</button>
  </div>;
}

export function historySafetyLabel(report: PushSafetyReport | undefined, hash: string, unpushed: boolean) {
  if (!report?.complete || report.state === "unknown") return unpushed ? "Push safety unverified" : undefined;
  const matches = report.commits.filter(commit => commit === hash || (hash.length >= 7 && commit.startsWith(hash)));
  if (matches.length !== 1) return unpushed ? "Outside checked outgoing history" : undefined;
  const commit = matches[0];
  if (!commit) return "Push safety unverified";
  if (report.files.some(file => file.blocked && file.commits[0] === commit)) return "Introduces push blocker";
  if (!report.canPrepare && report.files.some(file => file.blocked && file.commits.includes(commit))) return "Contains push blocker";
  // Only linear history has a verified ancestor order. Merge siblings must not
  // be described as descendants merely because rev-list orders them later.
  if (report.state === "blocked") {
    if (!report.canPrepare) return "Outgoing push contains a blocker";
    const index = report.commits.indexOf(commit);
    if (report.files.some(file => file.blocked && report.commits.indexOf(file.commits[0] ?? "") >= 0 && report.commits.indexOf(file.commits[0] ?? "") < index)) return "Push blocked by earlier commit";
  }
  return "File-size check passed";
}

export function HistorySafetyBadge({ hash, unpushed }: { hash: string; unpushed: boolean }) {
  const { report, review } = usePushSafety();
  const label = historySafetyLabel(report, hash, unpushed);
  if (!label || !review) return null;
  return <button type="button" className="rounded border border-amber-500/30 px-1 text-amber-200 text-[10px]" onClick={event => { event.stopPropagation(); review(); }} title="File-size policy check for the outgoing snapshot; this is separate from merge conflicts. Opens a read-only review.">{label}</button>;
}

export function StagedSafetyBadge({ path }: { path: string }) {
  const { report, review } = usePushSafety();
  const file = report?.stagedComplete ? report.stagedFiles.find(file => file.paths.includes(path)) : undefined;
  if (!file) return null;
  return <button type="button" className="shrink-0 text-xs text-amber-200" title={`${(Number(file.bytes) / 1048576).toFixed(2)} MiB in the index. ${file.blocked ? "Would block push if committed." : "Large staged file."}`} onClick={event => { event.stopPropagation(); review?.(); }}>{file.blocked ? "Push blocker" : "Large file"}</button>;
}

// The report stores paths and commits separately per blob, not exact
// path/commit pairs. Do not claim a renamed or deleted path contains that blob
// in the selected tree. Identify its association with outgoing history instead.
export function historyPathBlockers(report: PushSafetyReport | undefined, hash: string, path: string) {
  if (!report?.complete || report.state !== "blocked" || hash.length < 7) return [];
  if (report.commits.filter(commit => commit === hash || commit.startsWith(hash)).length !== 1) return [];
  return report.files.filter(file => file.blocked && file.paths.includes(path));
}

export function HistoryFileSafetyBadge({ hash, path }: { hash: string; path: string }) {
  const { report, review } = usePushSafety();
  const files = historyPathBlockers(report, hash, path);
  if (!files.length || !review) return null;
  const sizes = [...new Set(files.map(file => `${(Number(file.bytes) / 1048576).toFixed(2)} MiB`))].join(", ");
  return <button type="button" className="mt-1 block text-left text-xs text-amber-200 whitespace-normal"
    onClick={event => { event.stopPropagation(); review(); }}
    title="This path is associated with oversized outgoing history. It may have changed, moved, or been deleted in this commit. Opening review does not change files or history.">
    Push blocker in outgoing history · {sizes}
  </button>;
}
