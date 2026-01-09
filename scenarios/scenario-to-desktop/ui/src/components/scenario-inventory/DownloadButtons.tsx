import { Button } from "../ui/button";
import { Download, Copy, Check, FileDown } from "lucide-react";
import { useState, useCallback } from "react";
import { getDownloadUrl } from "../../lib/api";
import { triggerDownload, writeToClipboard } from "../../lib/browser";
import {
  getAvailablePlatforms,
  groupArtifactsByPlatform,
  getPlatformIcon,
  getPlatformName,
  formatBytes,
  type Platform,
  type DesktopBuildArtifact
} from "../../domain/download";

interface DownloadButtonsProps {
  scenarioName: string;
  artifacts: DesktopBuildArtifact[];
}

export function DownloadButtons({ scenarioName, artifacts }: DownloadButtonsProps) {
  const [copiedPath, setCopiedPath] = useState<string | null>(null);

  // Get only valid platforms (filters out "unknown")
  const availablePlatforms = getAvailablePlatforms(artifacts);
  const [selectedPlatform, setSelectedPlatform] = useState<Platform | null>(
    availablePlatforms[0] ?? null
  );

  const platformGroups = groupArtifactsByPlatform(artifacts);
  const selectedGroup = selectedPlatform ? platformGroups.get(selectedPlatform) : null;

  const handleDownload = useCallback(
    (platform: Platform) => {
      const url = getDownloadUrl(scenarioName, platform);
      triggerDownload({ url });
    },
    [scenarioName]
  );

  const handleCopyPath = useCallback(async (path?: string) => {
    if (!path) return;
    const result = await writeToClipboard(path);
    if (result.success) {
      setCopiedPath(path);
      setTimeout(() => setCopiedPath(null), 2000);
    }
  }, []);

  if (availablePlatforms.length === 0) {
    return null;
  }

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-4">
      {/* Platform selector tabs */}
      <div className="flex items-center gap-1 p-1 rounded-lg bg-slate-800/60 w-fit">
        {availablePlatforms.map((platform) => (
          <button
            key={platform}
            onClick={() => setSelectedPlatform(platform)}
            className={`
              flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors
              ${selectedPlatform === platform
                ? "bg-blue-600 text-white shadow-sm"
                : "text-slate-300 hover:text-white hover:bg-slate-700/60"
              }
            `}
          >
            <span>{getPlatformIcon(platform)}</span>
            <span>{getPlatformName(platform)}</span>
          </button>
        ))}
      </div>

      {/* Selected platform download area */}
      {selectedGroup && selectedPlatform && (
        <div className="space-y-3">
          {/* Main download button with file info */}
          <div className="flex flex-wrap items-center gap-3">
            <Button
              size="default"
              className="gap-2"
              onClick={() => handleDownload(selectedPlatform)}
            >
              <Download className="h-4 w-4" />
              Download {getPlatformName(selectedPlatform)} installer
            </Button>
            <span className="text-sm text-slate-400">
              {formatBytes(selectedGroup.totalSizeBytes)}
              {selectedGroup.artifacts.length > 1 && (
                <span className="ml-1">({selectedGroup.artifacts.length} files)</span>
              )}
            </span>
          </div>

          {/* File list */}
          <div className="rounded border border-slate-800 bg-black/30 p-3 space-y-2">
            <p className="text-xs font-medium text-slate-400 flex items-center gap-1.5">
              <FileDown className="h-3 w-3" />
              Included files
            </p>
            <div className="space-y-1.5">
              {selectedGroup.artifacts.map((artifact) => (
                <div
                  key={artifact.absolute_path || artifact.file_name}
                  className="flex flex-wrap items-center gap-2 text-xs"
                >
                  <span className="text-slate-200">{artifact.file_name}</span>
                  {artifact.size_bytes !== undefined && (
                    <span className="text-slate-500">({formatBytes(artifact.size_bytes)})</span>
                  )}
                  {artifact.relative_path && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-5 px-1.5 gap-1 text-xs"
                      onClick={() => handleCopyPath(artifact.relative_path)}
                    >
                      {copiedPath === artifact.relative_path ? (
                        <Check className="h-3 w-3 text-green-400" />
                      ) : (
                        <Copy className="h-3 w-3" />
                      )}
                      Copy path
                    </Button>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
