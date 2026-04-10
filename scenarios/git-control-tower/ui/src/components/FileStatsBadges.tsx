import { Badge } from "./ui/badge";
import { Popover } from "./ui/popover";

interface FileStatsBadgesProps {
  staged: number;
  unstaged: number;
  untracked: number;
  conflicts: number;
  cleanDetails?: string;
  /** "full" shows labeled badges; "compact" shows colored dot-counts with a click-to-expand popover */
  variant?: "full" | "compact";
}

const DOT_COLORS = {
  staged: "bg-emerald-400",
  unstaged: "bg-amber-400",
  untracked: "bg-slate-400",
  conflicts: "bg-red-400",
} as const;

const LABELS: Record<string, string> = {
  staged: "staged",
  unstaged: "modified",
  untracked: "untracked",
  conflicts: "conflicts",
};

type StatKey = keyof typeof DOT_COLORS;

export function FileStatsBadges({
  staged,
  unstaged,
  untracked,
  conflicts,
  cleanDetails,
  variant = "full",
}: FileStatsBadgesProps) {
  const isClean = staged === 0 && unstaged === 0 && untracked === 0 && conflicts === 0;

  if (isClean) {
    return (
      <div className="flex items-center gap-3" data-testid="file-stats">
        <span className="text-xs text-slate-500">
          {cleanDetails ? `Working tree clean (${cleanDetails})` : "Working tree clean"}
        </span>
      </div>
    );
  }

  if (variant === "full") {
    return (
      <div className="flex items-center gap-3" data-testid="file-stats">
        {staged > 0 && <Badge variant="staged">{staged} staged</Badge>}
        {unstaged > 0 && <Badge variant="unstaged">{unstaged} modified</Badge>}
        {untracked > 0 && <Badge variant="untracked">{untracked} untracked</Badge>}
        {conflicts > 0 && <Badge variant="conflict">{conflicts} conflicts</Badge>}
      </div>
    );
  }

  // Compact variant: colored dots with counts, click to expand
  const all: { key: StatKey; count: number }[] = [
    { key: "staged" as const, count: staged },
    { key: "unstaged" as const, count: unstaged },
    { key: "untracked" as const, count: untracked },
    { key: "conflicts" as const, count: conflicts },
  ];
  const entries = all.filter((e) => e.count > 0);

  const compactDisplay = (
    <span
      className="inline-flex items-center gap-2 cursor-pointer rounded-md px-2 py-1 hover:bg-slate-800 transition-colors"
      data-testid="file-stats"
    >
      {entries.map((entry, i) => (
        <span key={entry.key} className="inline-flex items-center gap-1">
          {i > 0 && <span className="text-slate-600 text-[10px] mx-0.5">·</span>}
          <span className={`inline-block h-2 w-2 rounded-full ${DOT_COLORS[entry.key]}`} />
          <span
            className={`text-xs tabular-nums font-medium ${
              entry.key === "conflicts" ? "text-red-400" : "text-slate-300"
            }`}
          >
            {entry.count}
          </span>
          {entry.key === "conflicts" && (
            <span className="text-[11px] text-red-400 font-medium">!</span>
          )}
        </span>
      ))}
    </span>
  );

  return (
    <Popover trigger={compactDisplay} align="end">
      <div className="p-3 space-y-2" data-testid="file-stats-popover">
        <div className="text-[11px] uppercase tracking-wide text-slate-500 mb-1">Changes</div>
        {entries.map((entry) => (
          <div key={entry.key} className="flex items-center gap-2.5">
            <span className={`inline-block h-2 w-2 rounded-full ${DOT_COLORS[entry.key]}`} />
            <span className="text-xs text-slate-300 flex-1">{LABELS[entry.key]}</span>
            <span
              className={`text-xs tabular-nums font-medium ${
                entry.key === "conflicts" ? "text-red-400" : "text-slate-200"
              }`}
            >
              {entry.count}
            </span>
          </div>
        ))}
      </div>
    </Popover>
  );
}
