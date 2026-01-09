import { useMemo, useState, useCallback } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import {
  UploadCloud,
  Loader2,
  CheckCircle2,
  Info,
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
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successPath, setSuccessPath] = useState<string | null>(null);
  const [showHelp, setShowHelp] = useState(false);
  const [copiedPath, setCopiedPath] = useState<string | null>(null);

  // Use domain function for path generation
  const telemetryPaths = useMemo(() => {
    const appName = appDisplayName || scenarioName;
    return generateTelemetryPaths(appName);
  }, [appDisplayName, scenarioName]);

  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!selectedFile) {
        throw new Error("Choose the telemetry file first");
      }

      // Read file content using browser seam
      const fileResult = await readFileAsText(selectedFile);
      if (!fileResult.success || !fileResult.content) {
        throw new Error(fileResult.error || "Failed to read file");
      }

      // Use domain function for parsing and validation
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

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-3 text-slate-200">
      <div className="flex items-center gap-2 text-sm font-semibold text-slate-100">
        <UploadCloud className="h-4 w-4" />
        Share telemetry (optional but extremely helpful)
      </div>
      <p className="text-xs text-slate-400">
        Every desktop wrapper records start-up events, dependency failures, and shutdowns to <code>deployment-telemetry.jsonl</code>. Uploading the file lets
        deployment-manager see exactly which dependencies or secrets failed so we know what to fix next—no credentials are stored inside the log.
      </p>
      <ol className="list-decimal space-y-1 rounded border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-300">
        <li>Use the path guide below to open the telemetry file on your machine.</li>
        <li>Drag the file into the picker or click to browse for it.</li>
        <li>Hit <strong>Upload telemetry</strong> so the events sync to deployment-manager.</li>
      </ol>
      <div className="rounded border border-slate-800 bg-black/30 p-3 text-xs text-slate-300">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Info className="h-3 w-3" />
            Where do I find the file?
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="gap-1"
            onClick={() => setShowHelp((prev) => !prev)}
          >
            {showHelp ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
            {showHelp ? 'Hide' : 'Show'} paths
          </Button>
        </div>
        {showHelp && (
          <ul className="mt-2 space-y-1">
            {telemetryPaths.map(({ os, path }) => (
              <li key={os} className="flex flex-wrap items-center gap-2">
                <span className="font-semibold text-slate-100">{os}:</span>
                <code className="rounded bg-slate-900/60 px-2 py-0.5 text-[11px]">{path}</code>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 gap-1"
                  onClick={() => handleCopyPath(path)}
                >
                  {copiedPath === path ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
                  Copy
                </Button>
              </li>
            ))}
          </ul>
        )}
      </div>
      <Input
        type="file"
        accept=".jsonl,.json,.txt"
        onChange={(event) => handleFileChange(event.target.files)}
      />
      <div className="rounded border border-slate-800 bg-slate-950/40 p-3 text-[11px] text-slate-300">
        <p className="font-semibold text-slate-100">Need an example?</p>
        <p className="mt-1">Each line is JSON. A single entry might look like:</p>
        <pre className="mt-2 overflow-x-auto rounded bg-black/40 p-2 text-[10px] text-slate-100">
          {generateExampleEvent()}
        </pre>
      </div>
      <div className="flex items-center gap-3">
        <Button
          size="sm"
          className="gap-2"
          disabled={!selectedFile || uploadMutation.isPending}
          onClick={() => uploadMutation.mutate()}
        >
          {uploadMutation.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" /> Uploading...
            </>
          ) : (
            <>Upload telemetry</>
          )}
        </Button>
        {selectedFile && (
          <span className="text-xs text-slate-400 truncate max-w-[180px]">{selectedFile.name}</span>
        )}
      </div>
      {error && <p className="text-xs text-red-300">{error}</p>}
      {successPath && (
        <p className="flex items-center gap-1 text-xs text-green-300">
          <CheckCircle2 className="h-3 w-3" /> Uploaded! Saved to {successPath}
        </p>
      )}
      <div className="rounded border border-slate-800 bg-slate-950/40 p-3 text-[11px] text-slate-400">
        Why upload? Telemetry tells deployment-manager which swaps or secret strategies to suggest. Logs remain local unless you press Upload, and the
        UI automatically strips infrastructure credentials.{' '}
        <a
          href="https://github.com/vrooli/vrooli/blob/main/docs/deployment/examples/picker-wheel-desktop.md"
          target="_blank"
          rel="noreferrer"
          className="text-blue-300 underline"
        >
          Read the real-world example
        </a>
        .
      </div>
    </div>
  );
}
