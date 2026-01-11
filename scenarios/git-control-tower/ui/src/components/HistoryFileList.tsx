import { useMemo, useRef, useState, useEffect } from "react";
import { File, History, ChevronDown, ChevronRight } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import type { ViewingCommit } from "../App";

interface HistoryFileListProps {
  viewingCommit: ViewingCommit;
  selectedFile?: string;
  onSelectFile: (path: string) => void;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
  fillHeight?: boolean;
}

function formatPath(path: string, maxChars: number) {
  if (path.length <= maxChars) return path;

  const segments = path.split("/");
  const last = segments[segments.length - 1] || path;
  const lastTwo = segments.length > 1 ? segments.slice(-2).join("/") : last;
  const first = segments[0] || path;

  const candidateMiddle = `${first}/.../${lastTwo}`;
  if (candidateMiddle.length <= maxChars) return candidateMiddle;

  const candidateMiddleShort = `${first}/.../${last}`;
  if (candidateMiddleShort.length <= maxChars) return candidateMiddleShort;

  const candidateEnd = `.../${lastTwo}`;
  if (candidateEnd.length <= maxChars) return candidateEnd;

  const candidateEndShort = `.../${last}`;
  if (candidateEndShort.length <= maxChars) return candidateEndShort;

  const tailMax = Math.max(1, maxChars - 4);
  return `.../${last.slice(-tailMax)}`;
}

export function HistoryFileList({
  viewingCommit,
  selectedFile,
  onSelectFile,
  collapsed = false,
  onToggleCollapse,
  fillHeight = true
}: HistoryFileListProps) {
  const handleToggleCollapse = onToggleCollapse ?? (() => {});
  const scrollAreaRef = useRef<HTMLDivElement | null>(null);
  const [maxPathChars, setMaxPathChars] = useState(72);

  useEffect(() => {
    if (!scrollAreaRef.current || typeof ResizeObserver === "undefined") return;

    const update = () => {
      const width = scrollAreaRef.current?.clientWidth ?? 0;
      // Account for: file icon (~22px), padding (~16px), some buffer (~42px) = ~80px
      const usable = Math.max(0, width - 80);
      const nextMax = Math.max(12, Math.min(100, Math.floor(usable / 7.5)));
      setMaxPathChars(nextMax);
    };

    // Defer initial measurement to ensure layout is complete after expanding
    const rafId = requestAnimationFrame(update);
    const observer = new ResizeObserver(update);
    observer.observe(scrollAreaRef.current);
    return () => {
      cancelAnimationFrame(rafId);
      observer.disconnect();
    };
    // Re-run when collapsed changes to re-observe the new ScrollArea element
  }, [collapsed]);

  const sortedFiles = useMemo(() => {
    return [...viewingCommit.files].sort((a, b) => a.localeCompare(b));
  }, [viewingCommit.files]);

  return (
    <Card
      className={`flex flex-col min-w-0 ${fillHeight ? "h-full" : "h-auto"}`}
      data-testid="history-file-list-panel"
    >
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3 gap-2 min-w-0">
        <CardTitle className="flex items-center gap-2 min-w-0">
          <button
            className="p-1 rounded hover:bg-slate-800/70 transition-colors"
            onClick={handleToggleCollapse}
            aria-label={collapsed ? "Expand files" : "Collapse files"}
            type="button"
          >
            {collapsed ? (
              <ChevronRight className="h-3 w-3 text-slate-400" />
            ) : (
              <ChevronDown className="h-3 w-3 text-slate-400" />
            )}
          </button>
          <History className="h-4 w-4 text-amber-400" />
          <span className="truncate text-amber-200">Commit Files</span>
        </CardTitle>
        <span className="text-xs text-slate-500">{sortedFiles.length} files</span>
      </CardHeader>

      {!collapsed && (
        <CardContent className="flex-1 min-w-0 p-0 overflow-hidden">
          {/* Read-only notice */}
          <div className="mx-2 mb-2 rounded-md border border-amber-800/50 bg-amber-950/20 p-2 text-xs text-amber-200/80">
            <div className="flex items-center gap-2">
              <History className="h-3.5 w-3.5 text-amber-400 flex-shrink-0" />
              <span>Viewing historical commit - read only</span>
            </div>
          </div>

          <ScrollArea className="h-full min-w-0 px-2 pt-2 select-none" ref={scrollAreaRef}>
            <div style={{ paddingBottom: 48 }}>
            {sortedFiles.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-center" data-testid="empty-state">
                <File className="h-8 w-8 text-slate-700 mb-3" />
                <p className="text-sm text-slate-500">No files in this commit</p>
              </div>
            ) : (
              <ul className="space-y-0.5 min-w-0">
                {sortedFiles.map((file) => {
                  const isSelected = selectedFile === file;
                  const displayPath = formatPath(file, maxPathChars);

                  return (
                    <li
                      key={file}
                      className={`group w-full flex items-center gap-2 rounded cursor-pointer transition-colors min-w-0 overflow-hidden select-none px-2 py-1.5 ${
                        isSelected
                          ? "bg-amber-900/30 text-amber-100 border border-amber-700/50"
                          : "hover:bg-slate-800/50 active:bg-slate-700/50 text-slate-300"
                      }`}
                      data-testid="history-file-item"
                      data-file-path={file}
                      onClick={() => onSelectFile(file)}
                    >
                      <File className="h-3.5 w-3.5 text-slate-500 flex-shrink-0" />
                      <div className="flex-1 min-w-0 overflow-hidden">
                        <span className="font-mono text-xs truncate block w-full" title={file}>
                          {displayPath}
                        </span>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
            </div>
          </ScrollArea>
        </CardContent>
      )}
    </Card>
  );
}
