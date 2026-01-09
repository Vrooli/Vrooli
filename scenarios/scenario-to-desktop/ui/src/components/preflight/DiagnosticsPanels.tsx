/**
 * Diagnostic panels for preflight log tails, fingerprints, and port info.
 */

import { useState } from "react";
import { Copy } from "lucide-react";
import type { BundlePreflightLogTail, BundlePreflightServiceFingerprint } from "../../lib/api";
import { Button } from "../ui/button";
import { countLines, formatBytes, formatTimestamp } from "../../lib/preflight-utils";

// ============================================================================
// Log Tails Panel
// ============================================================================

interface LogTailsPanelProps {
  logTails: BundlePreflightLogTail[];
}

export function LogTailsPanel({ logTails }: LogTailsPanelProps) {
  const [copyStatus, setCopyStatus] = useState<Record<string, "idle" | "copied" | "error">>({});

  if (!logTails || logTails.length === 0) {
    return null;
  }

  const handleCopy = async (tail: BundlePreflightLogTail, key: string) => {
    try {
      await navigator.clipboard.writeText(tail.content || "");
      setCopyStatus((prev) => ({ ...prev, [key]: "copied" }));
      window.setTimeout(() => {
        setCopyStatus((prev) => ({ ...prev, [key]: "idle" }));
      }, 1500);
    } catch (error) {
      console.warn("Failed to copy log tail", error);
      setCopyStatus((prev) => ({ ...prev, [key]: "error" }));
    }
  };

  return (
    <details className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200">
      <summary className="cursor-pointer text-xs font-semibold text-slate-100">
        Service log tails
      </summary>
      <div className="mt-2 space-y-2">
        {logTails.map((tail, idx) => {
          const copyKey = `${tail.service_id}-${idx}`;
          const lineCount = countLines(tail.content);
          const copyState = copyStatus[copyKey] || "idle";
          return (
            <div key={copyKey} className="rounded-md border border-slate-800/70 bg-slate-950/80 p-2 space-y-2">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="text-[11px] uppercase tracking-wide text-slate-400">
                  {tail.service_id} · {lineCount} of {tail.lines} lines
                </p>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  className="h-7 px-2 text-[10px]"
                  onClick={() => handleCopy(tail, copyKey)}
                >
                  <Copy className="h-3 w-3" />
                  <span className="ml-1">
                    {copyState === "copied" ? "Copied" : copyState === "error" ? "Failed" : "Copy"}
                  </span>
                </Button>
              </div>
              {tail.error ? (
                <p className="text-amber-200">{tail.error}</p>
              ) : (
                <pre className="max-h-40 overflow-auto whitespace-pre-wrap text-[11px] text-slate-200">
                  {tail.content || "No log output yet."}
                </pre>
              )}
            </div>
          );
        })}
      </div>
    </details>
  );
}

// ============================================================================
// Fingerprints Panel
// ============================================================================

interface FingerprintsPanelProps {
  fingerprints: BundlePreflightServiceFingerprint[];
}

export function FingerprintsPanel({ fingerprints }: FingerprintsPanelProps) {
  if (!fingerprints || fingerprints.length === 0) {
    return null;
  }

  return (
    <details className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200">
      <summary className="cursor-pointer text-xs font-semibold text-slate-100">
        Service fingerprints ({fingerprints.length})
      </summary>
      <div className="mt-2 space-y-2">
        {fingerprints.map((fp) => (
          <div key={`${fp.service_id}-${fp.binary_path || "unknown"}`} className="rounded-md border border-slate-800/70 bg-slate-950/80 p-2 space-y-1">
            <p className="text-[11px] uppercase tracking-wide text-slate-400">{fp.service_id}</p>
            {fp.error ? (
              <p className="text-[11px] text-amber-200">{fp.error}</p>
            ) : (
              <>
                {fp.binary_path && (
                  <p className="text-[11px] text-slate-300" title={fp.binary_resolved_path}>
                    {fp.binary_path}
                  </p>
                )}
                <div className="flex flex-wrap items-center gap-2 text-[10px] text-slate-400">
                  {fp.platform && <span>{fp.platform}</span>}
                  {fp.binary_size_bytes ? <span>{formatBytes(fp.binary_size_bytes)}</span> : null}
                  {fp.binary_mtime ? <span>{formatTimestamp(fp.binary_mtime)}</span> : null}
                </div>
                {fp.binary_sha256 && (
                  <p className="text-[10px] text-slate-500" title={fp.binary_sha256}>
                    {fp.binary_sha256.slice(0, 16)}
                  </p>
                )}
              </>
            )}
          </div>
        ))}
      </div>
    </details>
  );
}

// ============================================================================
// Port Summary Panel
// ============================================================================

interface PortSummaryPanelProps {
  portSummary?: string;
  telemetryPath?: string;
}

export function PortSummaryPanel({ portSummary, telemetryPath }: PortSummaryPanelProps) {
  if (!portSummary && !telemetryPath) {
    return null;
  }

  return (
    <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200 space-y-2">
      {portSummary && <p>Ports: {portSummary}</p>}
      {telemetryPath && <p>Telemetry: {telemetryPath}</p>}
    </div>
  );
}
