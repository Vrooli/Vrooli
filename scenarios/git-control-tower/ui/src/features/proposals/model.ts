import {
  FileChangeKind,
  FreshnessState,
  TrailerResolution,
  type Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";

type Tone = "success" | "warning" | "error" | "info" | "default";

export function freshnessMeta(state: FreshnessState | undefined): { label: string; tone: Tone; hint: string } {
  switch (state) {
    case FreshnessState.FRESH:
      return { label: "Fresh", tone: "success", hint: "Files still hold the proposed content" };
    case FreshnessState.DRIFTED:
      return { label: "Drifted", tone: "warning", hint: "Some files changed after the proposal; refresh before approving" };
    case FreshnessState.BASE_MOVED:
      return { label: "Base moved", tone: "warning", hint: "New commits touch proposal files; refresh before approving" };
    case FreshnessState.UNKNOWN:
      return { label: "Unknown", tone: "default", hint: "The repository could not be read" };
    default:
      return { label: "Closed", tone: "default", hint: "The proposal is no longer open" };
  }
}

const FLAG_LABELS: Record<string, { label: string; hint: string; tone: Tone }> = {
  mixed_prior_uncommitted: { label: "mixed", hint: "Dirty before the epoch began: the whole-file commit carries older uncommitted work", tone: "warning" },
  other_open_proposal: { label: "shared", hint: "Another open proposal includes this file", tone: "warning" },
  sandbox_pending: { label: "sandbox", hint: "Workspace Sandbox has applied changes to this file that are not committed", tone: "info" },
  already_staged: { label: "staged", hint: "Already staged in the index", tone: "info" },
  no_anchor: { label: "no anchor", hint: "No admission anchor existed, so older work could not be separated", tone: "default" },
  outside_anchor_scope: { label: "out of scope", hint: "Outside the anchored scope", tone: "default" },
};

export function flagMeta(code: string): { label: string; hint: string; tone: Tone } {
  return FLAG_LABELS[code] ?? { label: code.replace(/_/g, " "), hint: code, tone: "default" };
}

/** Flag counts across files, in first-seen order. */
export function flagCounts(proposal: Proposal): Array<{ code: string; count: number }> {
  const counts = new Map<string, number>();
  for (const file of proposal.files) {
    for (const flag of file.flags) {
      counts.set(flag.code, (counts.get(flag.code) ?? 0) + 1);
    }
  }
  return Array.from(counts, ([code, count]) => ({ code, count }));
}

export function kindMarker(kind: FileChangeKind): { letter: string; className: string; label: string } {
  switch (kind) {
    case FileChangeKind.ADDED:
      return { letter: "A", className: "text-emerald-400", label: "added" };
    case FileChangeKind.DELETED:
      return { letter: "D", className: "text-red-400", label: "deleted" };
    default:
      return { letter: "M", className: "text-amber-400", label: "modified" };
  }
}

export function resolutionMeta(resolution: TrailerResolution): { label: string; className: string } {
  switch (resolution) {
    case TrailerResolution.RESOLVED:
      return { label: "resolved", className: "border-emerald-800/70 bg-emerald-950/40 text-emerald-200" };
    case TrailerResolution.LEGACY:
      return { label: "legacy", className: "border-slate-700 bg-slate-900 text-slate-400" };
    case TrailerResolution.NOT_APPLICABLE:
      return { label: "kept", className: "border-slate-700 bg-slate-900 text-slate-300" };
    case TrailerResolution.INACCESSIBLE:
      return { label: "private", className: "border-slate-700 bg-slate-900 text-slate-400" };
    case TrailerResolution.DELETED:
      return { label: "deleted", className: "border-red-900/70 bg-red-950/30 text-red-300" };
    case TrailerResolution.AMBIGUOUS:
      return { label: "ambiguous", className: "border-amber-800/70 bg-amber-950/30 text-amber-200" };
    default:
      return { label: "unresolved", className: "border-blue-900/70 bg-blue-950/30 text-blue-200" };
  }
}

/** "effort:browser-automation-studio-rehabilitation" -> "browser-automation-studio-rehabilitation". */
export function effortLabel(effortRef: string): string {
  return effortRef.replace(/^effort:/, "");
}

export function shortOid(oid: string): string {
  return oid ? oid.slice(0, 10) : "";
}
