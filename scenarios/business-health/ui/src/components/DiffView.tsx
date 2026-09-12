import { useMemo } from "react";

import { cn } from "../lib/utils";
import { diffLines, diffStats } from "../lib/diff";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { StatusChip } from "./StatusChip";
import { useTranslation } from "../i18n";

export interface DiffViewProps {
  readonly before: string;
  readonly after: string;
  /** Optional repo-relative path shown as a monospace header. */
  readonly path?: string;
  readonly "data-testid"?: string;
}

const LINE_CLASSES = {
  context: "text-app-muted-foreground",
  add: "bg-app-success/10 text-app-success",
  remove: "bg-app-danger/10 text-app-danger",
} as const;

const LINE_PREFIX = { context: " ", add: "+", remove: "-" } as const;

/**
 * Unified line-diff renderer for scaffold and fix previews. Pure and
 * dependency-free (see `lib/diff.ts`); computes the diff with `useMemo` so
 * re-renders don't re-run the LCS. Shows an added/removed summary and a
 * "new file" marker when `before` is empty.
 */
export function DiffView({ before, after, path, "data-testid": testId }: DiffViewProps) {
  const { t } = useTranslation();
  const lines = useMemo(() => diffLines(before, after), [before, after]);
  const stats = useMemo(() => diffStats(lines), [lines]);
  const unchanged = stats.added === 0 && stats.removed === 0;

  return (
    <div
      data-testid={testId ?? selectors.findings.fixDiff}
      className="overflow-hidden rounded-panel border border-app-border bg-app-surface"
    >
      <div className="flex flex-wrap items-center gap-2 border-b border-app-border bg-app-surface-muted px-3 py-1.5">
        {path && (
          <span className="min-w-0 flex-1 truncate font-mono text-xs text-app-foreground">
            {path}
          </span>
        )}
        {before === "" && after !== "" && (
          <StatusChip tone="info">{t(strings.diff.newFile)}</StatusChip>
        )}
        {unchanged ? (
          <span className="text-xs text-app-muted-foreground">{t(strings.diff.empty)}</span>
        ) : (
          <span className="flex items-center gap-2 font-mono text-xs">
            <span className="text-app-success">{t(strings.diff.added, { count: stats.added })}</span>
            <span className="text-app-danger">
              {t(strings.diff.removed, { count: stats.removed })}
            </span>
          </span>
        )}
      </div>
      <div className="max-h-80 overflow-auto font-mono text-xs leading-relaxed">
        {lines.map((line, index) => (
            <div
              key={`${line.op}-${index}`}
              className={cn("whitespace-pre px-3", LINE_CLASSES[line.op])}
            >
              <span aria-hidden="true" className="select-none opacity-60">
                {LINE_PREFIX[line.op]}
              </span>
              {line.text === "" ? " " : line.text}
            </div>
          ))}
      </div>
    </div>
  );
}
