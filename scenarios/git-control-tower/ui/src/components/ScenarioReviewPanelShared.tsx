import { useState, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { CheckCircle2, AlertTriangle, ChevronRight, ChevronLeft, X, AlertCircle, Copy, Check } from "lucide-react";
import type { TidinessLightScanResult, TidinessStalenessInfo } from "../lib/api";
import { useGlobalKeydown } from "../hooks/useGlobalKeydown";

// Shared utilities extracted from ScenarioReviewPanel

export interface LightboxItem {
  label: string;
  sublabel?: string;
  type: "image" | "video";
  url: string;
}

export function MediaLightbox({
  items,
  initialIndex,
  isOpen,
  onClose,
}: {
  items: LightboxItem[];
  initialIndex: number;
  isOpen: boolean;
  onClose: () => void;
}) {
  const [index, setIndex] = useState(initialIndex);

  // Reset index when opening with a new initialIndex
  useEffect(() => {
    if (isOpen) setIndex(initialIndex);
  }, [isOpen, initialIndex]);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "ArrowLeft") setIndex(i => Math.max(0, i - 1));
    if (e.key === "ArrowRight") setIndex(i => Math.min(items.length - 1, i + 1));
  }, [onClose, items.length]);

  useGlobalKeydown(handleKeyDown, { disabled: !isOpen, target: document });

  if (!isOpen || items.length === 0) return null;

  const clampedIndex = Math.max(0, Math.min(index, items.length - 1));
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- clampedIndex is always in bounds after the length check above
  const current = items[clampedIndex]!;
  const hasPrev = clampedIndex > 0;
  const hasNext = clampedIndex < items.length - 1;

  return createPortal(
    <div
      className="fixed inset-0 z-[60] flex flex-col bg-black/95"
      onClick={onClose}
    >
      {/* Top info bar */}
      <div
        className="flex items-center justify-between px-4 py-3 bg-black/80 border-b border-slate-800/50"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-200 truncate">{current.label}</p>
          {current.sublabel && (
            <p className="text-[11px] text-slate-500 truncate">{current.sublabel}</p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0 ml-4">
          {items.length > 1 && (
            <span className="text-xs text-slate-500">{clampedIndex + 1} / {items.length}</span>
          )}
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Media area */}
      <div
        className="flex-1 flex items-center justify-center p-4 min-h-0"
        onClick={(e) => e.stopPropagation()}
      >
        {current.type === "image" ? (
          <img
            key={current.url}
            src={current.url}
            alt={current.label}
            className="max-w-full max-h-full object-contain rounded-lg"
          />
        ) : (
          <video
            key={current.url}
            controls
            autoPlay
            src={current.url}
            className="max-w-full max-h-full rounded-lg"
          />
        )}
      </div>

      {/* Bottom nav bar */}
      {items.length > 1 && (
        <div
          className="flex items-center justify-center gap-4 px-4 py-3 bg-black/80 border-t border-slate-800/50"
          onClick={(e) => e.stopPropagation()}
        >
          <button
            type="button"
            disabled={!hasPrev}
            onClick={() => setIndex(i => i - 1)}
            className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800 disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Previous"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
          <div className="flex gap-1.5">
            {items.map((_, i) => (
              <button
                key={i}
                type="button"
                onClick={() => setIndex(i)}
                className={`h-2 rounded-full transition-all ${
                  i === clampedIndex ? "w-6 bg-blue-400" : "w-2 bg-slate-600 hover:bg-slate-500"
                }`}
                aria-label={`Go to item ${i + 1}`}
              />
            ))}
          </div>
          <button
            type="button"
            disabled={!hasNext}
            onClick={() => setIndex(i => i + 1)}
            className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800 disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Next"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
        </div>
      )}
    </div>,
    document.body,
  );
}

export function MutationErrorBanner({ error, onDismiss }: { error: Error | null; onDismiss?: () => void }) {
  const [copied, setCopied] = useState(false);
  if (!error) return null;
  const handleCopy = () => {
    void navigator.clipboard.writeText(error.message).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };
  return (
    <div className="flex items-start gap-2 p-3 rounded-lg bg-red-950/30 border border-red-900/40">
      <AlertTriangle className="h-3.5 w-3.5 text-red-400 mt-0.5 shrink-0" />
      <p className="flex-1 text-xs text-red-300 max-h-32 overflow-y-auto break-words">{error.message}</p>
      <button type="button" onClick={handleCopy} className="text-red-400 hover:text-red-300 shrink-0" aria-label="Copy error" title="Copy error">
        {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      </button>
      {onDismiss && (
        <button type="button" onClick={onDismiss} className="text-red-400 hover:text-red-300 shrink-0" aria-label="Dismiss">
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}

export function ServiceUnavailableBanner({ name, message }: { name: string; message?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-slate-500">
      <AlertCircle className="h-8 w-8 mb-3 opacity-50" />
      <p className="text-sm">{name} is not available</p>
      {message && (
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">{message}</p>
      )}
    </div>
  );
}

export function sanitizePagePath(pagePath: string): string {
  if (pagePath === "/" || pagePath === "") return "_root_";
  let s = pagePath.startsWith("/") ? pagePath.slice(1) : pagePath;
  s = s.endsWith("/") ? s.slice(0, -1) : s;
  s = s.replace(/\//g, "_");
  return "_" + s + "_";
}

export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`;
}

export function formatRelativeTime(isoString: string): string {
  const date = new Date(isoString);
  const now = Date.now();
  const diffMs = now - date.getTime();
  if (diffMs < 0) return "just now";
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return "just now";
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

export function formatStalenessMessage(staleness: TidinessStalenessInfo): string {
  const parts: string[] = [];
  if (staleness.stale_reason) {
    parts.push(staleness.stale_reason);
  } else {
    parts.push("Quality data may be stale");
  }
  if (staleness.last_scan_at) {
    parts.push(`Last scan: ${formatRelativeTime(staleness.last_scan_at)}`);
  }
  if (staleness.modified_files && staleness.modified_files > 0 && !staleness.stale_reason?.includes("file")) {
    parts.push(`${staleness.modified_files} file${staleness.modified_files !== 1 ? "s" : ""} changed`);
  }
  return parts.join(" · ");
}

export function ScanResultSummary({ result }: { result: TidinessLightScanResult }) {
  const durationSec = (result.duration_ms / 1000).toFixed(1);
  const totalIssues = result.lint_issues + result.type_issues;
  return (
    <div className="flex items-center gap-2 px-3 py-2 bg-slate-800/50 border border-slate-700/50 rounded-lg text-xs text-slate-300 mt-2">
      <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
      <span>
        Scanned {result.total_files} files ({result.total_lines.toLocaleString()} lines) in {durationSec}s
        {" — "}
        {totalIssues === 0 ? (
          <span className="text-emerald-400">no issues found</span>
        ) : (
          <span className="text-amber-400">
            {result.lint_issues} lint, {result.type_issues} type, {result.long_files_count} long file{result.long_files_count !== 1 ? "s" : ""}
          </span>
        )}
      </span>
    </div>
  );
}
