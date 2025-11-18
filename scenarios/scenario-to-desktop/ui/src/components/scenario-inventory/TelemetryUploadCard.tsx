import { useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { UploadCloud, Loader2, CheckCircle2, Info, ChevronDown, ChevronUp, Copy, Check } from "lucide-react";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

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

  const telemetryPaths = useMemo(() => {
    const label = appDisplayName || scenarioName;
    return [
      { os: 'Windows', path: `%APPDATA%/${label}/deployment-telemetry.jsonl` },
      { os: 'macOS', path: `~/Library/Application Support/${label}/deployment-telemetry.jsonl` },
      { os: 'Linux', path: `~/.config/${label}/deployment-telemetry.jsonl` }
    ];
  }, [appDisplayName, scenarioName]);

  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!selectedFile) {
        throw new Error("Choose the telemetry file first");
      }

      const raw = await selectedFile.text();
      const lines = raw
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter(Boolean);
      if (lines.length === 0) {
        throw new Error("File is empty");
      }

      const events = lines.map((line, index) => {
        try {
          return JSON.parse(line);
        } catch {
          throw new Error(`Line ${index + 1} isn't valid JSON`);
        }
      });

      const res = await fetch(buildUrl('/deployment/telemetry'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scenario_name: scenarioName,
          deployment_mode: 'external-server',
          source: 'desktop-upload',
          events,
        }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to upload telemetry');
      }

      return res.json();
    },
    onSuccess: (data) => {
      setError(null);
      setSuccessPath(data.output_path);
    },
    onError: (err: Error) => {
      setSuccessPath(null);
      setError(err.message);
    },
  });

  const handleFileChange = (fileList: FileList | null) => {
    const file = fileList?.item(0) ?? null;
    setSelectedFile(file);
    setError(null);
    setSuccessPath(null);
  };

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-3 text-slate-200">
      <div className="flex items-center gap-2 text-sm font-semibold text-slate-100">
        <UploadCloud className="h-4 w-4" />
        Share telemetry (optional but extremely helpful)
      </div>
      <p className="text-xs text-slate-400">
        Every desktop wrapper records start-up events, dependency failures, and shutdowns to <code>deployment-telemetry.jsonl</code>. Uploading the file lets
        deployment-manager see where bundling still fails—no secrets are stored inside the log.
      </p>
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
                  onClick={() => {
                    navigator.clipboard.writeText(path);
                    setCopiedPath(path);
                    setTimeout(() => setCopiedPath(null), 2000);
                  }}
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
      <p className="text-[11px] text-slate-500">
        We only store these logs locally and forward them to deployment-manager. Read more about telemetry expectations in the{' '}
        <a
          href="https://github.com/vrooli/vrooli/blob/main/docs/deployment/examples/picker-wheel-desktop.md"
          target="_blank"
          rel="noreferrer"
          className="text-blue-300 underline"
        >
          Deployment Hub examples
        </a>
        .
      </p>
    </div>
  );
}
