import { useMemo } from "react";
import { cn } from "../../lib/utils";

/**
 * Basic unified-diff renderer.
 *
 * Parses a unified-diff string (the output of `git diff`) into:
 *   - hunk headers (`@@ -1,4 +1,5 @@`)
 *   - file headers (`diff --git`, `--- a/…`, `+++ b/…`)
 *   - added lines (`+`)
 *   - removed lines (`-`)
 *   - context lines (` `)
 *
 * Renders each line as a styled row with line-type-keyed background. This
 * is intentionally minimal — Monaco's diff editor is the future upgrade
 * path. Surfaces pass the raw diff text; the viewer parses + renders.
 */
type DiffLineKind = "header" | "hunk" | "added" | "removed" | "context";

interface DiffLine {
  kind: DiffLineKind;
  text: string;
}

const classify = (line: string): DiffLineKind => {
  if (line.startsWith("diff --git") || line.startsWith("--- ") || line.startsWith("+++ ") || line.startsWith("index ")) {
    return "header";
  }
  if (line.startsWith("@@")) return "hunk";
  if (line.startsWith("+")) return "added";
  if (line.startsWith("-")) return "removed";
  return "context";
};

const kindClass: Record<DiffLineKind, string> = {
  header: "text-app-muted-foreground",
  hunk: "bg-app-surface-muted text-app-info",
  added: "bg-status-pass-bg/40 text-status-pass",
  removed: "bg-status-unexpected-bg/40 text-status-unexpected",
  context: "text-app-foreground",
};

export interface DiffViewerProps {
  /** Raw unified-diff text. */
  diff: string;
  className?: string;
  testId?: string;
}

export function DiffViewer({ diff, className, testId }: DiffViewerProps) {
  const lines = useMemo<DiffLine[]>(() => {
    if (!diff) return [];
    return diff.split("\n").map((text) => ({ kind: classify(text), text }));
  }, [diff]);

  if (lines.length === 0) {
    return (
      <div
        data-testid={testId}
        className={cn(
          "rounded-panel border border-app-border p-3 text-xs text-app-muted-foreground",
          className,
        )}
      />
    );
  }

  return (
    <pre
      data-testid={testId}
      className={cn(
        "overflow-auto rounded-panel border border-app-border bg-app-surface font-mono text-[12px] leading-5",
        className,
      )}
    >
      {lines.map((line, idx) => (
        <code
          key={idx}
          data-kind={line.kind}
          className={cn("block whitespace-pre px-3", kindClass[line.kind])}
        >
          {line.text || " "}
        </code>
      ))}
    </pre>
  );
}
