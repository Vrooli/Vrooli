import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import type { PushSafetyReport } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { inspectPushSafety } from "../lib/api-push-safety";

type ReviewFocus = "staged" | "outgoing";
type Safety = { stagedFilesByPath?: ReadonlyMap<string, PushSafetyReport["stagedFiles"][number]>; report?: PushSafetyReport; label: string; review?: (focus?: ReviewFocus) => void; checking?: boolean };
const Context = createContext<Safety>({ label: "" });
export const usePushSafety = () => useContext(Context);

const mib = (n: bigint) => `${(Number(n) / 1048576).toFixed(2)} MiB`;
const basename = (path: string) => path.split("/").pop() || path;

export function PushSafetyProvider({ repoId, revision, review, paused = false, children }: { repoId?: string; revision: string; paused?: boolean; review: (focus?: ReviewFocus) => void; children: ReactNode }) {
  const [settled, setSettled] = useState(revision);
  const [now, setNow] = useState(Date.now());
  useEffect(() => { const timer = setTimeout(() => setSettled(revision), 1000); return () => clearTimeout(timer); }, [revision]);
  useEffect(() => { const timer = setInterval(() => setNow(Date.now()), 5000); return () => clearInterval(timer); }, []);
  const query = useQuery({ queryKey: ["push-safety-indicators", repoId, settled], queryFn: ({ signal }) => inspectPushSafety(repoId, signal), enabled: !paused, retry: false, staleTime: 60000, refetchInterval: paused ? false : 60000, gcTime: 60000 });
  const stale = paused || revision !== settled || now - query.dataUpdatedAt >= 60000;
  const report = !stale && !query.isFetching && !query.isError ? query.data : undefined;
  const count = new Set(report?.files.filter(f => f.blocked).flatMap(f => f.paths)).size;
  const largeCount = new Set(report?.files.flatMap(f => f.paths)).size;
  // Routine refreshes and successful checks stay silent. Unknown evidence is
  // surfaced once near Push; individual rows only identify actual findings.
  const label = query.isFetching || (stale && !query.isError) ? "" : query.isError || !report?.complete || report.state === "unknown"
    ? "Couldn’t verify push file sizes"
    : report.state === "blocked" ? `Push blocked: ${count} oversized file${count === 1 ? "" : "s"}`
    : largeCount ? `${largeCount} large file${largeCount === 1 ? "" : "s"} in outgoing commits` : "";
  const stagedFilesByPath = useMemo(() => {
    const index = new Map<string, PushSafetyReport["stagedFiles"][number]>();
    if (report?.stagedComplete) {
      for (const file of report.stagedFiles) {
        for (const path of file.paths) if (!index.has(path)) index.set(path, file);
      }
    }
    return index;
  }, [report]);
  const checking = query.isFetching;
  const value = useMemo(() => ({ report, label, review, checking, stagedFilesByPath }), [report, label, review, checking, stagedFilesByPath]);
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function PushSafetyNotice() {
  const { label, review } = usePushSafety();
  if (!review || !label) return null;
  return <button type="button" onClick={() => review()} className="block max-w-[16rem] whitespace-normal text-xs text-amber-200 text-left" title="Review file-size details">{label} · Review</button>;
}

export function StagedSafetyNotice() {
  const { report, review } = usePushSafety();
  if (!review || !report?.stagedComplete || !report.stagedFiles.length) return null;
  const count = new Set(report.stagedFiles.flatMap(file => file.paths)).size;
  const blocked = report.stagedFiles.filter(file => file.blocked);
  // Name the single largest file and its size against the limit so the operator
  // can identify and act on the offender without opening the drawer.
  const largest = report.stagedFiles.reduce((a, b) => (b.bytes > a.bytes ? b : a));
  const name = basename(largest.paths[0] ?? "staged file");
  const limit = report.limit > 0n ? `the ${mib(report.limit)} push limit` : "the push limit";
  const message = blocked.length
    ? blocked.length === 1
      ? `${name} (${mib(largest.bytes)}) exceeds ${limit} and would block push — unstage before committing`
      : `${blocked.length} staged files exceed ${limit} (largest ${name}, ${mib(largest.bytes)}) — unstage before committing`
    : count === 1
      ? `${name} (${mib(largest.bytes)}) is large — review before committing`
      : `${count} large staged files (largest ${name}, ${mib(largest.bytes)}) — review before committing`;
  return <button type="button" className="block max-w-[18rem] whitespace-normal text-left text-xs text-amber-200" title="Review staged file sizes before committing" onClick={() => review("staged")}>
    {message} · Review
  </button>;
}

export function historySafetyLabel(report: PushSafetyReport | undefined, hash: string, unpushed: boolean) {
  if (!unpushed || !report?.complete || report.state === "unknown") return undefined;
  const matches = report.commits.filter(commit => commit === hash || (hash.length >= 7 && commit.startsWith(hash)));
  if (matches.length !== 1) return undefined;
  const commit = matches[0];
  if (report.files.some(file => file.blocked && file.commits[0] === commit)) return "Introduces push blocker";
  // The push summary explains inherited blockers once, rather than repeating
  // the same warning on every descendant commit.
  return undefined;
}

export function HistorySafetyBadge({ hash, unpushed }: { hash: string; unpushed: boolean }) {
  const { report, review } = usePushSafety();
  const label = historySafetyLabel(report, hash, unpushed);
  if (!label || !review) return null;
  return <button type="button" className="rounded border border-amber-500/30 px-1 text-amber-200 text-[10px]" onClick={event => { event.stopPropagation(); review(); }} title="File-size policy check for the outgoing snapshot; this is separate from merge conflicts. Opens a read-only review.">{label}</button>;
}

export function StagedSafetyBadge({ path }: { path: string }) {
  const { stagedFilesByPath, review } = usePushSafety();
  const file = stagedFilesByPath?.get(path);
  if (!file) return null;
  return <button type="button" className="shrink-0 text-xs text-amber-200" title={`${mib(file.bytes)} in the index. ${file.blocked ? "Would block push if committed." : "Large staged file."}`} onClick={event => { event.stopPropagation(); review?.("staged"); }}>{file.blocked ? "Push blocker" : "Large file"}</button>;
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
