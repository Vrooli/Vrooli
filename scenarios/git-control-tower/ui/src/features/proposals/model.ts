import {
  FileChangeKind,
  FreshnessState,
  TrailerResolution,
  type Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";

type Tone = "success" | "warning" | "error" | "info" | "default";

export function freshnessMeta(state: FreshnessState | undefined): {
  label: string;
  tone: Tone;
  hint: string;
} {
  switch (state) {
    case FreshnessState.FRESH:
      return {
        label: "Unchanged",
        tone: "success",
        hint: "Working files match the saved proposal version; this does not establish attribution",
      };
    case FreshnessState.DRIFTED:
      return {
        label: "Newer edits",
        tone: "warning",
        hint: "These files have newer working edits. You can commit the proposal version and keep those edits uncommitted, or update the proposal to review the latest files.",
      };
    case FreshnessState.BASE_MOVED:
      return {
        label: "Base changed",
        tone: "warning",
        hint: "New commits touch these files. Compare them and update the proposal before committing.",
      };
    case FreshnessState.UNKNOWN:
      return {
        label: "Unknown",
        tone: "default",
        hint: "Approval standing could not be confirmed. See the reason below.",
      };
    default:
      return {
        label: "Closed",
        tone: "default",
        hint: "The proposal is no longer open",
      };
  }
}

export function kindMarker(kind: FileChangeKind): {
  letter: string;
  className: string;
  label: string;
} {
  switch (kind) {
    case FileChangeKind.ADDED:
      return { letter: "A", className: "text-emerald-400", label: "added" };
    case FileChangeKind.DELETED:
      return { letter: "D", className: "text-red-400", label: "deleted" };
    default:
      return { letter: "M", className: "text-amber-400", label: "modified" };
  }
}

export function resolutionMeta(resolution: TrailerResolution): {
  label: string;
  className: string;
} {
  switch (resolution) {
    case TrailerResolution.RESOLVED:
      return {
        label: "resolved",
        className: "border-emerald-800/70 bg-emerald-950/40 text-emerald-200",
      };
    case TrailerResolution.LEGACY:
      return {
        label: "legacy",
        className: "border-slate-700 bg-slate-900 text-slate-400",
      };
    case TrailerResolution.NOT_APPLICABLE:
      return {
        label: "kept",
        className: "border-slate-700 bg-slate-900 text-slate-300",
      };
    case TrailerResolution.INACCESSIBLE:
      return {
        label: "private",
        className: "border-slate-700 bg-slate-900 text-slate-400",
      };
    case TrailerResolution.DELETED:
      return {
        label: "deleted",
        className: "border-red-900/70 bg-red-950/30 text-red-300",
      };
    case TrailerResolution.AMBIGUOUS:
      return {
        label: "ambiguous",
        className: "border-amber-800/70 bg-amber-950/30 text-amber-200",
      };
    default:
      return {
        label: "unresolved",
        className: "border-blue-900/70 bg-blue-950/30 text-blue-200",
      };
  }
}

/** "effort:browser-automation-studio-rehabilitation" -> "browser-automation-studio-rehabilitation". */
export function effortLabel(effortRef: string): string {
  return effortRef.replace(/^effort:/, "");
}

export function shortOid(oid: string): string {
  return oid ? oid.slice(0, 10) : "";
}

/** Epochs a proposal carries, oldest first, from its Vrooli-Epoch trailers.
 *  A proposal that folded earlier open proposals of its effort carries
 *  several; others carry their own epoch. */
export function proposalEpochs(proposal: Proposal): string[] {
  const epochs: string[] = [];
  for (const trailer of proposal.message?.trailers ?? []) {
    const epoch =
      trailer.key.toLowerCase() === "vrooli-epoch"
        ? /#(E\d+)$/.exec(trailer.value.trim())?.[1]
        : undefined;
    if (epoch && !epochs.includes(epoch)) epochs.push(epoch);
  }
  const own = proposal.work?.epoch;
  if (own && !epochs.includes(own)) epochs.push(own);
  return epochs;
}

/** "E27", "E28–E33" for a consecutive run, otherwise "E28, E30, E33". */
export function epochsLabel(epochs: string[]): string {
  if (epochs.length <= 1) return epochs[0] ?? "";
  const numbers = epochs.map((epoch) => Number(epoch.slice(1)));
  const consecutive = numbers.every(
    (value, index) =>
      index === 0 || value === (numbers[index - 1] ?? Number.NaN) + 1,
  );
  return consecutive
    ? `${epochs[0]}–${epochs[epochs.length - 1]}`
    : epochs.join(", ");
}

/** Earlier/unattributed whole-file content requires deliberate operator review. */
export function hasPriorWork(
  file: Pick<Proposal["files"][number], "flags">,
): boolean {
  return file.flags.some((flag) =>
    ["mixed_prior_uncommitted", "no_anchor", "outside_anchor_scope"].includes(
      flag.code,
    ),
  );
}

/** Plain-language review metadata shared by file rows and the captured diff. */
export function describeFlag(code: string): string {
  switch (code) {
    case "mixed_prior_uncommitted":
      return "Includes earlier uncommitted work";
    case "no_anchor":
      return "Earlier work could not be separated";
    case "outside_anchor_scope":
      return "Outside the epoch's anchored scope";
    case "other_open_proposal":
      return "Shared with another open proposal";
    case "sandbox_pending":
      return "Includes pending sandbox changes";
    case "staged_version_differs":
      return "A different version is staged";
    case "already_staged":
      return "Already staged";
    default:
      return code.replace(/_/g, " ");
  }
}
