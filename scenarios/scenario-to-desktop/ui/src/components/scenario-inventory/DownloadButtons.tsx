import { Button } from "../ui/button";
import { Download, Copy, Check } from "lucide-react";
import { useState, useCallback } from "react";
import { getDownloadUrl } from "../../lib/api";
import { triggerDownload, writeToClipboard } from "../../lib/browser";
import {
  getSortedPlatformGroups,
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
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  // Use domain function for grouping artifacts by platform
  const platformGroups = getSortedPlatformGroups(artifacts);

  const handleDownload = useCallback(
    (platform: Platform | "unknown") => {
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

  if (platformGroups.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      {platformGroups.map((group) => (
        <div
          key={group.platform}
          className="rounded-lg border border-slate-700 bg-slate-900/40 p-3 space-y-2"
        >
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xl">{getPlatformIcon(group.platform)}</span>
            <p className="text-sm font-semibold text-slate-100">
              {getPlatformName(group.platform)}
            </p>
            <Button
              variant="outline"
              size="sm"
              className="gap-1"
              onClick={() => handleDownload(group.platform)}
            >
              <Download className="h-3 w-3" /> Download packaged build
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs"
              onClick={() =>
                setExpanded((prev) => ({
                  ...prev,
                  [group.platform]: !prev[group.platform]
                }))
              }
            >
              {expanded[group.platform] ? "Hide file list" : "Show file paths"}
            </Button>
          </div>
          {expanded[group.platform] && (
            <div className="max-h-48 overflow-y-auto rounded border border-slate-800 bg-black/30 p-2 text-xs text-slate-400">
              {group.artifacts.map((artifact) => (
                <div
                  key={artifact.absolute_path}
                  className="flex flex-wrap items-center gap-2 py-1"
                >
                  <span>{artifact.file_name}</span>
                  {artifact.size_bytes && <span>({formatBytes(artifact.size_bytes)})</span>}
                  {artifact.relative_path && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 gap-1"
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
          )}
        </div>
      ))}
    </div>
  );
}
