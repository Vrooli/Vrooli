import { useState, useCallback } from "react";
import { ChevronDown, ChevronRight, ChevronsDownUp, ChevronsUpDown } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { cn } from "../lib/utils";
import type { RunDiff } from "../types";

// =============================================================================
// Hunk parsing — pure function, tested separately
// =============================================================================

export interface HunkLine {
  type: "add" | "remove" | "context" | "no-newline";
  content: string;
  oldLine?: number;
  newLine?: number;
}

export interface Hunk {
  header: string;
  context: string;
  oldStart: number;
  newStart: number;
  lines: HunkLine[];
}

export function parseHunks(patch: string): Hunk[] {
  if (!patch) return [];
  const hunks: Hunk[] = [];
  let currentHunk: Hunk | null = null;
  let oldLine = 0;
  let newLine = 0;

  for (const raw of patch.split("\n")) {
    const hunkMatch = raw.match(/^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@(.*)/);
    if (hunkMatch) {
      currentHunk = {
        header: raw,
        context: hunkMatch[3]?.trim() || "",
        oldStart: parseInt(hunkMatch[1] ?? "0"),
        newStart: parseInt(hunkMatch[2] ?? "0"),
        lines: [],
      };
      oldLine = currentHunk.oldStart;
      newLine = currentHunk.newStart;
      hunks.push(currentHunk);
      continue;
    }

    if (!currentHunk) continue;

    if (raw.startsWith("\\ No newline")) {
      currentHunk.lines.push({ type: "no-newline", content: raw });
    } else if (raw.startsWith("+")) {
      currentHunk.lines.push({ type: "add", content: raw.slice(1), newLine: newLine++ });
    } else if (raw.startsWith("-")) {
      currentHunk.lines.push({ type: "remove", content: raw.slice(1), oldLine: oldLine++ });
    } else {
      // Context line (starts with space or is empty)
      currentHunk.lines.push({
        type: "context",
        content: raw.startsWith(" ") ? raw.slice(1) : raw,
        oldLine: oldLine++,
        newLine: newLine++,
      });
    }
  }

  return hunks;
}

// =============================================================================
// Props
// =============================================================================

interface DiffViewerProps {
  diff: RunDiff;
  /** Enable per-file checkboxes for partial approve */
  selectable?: boolean;
  selectedFiles?: Set<string>;
  onFileSelectionChange?: (path: string, selected: boolean) => void;
}

// =============================================================================
// Component
// =============================================================================

