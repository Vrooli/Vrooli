import { AlertTriangle, Clock3, GitBranch } from "lucide-react";
import type { RepoFileStats } from "../lib/api";

interface AdvisorySubjectBarProps {
  repoId?: string | null;
  scenarioSlug: string;
  fileStats?: RepoFileStats;
}

function fileCount(fileStats?: RepoFileStats): number {
  if (!fileStats) return 0;
  return new Set([
    ...Object.keys(fileStats.staged ?? {}),
    ...Object.keys(fileStats.unstaged ?? {}),
    ...Object.keys(fileStats.untracked ?? {}),
  ]).size;
}

export function AdvisorySubjectBar({ repoId, scenarioSlug, fileStats }: AdvisorySubjectBarProps) {
  const count = fileCount(fileStats);
  return (
    <section aria-label="Advisory subject" className="flex flex-wrap items-center gap-3 border-b border-slate-800 bg-slate-950/70 px-4 py-2 text-xs text-slate-300">
      <span className="inline-flex items-center gap-1.5 font-medium text-slate-100">
        <GitBranch className="h-3.5 w-3.5 text-sky-400" aria-hidden="true" />
        {repoId ? `Repository ${repoId}` : "Repository identity unavailable"}
      </span>
      <span className="text-slate-500">/</span>
      <span>Scope: {scenarioSlug} ({count} file{count === 1 ? "" : "s"})</span>
      <span className="inline-flex items-center gap-1 text-amber-300" title="Refresh before relying on an advisory result">
        <Clock3 className="h-3.5 w-3.5" aria-hidden="true" />
        Live selection; refresh before drafting
      </span>
      {!repoId && <span className="inline-flex items-center gap-1 text-amber-300"><AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />Exact subject required</span>}
    </section>
  );
}
