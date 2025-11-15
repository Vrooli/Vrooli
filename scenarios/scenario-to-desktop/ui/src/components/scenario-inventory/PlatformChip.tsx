import { useState, useEffect } from "react";
import type { PlatformBuildResult } from "../../lib/api";
import { Button } from "../ui/button";
import { CheckCircle, XCircle, Loader2, AlertCircle, ChevronDown, ChevronUp, Copy, Check, FileDown } from "lucide-react";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { formatBytes, platformIcons, platformNames } from "./utils";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface PlatformChipProps {
  platform: string;
  result?: PlatformBuildResult;
  scenarioName: string;
}

export function PlatformChip({ platform, result, scenarioName }: PlatformChipProps) {
  // Auto-expand errors for failed builds, persist in sessionStorage
  const storageKey = `error-expanded-${scenarioName}-${platform}`;
  const [showError, setShowError] = useState(() => {
    const stored = sessionStorage.getItem(storageKey);
    return stored !== null ? stored === 'true' : (result?.status === 'failed');
  });
  const [copied, setCopied] = useState(false);

  // Update sessionStorage when showError changes
  useEffect(() => {
    sessionStorage.setItem(storageKey, String(showError));
  }, [showError, storageKey]);

  // Auto-expand when status changes to failed
  useEffect(() => {
    if (result?.status === 'failed') {
      setShowError(true);
    }
  }, [result?.status]);

  const handleDownload = () => {
    const downloadUrl = buildUrl(`/desktop/download/${scenarioName}/${platform}`);
    window.open(downloadUrl, '_blank');
  };

  const handleCopyErrors = async () => {
    if (!result?.error_log) return;
    const errorText = result.error_log.join('\n\n---\n\n');
    await navigator.clipboard.writeText(errorText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Determine chip style based on status
  let chipClass = "flex items-center gap-2 px-3 py-2 rounded-lg border transition-all";
  let icon = null;
  let statusText = "";

  if (!result || result.status === "pending") {
    chipClass += " bg-slate-800 border-slate-600 text-slate-400";
    icon = <div className="h-2 w-2 rounded-full bg-slate-500" />;
    statusText = "Pending";
  } else if (result.status === "building") {
    chipClass += " bg-blue-950/30 border-blue-700 text-blue-300 animate-pulse";
    icon = <Loader2 className="h-3 w-3 animate-spin" />;
    statusText = "Building";
  } else if (result.status === "ready") {
    chipClass += " bg-green-950/30 border-green-700 text-green-300 hover:border-green-600 cursor-pointer";
    icon = <CheckCircle className="h-3 w-3" />;
    statusText = "Ready";
  } else if (result.status === "failed") {
    chipClass += " bg-red-950/30 border-red-700 text-red-300";
    icon = <XCircle className="h-3 w-3" />;
    statusText = "Failed";
  } else if (result.status === "skipped") {
    chipClass += " bg-yellow-950/30 border-yellow-700 text-yellow-300";
    icon = <AlertCircle className="h-3 w-3" />;
    statusText = "Skipped";
  }

  return (
    <div className="flex flex-col gap-2">
      <div
        className={chipClass}
        onClick={result?.status === "ready" ? handleDownload : undefined}
        title={result?.status === "ready" ? "Click to download" : undefined}
      >
        <span className="text-lg">{platformIcons[platform]}</span>
        {icon}
        <div className="flex flex-col gap-0.5">
          <span className="text-xs font-medium">{platformNames[platform]}</span>
          <span className="text-[10px] opacity-80">{statusText}</span>
        </div>
        {result?.file_size && result.status === "ready" && (
          <span className="text-[10px] ml-auto opacity-70">{formatBytes(result.file_size)}</span>
        )}
        {result?.status === "ready" && (
          <FileDown className="h-3 w-3 ml-auto" />
        )}
      </div>

      {/* Show skip reason for skipped platforms */}
      {result?.status === "skipped" && result.skip_reason && (
        <div className="bg-yellow-950/20 border border-yellow-800/30 rounded p-2 text-xs text-yellow-300">
          <div className="flex items-start gap-2">
            <AlertCircle className="h-3 w-3 flex-shrink-0 mt-0.5" />
            <div className="flex-1">{result.skip_reason}</div>
          </div>
        </div>
      )}

      {/* Show error details for failed platforms */}
      {result?.status === "failed" && result.error_log && result.error_log.length > 0 && (
        <div className="flex flex-col gap-1">
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-xs"
              onClick={() => setShowError(!showError)}
            >
              {showError ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
              {showError ? 'Hide' : 'Show'} Error
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-xs"
              onClick={handleCopyErrors}
            >
              {copied ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
              {copied ? 'Copied' : 'Copy'}
            </Button>
          </div>
          {showError && (
            <div className="bg-red-950/20 border border-red-800/30 rounded p-2 text-[10px] font-mono text-red-300 max-h-32 overflow-y-auto">
              {result.error_log.map((error, idx) => (
                <div key={idx} className="whitespace-pre-wrap break-words mb-1 opacity-90">
                  {error}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
