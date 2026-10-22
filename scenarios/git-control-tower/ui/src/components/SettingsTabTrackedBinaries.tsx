import { useState } from "react";
import { Check, HardDrive, Info } from "lucide-react";
import { Button } from "./ui/button";
import { useTrackedBinaries, useUntrackBinary } from "../lib/hooks";
import type { TrackedBinary } from "../lib/api";

interface TrackedBinariesSectionProps {
  isMobile: boolean;
  repoId?: string | null;
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${units[unit]}`;
}

/**
 * Compiled executables committed to git. They are build output: large,
 * unmergeable, platform-specific, and stale the moment their source changes.
 *
 * The fix button removes the file from the index and adds it to the owning
 * scenario's .gitignore, so the next build does not re-stage it. It deliberately
 * does NOT claim to reclaim space -- the bytes stay in history until someone
 * rewrites it, and the panel says so rather than implying a cleanup that did
 * not happen.
 */
export function TrackedBinariesSection({ isMobile, repoId }: TrackedBinariesSectionProps) {
  const query = useTrackedBinaries(repoId);
  const untrack = useUntrackBinary(repoId);
  const [pendingPath, setPendingPath] = useState<string | null>(null);

  const textSm = isMobile ? "text-sm" : "text-xs";
  const textXs = isMobile ? "text-xs" : "text-[11px]";
  const py = isMobile ? "py-3" : "py-2";
  const px = isMobile ? "px-4" : "px-3";

  const handleUntrack = (binary: TrackedBinary) => {
    setPendingPath(binary.path);
    untrack.mutate(
      {
        path: binary.path,
        owner_dir: binary.owner_dir,
        ignore_pattern: binary.ignore_pattern,
      },
      { onSettled: () => setPendingPath(null) }
    );
  };

  const heading = <h3 className={`font-semibold text-slate-200 ${textSm}`}>Tracked files to remove</h3>;

  if (query.isLoading) {
    return (
      <div className="space-y-4">
        {heading}
        <p className={`${textXs} text-slate-500`}>Scanning the index...</p>
      </div>
    );
  }

  if (query.isError) {
    return (
      <div className="space-y-4">
        {heading}
        <p className={`${textXs} text-red-400`}>Failed to scan for tracked binaries: {query.error.message}</p>
      </div>
    );
  }

  const binaries = query.data?.binaries ?? [];

  if (binaries.length === 0) {
    return (
      <div className="space-y-4">
        {heading}
        <div className={`flex items-center gap-2 rounded-lg border border-green-900/50 bg-green-950/20 ${px} ${py}`}>
          <Check className="h-4 w-4 text-green-400 shrink-0" />
          <p className={`${textXs} text-green-300`}>No build output or oversized files are tracked in git.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        {heading}
        <span className={`${textXs} text-slate-500`}>
          {binaries.length} {binaries.length === 1 ? "file" : "files"} &middot; {formatBytes(query.data?.total_bytes ?? 0)}
        </span>
      </div>

      {query.data?.history_warning && (
        <div className={`flex items-start gap-2 rounded-lg border border-slate-800 bg-slate-900/40 ${px} ${py}`}>
          <Info className="h-4 w-4 text-slate-400 mt-0.5 shrink-0" />
          <p className={`${textXs} text-slate-400`}>{query.data.history_warning}</p>
        </div>
      )}

      <div className="space-y-2">
        {binaries.map((binary) => {
          const slash = binary.path.lastIndexOf("/");
          const dir = slash >= 0 ? binary.path.slice(0, slash + 1) : "";
          const name = slash >= 0 ? binary.path.slice(slash + 1) : binary.path;
          const ignoreTarget = binary.owner_dir ? `${binary.owner_dir}/.gitignore` : ".gitignore";
          const busy = pendingPath === binary.path || untrack.isPending;
          return (
            <div
              key={binary.path}
              className={`space-y-2 rounded-lg border border-amber-900/40 bg-amber-950/10 ${px} ${py}`}
            >
              <div className="flex items-start gap-2">
                <HardDrive className="h-4 w-4 text-amber-500 mt-0.5 shrink-0" />
                {/* Full path, filename emphasized; wraps instead of truncating so
                    every name stays readable on a narrow screen. */}
                <p className={`${textSm} min-w-0 flex-1 break-all font-mono leading-snug`} title={binary.path}>
                  <span className="text-slate-500">{dir}</span>
                  <span className="font-semibold text-slate-100">{name}</span>
                </p>
              </div>
              <p className={`flex flex-wrap items-center gap-x-1.5 gap-y-0.5 ${textXs} text-slate-500`}>
                <span className="font-medium text-slate-300">{formatBytes(binary.bytes)}</span>
                <span aria-hidden>&middot;</span>
                <span className="uppercase">{binary.format === "oversized" ? "large file" : binary.format}</span>
                <span aria-hidden>&middot;</span>
                <span className="break-all">ignore in {ignoreTarget}</span>
                {binary.already_ignored && <span className="text-slate-600">&middot; already ignored</span>}
              </p>
              <div className="flex justify-end">
                <Button
                  variant="outline"
                  size="sm"
                  className={`h-8 px-3 text-xs ${isMobile ? "w-full" : ""}`}
                  onClick={() => handleUntrack(binary)}
                  disabled={busy}
                >
                  {pendingPath === binary.path ? "Untracking…" : "Untrack & ignore"}
                </Button>
              </div>
            </div>
          );
        })}
      </div>

      {untrack.isError && (
        <p className={`${textXs} text-red-400`}>Untrack failed: {untrack.error.message}</p>
      )}
    </div>
  );
}
