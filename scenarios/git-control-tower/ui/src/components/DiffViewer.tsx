import { useEffect, useState } from "react";
import { FileDiff, Plus, Minus, Loader2, AlertTriangle, Copy, Check } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Badge } from "./ui/badge";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import type { DiffResponse, DiffHunk } from "../lib/api";

interface DiffViewerProps {
  diff?: DiffResponse;
  selectedFile?: string;
  isStaged: boolean;
  isLoading: boolean;
  error?: Error | null;
  repoDir?: string;
}

function DiffLine({ line, lineNumber }: { line: string; lineNumber?: number }) {
  const isAddition = line.startsWith("+") && !line.startsWith("+++");
  const isDeletion = line.startsWith("-") && !line.startsWith("---");
  const isHeader = line.startsWith("@@");
  const isContext = !isAddition && !isDeletion && !isHeader;

  let bgColor = "";
  let textColor = "text-slate-300";
  let lineNumColor = "text-slate-600";

  if (isAddition) {
    bgColor = "bg-emerald-950/30";
    textColor = "text-emerald-300";
    lineNumColor = "text-emerald-700";
  } else if (isDeletion) {
    bgColor = "bg-red-950/30";
    textColor = "text-red-300";
    lineNumColor = "text-red-700";
  } else if (isHeader) {
    bgColor = "bg-blue-950/30";
    textColor = "text-blue-400";
    lineNumColor = "text-blue-700";
  }

  return (
    <div className={`flex font-mono text-xs ${bgColor}`} data-testid="diff-line">
      {lineNumber !== undefined && (
        <span
          className={`w-12 flex-shrink-0 px-2 py-0.5 text-right select-none border-r border-slate-800 ${lineNumColor}`}
        >
          {isContext ? lineNumber : ""}
        </span>
      )}
      <pre className={`flex-1 px-3 py-0.5 whitespace-pre overflow-x-auto ${textColor}`}>
        {line || " "}
      </pre>
    </div>
  );
}

function HunkDisplay({ hunk, index }: { hunk: DiffHunk; index: number }) {
  let currentLine = hunk.new_start;

  return (
    <div className="border-b border-slate-800 last:border-b-0" data-testid={`diff-hunk-${index}`}>
      {/* Hunk header */}
      <div className="bg-slate-800/50 px-3 py-1.5 font-mono text-xs text-slate-500">
        {hunk.header}
      </div>

      {/* Hunk lines */}
      <div className="divide-y divide-slate-800/30">
        {hunk.lines.map((line, lineIdx) => {
          const isAddition = line.startsWith("+") && !line.startsWith("+++");
          const isDeletion = line.startsWith("-") && !line.startsWith("---");
          const lineNum = isDeletion ? undefined : currentLine;

          if (!isDeletion && !line.startsWith("@@")) {
            currentLine++;
          }

          return (
            <DiffLine
              key={`${index}-${lineIdx}`}
              line={line}
              lineNumber={lineNum}
            />
          );
        })}
      </div>
    </div>
  );
}

