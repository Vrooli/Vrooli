import { useMemo, useState, useCallback } from "react";
import { useMutation } from "@tanstack/react-query";
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
import { uploadTelemetry } from "../../lib/api";
import { readFileAsText, writeToClipboard } from "../../lib/browser";
import {
  generateTelemetryPaths,
  processTelemetryContent,
  generateExampleEvent
} from "../../domain/telemetry";

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

  const telemetryPaths = useMemo(() => {
    const appName = appDisplayName || scenarioName;
    return generateTelemetryPaths(appName);
  }, [appDisplayName, scenarioName]);

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

  // Success state - show minimal confirmation
  if (successPath) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 rounded-lg border border-green-800/40 bg-green-950/20 text-sm text-green-300">
        <CheckCircle2 className="h-4 w-4 flex-shrink-0" />
        <span>Telemetry uploaded successfully</span>
      </div>
    );
  }

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
