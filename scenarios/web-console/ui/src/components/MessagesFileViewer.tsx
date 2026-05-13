import { useEffect, useMemo, useRef, useState } from "react";
import { AlertTriangle, Check, Copy, Loader2, X } from "lucide-react";
import { MarkdownRenderer } from "./markdown";
import { cn } from "../lib/classnames";
import type { FileReferenceContentResponse, FileReferenceResolveResponse } from "../api/conversation";

interface MessagesFileViewerProps {
  open: boolean;
  loading: boolean;
  error: string | null;
  requestedPath: string | null;
  resolved: FileReferenceResolveResponse | null;
  content: FileReferenceContentResponse | null;
  onClose: () => void;
}

export default function MessagesFileViewer({
  open,
  loading,
  error,
  requestedPath,
  resolved,
  content,
  onClose,
}: MessagesFileViewerProps) {
  const targetLine = content?.line ?? resolved?.line ?? null;
  const displayPath = content?.path ?? resolved?.resolved_path ?? requestedPath ?? "";
  const [copied, setCopied] = useState(false);
  const copyPath = () => {
    if (!displayPath) return;
    void navigator.clipboard.writeText(displayPath);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  const basename = useMemo(() => {
    const fullPath = content?.path ?? resolved?.resolved_path ?? requestedPath ?? "";
    if (!fullPath) return "File preview";
    const parts = fullPath.split(/[\\/]/);
    return parts[parts.length - 1] || fullPath;
  }, [content?.path, requestedPath, resolved?.resolved_path]);

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose, open]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[80]">
      <div className="absolute inset-0 bg-wc-backdrop" onClick={onClose} />
      <div className="absolute inset-x-0 bottom-0 top-4 overflow-hidden rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised shadow-2xl md:inset-x-8 md:bottom-8 md:top-8 md:rounded-2xl md:border">
        <div className="flex items-start justify-between gap-4 border-b border-wc-default px-4 py-3">
          <div className="min-w-0">
            <h2 className="truncate text-sm font-semibold text-wc-text-primary">{basename}</h2>
            <div className="flex items-center gap-1.5">
              <p className="truncate text-xs text-wc-text-muted">
                {displayPath || "Loading file..."}
              </p>
              {displayPath && (
                <button
                  type="button"
                  onClick={copyPath}
                  className="shrink-0 rounded p-1 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
                  aria-label={copied ? "Copied" : "Copy path"}
                  title={copied ? "Copied" : "Copy path"}
                >
                  {copied ? (
                    <Check className="h-3.5 w-3.5 text-green-400" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </button>
              )}
            </div>
            <div className="mt-1 flex flex-wrap gap-2 text-[11px] text-wc-text-faint">
              {resolved?.resolution_basis && (
                <span className="rounded-full border border-wc-default px-2 py-0.5">
                  {resolved.resolution_basis}
                </span>
              )}
              {content?.category && (
                <span className="rounded-full border border-wc-default px-2 py-0.5">
                  {content.category}
                </span>
              )}
              {targetLine && (
                <span className="rounded-full border border-wc-default px-2 py-0.5">
                  line {targetLine}
                </span>
              )}
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            aria-label="Close file preview"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="h-[calc(100%-76px)] overflow-auto px-4 py-4">
          {loading && (
            <div className="flex h-full items-center justify-center gap-2 text-wc-text-muted">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>Loading file preview…</span>
            </div>
          )}

          {!loading && error && (
            <div className="mx-auto max-w-2xl rounded-2xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-300">
              <div className="mb-2 flex items-center gap-2 font-medium">
                <AlertTriangle className="h-4 w-4" />
                <span>File preview unavailable</span>
              </div>
              <p>{error}</p>
              {requestedPath && (
                <p className="mt-2 break-all text-xs text-red-200/80">Requested: {requestedPath}</p>
              )}
            </div>
          )}

          {!loading && !error && content && (
            <>
              {content.category === "markdown"
                ? <MarkdownRenderer content={content.content} className="text-sm" />
                : <CodeLinePreview content={content.content} highlightLine={targetLine} />}
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function CodeLinePreview({ content, highlightLine }: { content: string; highlightLine: number | null }) {
  const lineRefs = useRef<Record<number, HTMLDivElement | null>>({});
  const lines = useMemo(() => content.split("\n"), [content]);

  useEffect(() => {
    if (!highlightLine) return;
    lineRefs.current[highlightLine]?.scrollIntoView({ block: "center", behavior: "smooth" });
  }, [highlightLine, lines.length]);

  return (
    <div className="overflow-hidden rounded-xl border border-wc-default bg-wc-surface">
      <div className="max-h-full overflow-auto font-mono text-sm">
        {lines.map((line, index) => {
          const lineNumber = index + 1;
          const isHighlighted = lineNumber === highlightLine;
          return (
            <div
              key={lineNumber}
              ref={(node) => {
                lineRefs.current[lineNumber] = node;
              }}
              className={cn(
                "grid grid-cols-[auto,1fr] gap-4 border-b border-wc-default/40 px-4 py-1.5 last:border-b-0",
                isHighlighted && "bg-wc-accent/10",
              )}
            >
              <span className={cn("select-none text-right text-xs text-wc-text-faint", isHighlighted && "text-wc-accent")}>
                {lineNumber}
              </span>
              <pre className="whitespace-pre-wrap break-words text-wc-text-primary">{line || " "}</pre>
            </div>
          );
        })}
      </div>
    </div>
  );
}