export function DiffViewer({
  diff,
  selectedFile,
  isStaged,
  isLoading,
  error,
  repoDir
}: DiffViewerProps) {
  const [showBinary, setShowBinary] = useState(false);
  const [copied, setCopied] = useState(false);
  const isBinaryDiff = Boolean(
    diff?.raw && (diff.raw.includes("Binary files") || diff.raw.includes("GIT binary patch"))
  );
  const absolutePath =
    selectedFile && repoDir ? `${repoDir.replace(/\/$/, "")}/${selectedFile}` : selectedFile;

  useEffect(() => {
    setShowBinary(false);
  }, [selectedFile, diff?.raw]);

  useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), 1500);
    return () => window.clearTimeout(timer);
  }, [copied]);

  const handleCopyPath = async () => {
    if (!absolutePath) return;

    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(absolutePath);
        setCopied(true);
        return;
      } catch {
        // Fallback to legacy copy.
      }
    }

    try {
      const textarea = document.createElement("textarea");
      textarea.value = absolutePath;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(true);
    } catch {
      // Ignore copy errors.
    }
  };

  const showBinaryNotice =
    selectedFile && !isLoading && !error && diff?.has_diff && isBinaryDiff && !showBinary;

  return (
    <Card className="h-full flex flex-col" data-testid="diff-viewer-panel">
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3">
        <div className="flex items-center gap-3">
          <CardTitle className="flex items-center gap-2">
            <FileDiff className="h-4 w-4 text-slate-500" />
            {selectedFile ? (
              <span className="font-mono text-xs">{selectedFile}</span>
            ) : (
              "Diff Viewer"
            )}
          </CardTitle>
          {selectedFile && (
            <button
              type="button"
              className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-white/20 text-slate-300 transition-colors hover:bg-white/10"
              onClick={handleCopyPath}
              title={copied ? "Copied" : "Copy absolute path"}
              aria-label="Copy absolute path"
              data-testid="copy-absolute-path"
            >
              {copied ? (
                <Check className="h-3.5 w-3.5 text-emerald-300" />
              ) : (
                <Copy className="h-3.5 w-3.5" />
              )}
            </button>
          )}
          {selectedFile && (
            <Badge variant={isStaged ? "staged" : "unstaged"}>
              {isStaged ? "staged" : "unstaged"}
            </Badge>
          )}
        </div>

        {diff?.stats && diff.has_diff && (
          <div className="flex items-center gap-3" data-testid="diff-stats">
            <span className="flex items-center gap-1 text-xs text-emerald-500">
              <Plus className="h-3 w-3" />
              {diff.stats.additions}
            </span>
            <span className="flex items-center gap-1 text-xs text-red-500">
              <Minus className="h-3 w-3" />
              {diff.stats.deletions}
            </span>
          </div>
        )}
      </CardHeader>

      <CardContent className="flex-1 p-0 overflow-hidden">
        <ScrollArea className="h-full">
          {/* Loading State */}
          {isLoading && (
            <div className="flex items-center justify-center py-12" data-testid="diff-loading">
              <Loader2 className="h-6 w-6 animate-spin text-slate-500" />
            </div>
          )}

          {/* Error State */}
          {error && !isLoading && (
            <div className="flex flex-col items-center justify-center py-12 text-center px-4" data-testid="diff-error">
              <p className="text-sm text-red-400">{error.message}</p>
            </div>
          )}

          {/* Empty State - No file selected */}
          {!selectedFile && !isLoading && !error && (
            <div className="flex flex-col items-center justify-center py-12 text-center" data-testid="diff-empty">
              <FileDiff className="h-10 w-10 text-slate-700 mb-4" />
              <p className="text-sm text-slate-500">Select a file to view changes</p>
              <p className="text-xs text-slate-600 mt-1">
                Click on a file from the list on the left
              </p>
            </div>
          )}

          {/* No diff content */}
          {selectedFile && !isLoading && !error && diff && !diff.has_diff && (
            <div className="flex flex-col items-center justify-center py-12 text-center" data-testid="diff-no-changes">
              <FileDiff className="h-10 w-10 text-slate-700 mb-4" />
              <p className="text-sm text-slate-500">No changes detected</p>
              <p className="text-xs text-slate-600 mt-1">
                This file has no differences from HEAD
              </p>
            </div>
          )}

          {/* Diff content */}
          {selectedFile && !isLoading && !error && diff?.has_diff && diff.hunks && (
            <div data-testid="diff-content">
              {diff.hunks.map((hunk, index) => (
                <HunkDisplay key={index} hunk={hunk} index={index} />
              ))}
            </div>
          )}

          {/* Binary diff notice */}
          {showBinaryNotice && (
            <div className="flex flex-col items-center justify-center py-16 text-center px-6">
              <AlertTriangle className="h-10 w-10 text-amber-400 mb-4" />
              <p className="text-sm text-slate-300">
                The file is not displayed in the text editor because it is either binary or uses an unsupported text encoding.
              </p>
              <Button
                variant="outline"
                size="sm"
                className="mt-4"
                onClick={() => setShowBinary(true)}
              >
                Show Anyway
              </Button>
            </div>
          )}

          {/* Raw diff fallback */}
          {selectedFile &&
            !isLoading &&
            !error &&
            diff?.has_diff &&
            !diff.hunks &&
            diff.raw &&
            (!isBinaryDiff || showBinary) && (
            <pre
              className="p-4 font-mono text-xs text-slate-300 whitespace-pre overflow-x-auto"
              data-testid="diff-raw"
            >
              {diff.raw}
            </pre>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
