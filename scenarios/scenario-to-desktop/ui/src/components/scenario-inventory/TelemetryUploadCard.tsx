import { useMemo, useState, useCallback } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import {
  UploadCloud,
  Loader2,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Copy,
  Check
} from "lucide-react";
import {
  uploadTelemetry,
  fetchTelemetrySummary,
  fetchTelemetryTail,
  getTelemetryDownloadUrl
} from "../../lib/api";
import { readFileAsText, writeToClipboard } from "../../lib/browser";
import {
  generateTelemetryPaths,
  processTelemetryContent,
  generateExampleEvent,
  formatEventPreview
} from "../../domain/telemetry";
import { formatBytes } from "../../domain/download";
import type { TelemetryTailEntry, TelemetryTailResponse, TelemetrySummary } from "../../domain/types";

interface TelemetryUploadCardProps {
  scenarioName: string;
  appDisplayName?: string;
}

export function TelemetryUploadCard({ scenarioName, appDisplayName }: TelemetryUploadCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successPath, setSuccessPath] = useState<string | null>(null);
  const [showPaths, setShowPaths] = useState(false);
  const [copiedPath, setCopiedPath] = useState<string | null>(null);
  const [showTail, setShowTail] = useState(false);
  const [tailLimit] = useState(200);

  const telemetryPaths = useMemo(() => {
    const appName = appDisplayName || scenarioName;
    return generateTelemetryPaths(appName);
  }, [appDisplayName, scenarioName]);

  const {
    data: telemetrySummary,
    isFetching: summaryLoading,
    error: summaryError,
    refetch: refetchSummary
  } = useQuery<TelemetrySummary>({
    queryKey: ["telemetry-summary", scenarioName],
    queryFn: () => fetchTelemetrySummary(scenarioName),
    enabled: isExpanded,
    refetchInterval: isExpanded ? 15000 : false,
    refetchIntervalInBackground: true
  });

  const {
    data: telemetryTail,
    isFetching: tailLoading,
    error: tailError,
    refetch: refetchTail
  } = useQuery<TelemetryTailResponse>({
    queryKey: ["telemetry-tail", scenarioName, tailLimit],
    queryFn: () => fetchTelemetryTail(scenarioName, tailLimit),
    enabled: isExpanded && showTail && !!telemetrySummary?.exists
  });

  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!selectedFile) {
        throw new Error("Choose the telemetry file first");
      }

      const fileResult = await readFileAsText(selectedFile);
      if (!fileResult.success || !fileResult.content) {
        throw new Error(fileResult.error || "Failed to read file");
      }

      const result = processTelemetryContent(fileResult.content);
      if (!result.success || !result.events) {
        throw new Error(result.error || "Failed to process telemetry file");
      }

      return uploadTelemetry({
        scenario_name: scenarioName,
        events: result.events
      });
    },
    onSuccess: (data) => {
      setError(null);
      setSuccessPath(data.output_path);
      void refetchSummary();
    },
    onError: (err: Error) => {
      setSuccessPath(null);
      setError(err.message);
    }
  });

  const handleFileChange = useCallback((fileList: FileList | null) => {
    const file = fileList?.item(0) ?? null;
    setSelectedFile(file);
    setError(null);
    setSuccessPath(null);
  }, []);

  const handleCopyPath = useCallback(async (path: string) => {
    const result = await writeToClipboard(path);
    if (result.success) {
      setCopiedPath(path);
      setTimeout(() => setCopiedPath(null), 2000);
    }
  }, []);

  // Collapsed state - single line trigger
  if (!isExpanded) {
    return (
      <button
        onClick={() => setIsExpanded(true)}
        className="flex items-center gap-2 w-full px-3 py-2 rounded-lg border border-slate-700/60 bg-slate-900/20 text-sm text-slate-400 hover:text-slate-300 hover:border-slate-600 transition-colors text-left"
      >
        <UploadCloud className="h-4 w-4 flex-shrink-0" />
        <span className="flex-1">After testing, help us improve by uploading telemetry</span>
        <ChevronDown className="h-4 w-4 flex-shrink-0" />
      </button>
    );
  }

  // Expanded state - full upload interface
  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-3">
      {/* Header with collapse button */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
          <UploadCloud className="h-4 w-4" />
          Upload telemetry
        </div>
        <Button
          variant="outline"
          size="sm"
          className="h-7 px-2"
          onClick={() => setIsExpanded(false)}
        >
          <ChevronUp className="h-4 w-4" />
        </Button>
      </div>

      <p className="text-xs text-slate-400">
        The desktop app logs startup events and dependency failures to <code className="text-slate-300">deployment-telemetry.jsonl</code>.
        Upload this file so we can improve future builds.
      </p>

      {successPath && (
        <div className="flex items-center gap-2 rounded-lg border border-green-800/40 bg-green-950/20 px-3 py-2 text-xs text-green-300">
          <CheckCircle2 className="h-4 w-4 flex-shrink-0" />
          <span>Telemetry uploaded successfully</span>
        </div>
      )}

      <div className="rounded border border-slate-800 bg-black/20 p-3 space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="text-xs font-semibold text-slate-200">Uploaded telemetry</div>
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="ghost" size="sm" className="h-6 px-2" onClick={() => refetchSummary()}>
              Refresh
            </Button>
            {telemetrySummary?.exists && (
              <a
                className="text-xs text-blue-300 underline"
                href={getTelemetryDownloadUrl(scenarioName)}
                target="_blank"
                rel="noreferrer"
              >
                Download JSONL
              </a>
            )}
          </div>
        </div>

        {summaryLoading && (
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <Loader2 className="h-3 w-3 animate-spin" />
            Loading telemetry summary...
          </div>
        )}

        {summaryError && (
          <p className="text-xs text-red-400">
            {summaryError instanceof Error ? summaryError.message : "Failed to load telemetry summary"}
          </p>
        )}

        {!summaryLoading && telemetrySummary && !telemetrySummary.exists && (
          <p className="text-xs text-slate-400">No uploaded telemetry yet.</p>
        )}

        {telemetrySummary?.exists && (
          <div className="grid gap-2 text-xs text-slate-300">
            <div className="flex items-center justify-between">
              <span>Events</span>
              <span className="text-slate-100">{telemetrySummary.event_count ?? 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>File size</span>
              <span className="text-slate-100">{formatBytes(telemetrySummary.file_size_bytes)}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Last upload</span>
              <span className="text-slate-100">
                {telemetrySummary.last_ingested_at
                  ? new Date(telemetrySummary.last_ingested_at).toLocaleString()
                  : "Unknown"}
              </span>
            </div>
            {telemetrySummary.file_path && (
              <div className="text-[11px] text-slate-500">
                Server file: <span className="font-mono text-slate-300">{telemetrySummary.file_path}</span>
              </div>
            )}
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="h-7 px-2"
                onClick={() => {
                  setShowTail((prev) => !prev);
                }}
              >
                {showTail ? "Hide telemetry" : `View last ${tailLimit} events`}
              </Button>
              {showTail && (
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  className="h-7 px-2"
                  onClick={() => refetchTail()}
                >
                  Refresh events
                </Button>
              )}
            </div>
          </div>
        )}

        {showTail && telemetrySummary?.exists && (
          <div className="mt-2 max-h-72 space-y-2 overflow-auto rounded border border-slate-800 bg-slate-950/40 p-2">
            <div className="flex items-center justify-between text-[11px] text-slate-400">
              <span>
                Showing last {tailLimit} events
                {telemetryTail?.total_lines ? ` (of ${telemetryTail.total_lines})` : ""}
              </span>
              {tailLoading && <Loader2 className="h-3 w-3 animate-spin" />}
            </div>
            {tailError && (
              <p className="text-xs text-red-400">
                {tailError instanceof Error ? tailError.message : "Failed to load telemetry entries"}
              </p>
            )}
            {!tailLoading && telemetryTail?.entries && telemetryTail.entries.length === 0 && (
              <p className="text-xs text-slate-400">No telemetry entries found.</p>
            )}
            {telemetryTail?.entries && telemetryTail.entries.length > 0 && (
              <ul className="space-y-2">
                {telemetryTail.entries.map((entry, index) => (
                  <li
                    key={`${index}-${entry.raw.slice(0, 16)}`}
                    className="rounded border border-slate-800 bg-black/30 p-2 text-xs text-slate-300"
                  >
                    <div>{formatTailEntry(entry)}</div>
                    {entry.error && (
                      <div className="mt-1 text-[11px] text-amber-300">Parse error: {entry.error}</div>
                    )}
                    <details className="mt-1">
                      <summary className="cursor-pointer text-[11px] text-slate-400">Raw</summary>
                      <pre className="mt-1 overflow-x-auto rounded bg-black/40 p-2 text-[10px] text-slate-400">
                        {entry.raw}
                      </pre>
                    </details>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>

      {/* Path guide - collapsible */}
      <div className="rounded border border-slate-800 bg-black/20 p-2">
        <button
          onClick={() => setShowPaths((prev) => !prev)}
          className="flex items-center justify-between w-full text-xs text-slate-400 hover:text-slate-300"
        >
          <span>Where is the file?</span>
          {showPaths ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
        </button>
        {showPaths && (
          <ul className="mt-2 space-y-1.5">
            {telemetryPaths.map(({ os, path }) => (
              <li key={os} className="flex flex-wrap items-center gap-2 text-xs">
                <span className="font-medium text-slate-300 w-16">{os}:</span>
                <code className="flex-1 rounded bg-slate-900/60 px-1.5 py-0.5 text-[11px] text-slate-400 truncate">
                  {path}
                </code>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-5 px-1.5 gap-1"
                  onClick={() => handleCopyPath(path)}
                >
                  {copiedPath === path ? (
                    <Check className="h-3 w-3 text-green-400" />
                  ) : (
                    <Copy className="h-3 w-3" />
                  )}
                </Button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* File picker and upload */}
      <div className="flex flex-wrap items-center gap-2">
        <Input
          type="file"
          accept=".jsonl,.json,.txt"
          className="flex-1 min-w-[200px]"
          onChange={(event) => handleFileChange(event.target.files)}
        />
        <Button
          size="sm"
          className="gap-1.5"
          disabled={!selectedFile || uploadMutation.isPending}
          onClick={() => uploadMutation.mutate()}
        >
          {uploadMutation.isPending ? (
            <>
              <Loader2 className="h-3 w-3 animate-spin" />
              Uploading...
            </>
          ) : (
            <>
              <UploadCloud className="h-3 w-3" />
              Upload
            </>
          )}
        </Button>
      </div>

      {selectedFile && (
        <p className="text-xs text-slate-500">Selected: {selectedFile.name}</p>
      )}

      {error && <p className="text-xs text-red-400">{error}</p>}

      {/* Example format - very compact */}
      <details className="text-xs">
        <summary className="text-slate-500 cursor-pointer hover:text-slate-400">
          Expected format
        </summary>
        <pre className="mt-1 overflow-x-auto rounded bg-black/40 p-2 text-[10px] text-slate-400">
          {generateExampleEvent()}
        </pre>
      </details>
    </div>
  );
}

function formatTailEntry(entry: TelemetryTailEntry): string {
  if (entry.event) {
    return formatEventPreview(entry.event);
  }
  return "Unparsed telemetry line";
}
