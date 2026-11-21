import { useState, useEffect, useCallback } from "react";
import {
  PlatformBuildStatus,
  type PlatformBuildResult,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/operation_results_pb";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { Button } from "../ui/button";
import {
  CheckCircle,
  XCircle,
  Loader2,
  AlertCircle,
  ChevronDown,
  ChevronUp,
  Copy,
  Check,
  FileDown,
} from "lucide-react";
import { getDownloadUrl } from "../../lib/api";
import { triggerDownload, writeToClipboard } from "../../lib/browser";
import {
  formatBytes,
  getPlatformIcon,
  getPlatformName,
} from "../../domain/download";

interface PlatformChipProps {
  platform: Platform;
  result?: PlatformBuildResult;
  scenarioName: string;
}

function platformKey(platform: Platform): "win" | "mac" | "linux" | "unknown" {
  switch (platform) {
    case Platform.WIN:
      return "win";
    case Platform.MAC:
      return "mac";
    case Platform.LINUX:
      return "linux";
    default:
      return "unknown";
  }
}

export function PlatformChip({
  platform,
  result,
  scenarioName,
}: PlatformChipProps) {
  const key = platformKey(platform);
  // Auto-expand errors for failed builds, persist in sessionStorage
  const storageKey = `error-expanded-${scenarioName}-${String(platform)}`;
  const [showError, setShowError] = useState(() => {
    const stored = sessionStorage.getItem(storageKey);
    return stored !== null
      ? stored === "true"
      : result?.status === PlatformBuildStatus.FAILED;
  });
  const [copied, setCopied] = useState(false);

  // Update sessionStorage when showError changes
  useEffect(() => {
    sessionStorage.setItem(storageKey, String(showError));
  }, [showError, storageKey]);

  // Auto-expand when status changes to failed
  useEffect(() => {
    if (result?.status === PlatformBuildStatus.FAILED) {
      setShowError(true);
    }
  }, [result?.status]);

  const handleDownload = useCallback(() => {
    const url = getDownloadUrl(scenarioName, key);
    triggerDownload({ url });
  }, [scenarioName, key]);

  const handleCopyErrors = useCallback(async () => {
    if (!result?.errorLog) return;
    const errorText = result.errorLog.join("\n\n---\n\n");
    const clipResult = await writeToClipboard(errorText);
    if (clipResult.success) {
      setCopied(true);
      setTimeout(() => {
        setCopied(false);
      }, 2000);
    }
  }, [result?.errorLog]);

  // Determine chip style based on status
  let chipClass =
    "flex items-center gap-2 px-3 py-2 rounded-lg border transition-all";
  let icon = null;
  let statusText = "";

  if (!result || result.status === PlatformBuildStatus.UNSPECIFIED) {
    chipClass += " bg-slate-800 border-slate-600 text-slate-400";
    icon = <div className="h-2 w-2 rounded-full bg-slate-500" />;
    statusText = "Pending";
  } else if (result.status === PlatformBuildStatus.BUILDING) {
    chipClass += " bg-blue-950/30 border-blue-700 text-blue-300 animate-pulse";
    icon = <Loader2 className="h-3 w-3 animate-spin" />;
    statusText = "Building";
  } else if (result.status === PlatformBuildStatus.READY) {
    chipClass +=
      " bg-green-950/30 border-green-700 text-green-300 hover:border-green-600 cursor-pointer";
    icon = <CheckCircle className="h-3 w-3" />;
    statusText = "Ready";
  } else if (result.status === PlatformBuildStatus.FAILED) {
    chipClass += " bg-red-950/30 border-red-700 text-red-300";
    icon = <XCircle className="h-3 w-3" />;
    statusText = "Failed";
  } else {
    chipClass += " bg-yellow-950/30 border-yellow-700 text-yellow-300";
    icon = <AlertCircle className="h-3 w-3" />;
    statusText = "Skipped";
  }

  return (
    <div className="flex flex-col gap-2">
      <button
        type="button"
        className={chipClass}
        onClick={
          result?.status === PlatformBuildStatus.READY
            ? handleDownload
            : undefined
        }
        disabled={result?.status !== PlatformBuildStatus.READY}
        aria-label={
          result?.status === PlatformBuildStatus.READY
            ? `Download ${getPlatformName(key)} build`
            : `${getPlatformName(key)} build ${statusText.toLowerCase()}`
        }
        title={
          result?.status === PlatformBuildStatus.READY
            ? "Click to download"
            : undefined
        }
      >
        <span className="text-lg">{getPlatformIcon(key)}</span>
        {icon}
        <div className="flex flex-col gap-0.5">
          <span className="text-xs font-medium">{getPlatformName(key)}</span>
          <span className="text-[10px] opacity-80">{statusText}</span>
        </div>
        {result?.fileSize !== undefined &&
          result.status === PlatformBuildStatus.READY && (
            <span className="text-[10px] ml-auto opacity-70">
              {formatBytes(Number(result.fileSize))}
            </span>
          )}
        {result?.status === PlatformBuildStatus.READY && (
          <FileDown className="h-3 w-3 ml-auto" />
        )}
      </button>

      {/* Show skip reason for skipped platforms */}
      {result?.status === PlatformBuildStatus.SKIPPED && result.skipReason && (
        <div className="bg-yellow-950/20 border border-yellow-800/30 rounded p-2 text-xs text-yellow-300">
          <div className="flex items-start gap-2">
            <AlertCircle className="h-3 w-3 flex-shrink-0 mt-0.5" />
            <div className="flex-1">{result.skipReason}</div>
          </div>
        </div>
      )}

      {/* Show error details for failed platforms */}
      {result?.status === PlatformBuildStatus.FAILED &&
        result.errorLog.length > 0 && (
          <div className="flex flex-col gap-1">
            <div className="flex gap-1">
              <Button
                variant="ghost"
                size="sm"
                className="h-6 px-2 text-xs"
                onClick={() => {
                  setShowError(!showError);
                }}
              >
                {showError ? (
                  <ChevronUp className="h-3 w-3" />
                ) : (
                  <ChevronDown className="h-3 w-3" />
                )}
                {showError ? "Hide" : "Show"} Error
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-6 px-2 text-xs"
                onClick={() => {
                  void handleCopyErrors();
                }}
              >
                {copied ? (
                  <Check className="h-3 w-3 text-green-400" />
                ) : (
                  <Copy className="h-3 w-3" />
                )}
                {copied ? "Copied" : "Copy"}
              </Button>
            </div>
            {showError && (
              <div className="bg-red-950/20 border border-red-800/30 rounded p-2 text-[10px] font-mono text-red-300 max-h-32 overflow-y-auto">
                {result.errorLog.map((error, idx) => (
                  <div
                    key={idx}
                    className="whitespace-pre-wrap break-words mb-1 opacity-90"
                  >
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