export function DiffViewer({ diff, selectable, selectedFiles, onFileSelectionChange }: DiffViewerProps) {
  const files = diff.files ?? [];
  const fileCount = files.length;
  const totals = files.reduce(
    (acc, file) => {
      acc.additions += file.additions || 0;
      acc.deletions += file.deletions || 0;
      return acc;
    },
    { additions: 0, deletions: 0 },
  );

  // All files expanded by default
  const [expandedFiles, setExpandedFiles] = useState<Set<string>>(
    () => new Set(files.map((f) => f.path)),
  );

  const toggleFile = useCallback((path: string) => {
    setExpandedFiles((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }, []);

  const expandAll = useCallback(() => {
    setExpandedFiles(new Set(files.map((f) => f.path)));
  }, [files]);

  const collapseAll = useCallback(() => {
    setExpandedFiles(new Set());
  }, []);

  const scrollToFile = useCallback(
    (path: string) => {
      // Expand if collapsed
      if (!expandedFiles.has(path)) toggleFile(path);
      // Defer scroll to let expansion render
      requestAnimationFrame(() => {
        const el = document.getElementById(`diff-file-${encodeURIComponent(path)}`);
        el?.scrollIntoView({ behavior: "smooth", block: "start" });
      });
    },
    [expandedFiles, toggleFile],
  );

  return (
    <div className="space-y-3">
      {/* Summary header */}
      <div className="rounded-lg border border-border bg-card/50 p-3 space-y-2">
        <div className="flex items-center gap-3 text-xs">
          <span className="text-success font-medium">+{totals.additions}</span>
          <span className="text-destructive font-medium">-{totals.deletions}</span>
          <span className="text-muted-foreground">{fileCount} file{fileCount !== 1 ? "s" : ""}</span>
          <div className="flex-1" />
          <Button variant="ghost" size="sm" className="h-6 px-2 text-xs gap-1" onClick={collapseAll}>
            <ChevronsDownUp className="h-3 w-3" /> Collapse
          </Button>
          <Button variant="ghost" size="sm" className="h-6 px-2 text-xs gap-1" onClick={expandAll}>
            <ChevronsUpDown className="h-3 w-3" /> Expand
          </Button>
        </div>

        {/* File TOC */}
        {fileCount > 0 && (
          <div className="space-y-0.5">
            {files.map((file) => (
              <div
                key={file.path}
                className="flex items-center gap-2 text-xs rounded px-2 py-1 hover:bg-muted/70 cursor-pointer transition-colors"
                onClick={() => scrollToFile(file.path)}
              >
                {selectable && (
                  <input
                    type="checkbox"
                    checked={selectedFiles?.has(file.path) ?? false}
                    onChange={(e) => {
                      e.stopPropagation();
                      onFileSelectionChange?.(file.path, e.target.checked);
                    }}
                    onClick={(e) => e.stopPropagation()}
                    className="h-3.5 w-3.5 rounded border-border"
                  />
                )}
                <Badge
                  variant={
                    file.changeType === "added"
                      ? "success"
                      : file.changeType === "deleted"
                        ? "destructive"
                        : "secondary"
                  }
                  className="text-[10px] px-1 shrink-0"
                >
                  {file.changeType}
                </Badge>
                <span className="font-mono truncate">{file.path}</span>
                <span className="text-success ml-auto shrink-0">+{file.additions}</span>
                <span className="text-destructive shrink-0">-{file.deletions}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Per-file patches */}
      {files.map((file) => {
        const expanded = expandedFiles.has(file.path);
        const hunks = file.isBinary ? [] : parseHunks(file.patch);

        return (
          <div
            key={file.path}
            id={`diff-file-${encodeURIComponent(file.path)}`}
            className="rounded-lg border border-border bg-card/50 overflow-hidden"
          >
            {/* File header */}
            <div
              className="flex items-center gap-2 px-3 py-2 bg-muted/30 cursor-pointer hover:bg-muted/50 transition-colors sticky top-0 z-10 border-b border-border"
              onClick={() => toggleFile(file.path)}
            >
              {expanded ? (
                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              )}
              {selectable && (
                <input
                  type="checkbox"
                  checked={selectedFiles?.has(file.path) ?? false}
                  onChange={(e) => {
                    e.stopPropagation();
                    onFileSelectionChange?.(file.path, e.target.checked);
                  }}
                  onClick={(e) => e.stopPropagation()}
                  className="h-3.5 w-3.5 rounded border-border"
                />
              )}
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
              <span className="font-mono text-xs truncate">{file.path}</span>
              <span className="text-xs text-success ml-auto shrink-0">+{file.additions}</span>
              <span className="text-xs text-destructive shrink-0">-{file.deletions}</span>
            </div>

            {/* File content */}
            {expanded && (
              <div className="overflow-x-auto">
                {file.isBinary ? (
                  <div className="py-4 text-center text-xs text-muted-foreground">Binary file</div>
                ) : hunks.length === 0 ? (
                  <div className="py-4 text-center text-xs text-muted-foreground">No changes</div>
                ) : (
                  <table className="w-full text-xs font-mono border-collapse">
                    <tbody>
                      {hunks.map((hunk, hi) => (
                        <HunkRows key={hi} hunk={hunk} />
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

// =============================================================================
// Hunk rendering
// =============================================================================

function HunkRows({ hunk }: { hunk: Hunk }) {
  return (
    <>
      <tr className="bg-primary/5">
        <td colSpan={3} className="px-3 py-1 text-primary/70 text-[11px] select-none">
          {hunk.header}
        </td>
      </tr>
      {hunk.lines.map((line, li) => (
        <tr
          key={li}
          className={cn(
            line.type === "add" && "diff-add",
            line.type === "remove" && "diff-remove",
            line.type === "no-newline" && "text-muted-foreground/60 italic",
          )}
        >
          <td className="select-none text-right text-muted-foreground/40 px-2 w-[1%] whitespace-nowrap border-r border-border/50 align-top">
            {line.oldLine ?? ""}
          </td>
          <td className="select-none text-right text-muted-foreground/40 px-2 w-[1%] whitespace-nowrap border-r border-border/50 align-top">
            {line.newLine ?? ""}
          </td>
          <td className="px-3 whitespace-pre-wrap break-all">{line.content}</td>
        </tr>
      ))}
    </>
  );
}
