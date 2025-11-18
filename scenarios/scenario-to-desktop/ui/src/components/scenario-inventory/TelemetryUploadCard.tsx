import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { UploadCloud, Loader2, CheckCircle2 } from "lucide-react";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface TelemetryUploadCardProps {
  scenarioName: string;
}

export function TelemetryUploadCard({ scenarioName }: TelemetryUploadCardProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successPath, setSuccessPath] = useState<string | null>(null);

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
        Share telemetry (optional)
      </div>
      <p className="text-xs text-slate-400">
        Desktop wrappers write <code>deployment-telemetry.jsonl</code> under the user's app data folder. Drop it here so scenario-to-desktop can forward health
        data to deployment-manager.
      </p>
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
          <CheckCircle2 className="h-3 w-3" /> Saved to {successPath}
        </p>
      )}
    </div>
  );
}
