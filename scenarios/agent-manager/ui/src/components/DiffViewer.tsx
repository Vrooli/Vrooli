import { Badge } from "./ui/badge";
import { cn } from "../lib/utils";
import type { RunDiff } from "../types";

export function DiffViewer({ diff }: { diff: RunDiff }) {
  const fileCount = diff.files?.length ?? 0;
  const totals = (diff.files ?? []).reduce(
    (acc, file) => {
      acc.additions += file.additions || 0;
      acc.deletions += file.deletions || 0;
      return acc;
    },
    { additions: 0, deletions: 0 }
  );

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-border bg-card/50 p-4 space-y-3">
        <div className="flex gap-4 text-xs">
          <span className="text-success">+{totals.additions}</span>
          <span className="text-destructive">-{totals.deletions}</span>
          <span className="text-muted-foreground">{fileCount} files</span>
        </div>

        {diff.files && diff.files.length > 0 && (
          <div className="space-y-1">
            {diff.files.map((file) => (
              <div
                key={file.path}
                className="flex items-center gap-2 text-xs rounded px-2 py-1 bg-muted/50"
              >
                <Badge
                  variant={
                    file.changeType === "added"
                      ? "success"
                      : file.changeType === "deleted"
                      ? "destructive"
                      : "secondary"
                  }
                  className="text-[10px] px-1"
                >
                  {file.changeType}
                </Badge>
                <span className="font-mono truncate">{file.path}</span>
                <span className="text-success">+{file.additions}</span>
                <span className="text-destructive">-{file.deletions}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {diff.content && (
        <div className="rounded-lg border border-border bg-card/50 p-4">
          <pre className="text-[10px] font-mono bg-muted/50 rounded p-3 overflow-x-auto whitespace-pre-wrap">
            {diff.content.split("\n").map((line, i) => (
              <div
                key={i}
                className={cn(
                  line.startsWith("+") && !line.startsWith("+++")
                    ? "diff-add"
                    : line.startsWith("-") && !line.startsWith("---")
                    ? "diff-remove"
                    : line.startsWith("@@")
                    ? "text-primary"
                    : "diff-context"
                )}
              >
                {line}
              </div>
            ))}
          </pre>
        </div>
      )}
    </div>
  );
}
