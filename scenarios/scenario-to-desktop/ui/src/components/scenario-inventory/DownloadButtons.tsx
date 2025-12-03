import { Button } from "../ui/button";
import { Download, Copy, Check } from "lucide-react";
import { platformIcons, platformNames, formatBytes } from "./utils";
import type { DesktopBuildArtifact } from "./types";
import { useState } from "react";
import { getDownloadUrl } from "../../lib/api";

interface DownloadButtonsProps {
  scenarioName: string;
  artifacts: DesktopBuildArtifact[];
}

export function DownloadButtons({ scenarioName, artifacts }: DownloadButtonsProps) {
  const [copiedPath, setCopiedPath] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  if (!artifacts?.length) {
    return null;
  }

  const artifactByPlatform: Record<string, DesktopBuildArtifact[]> = {};
  artifacts.forEach((artifact) => {
    const key = artifact.platform || "unknown";
    if (!artifactByPlatform[key]) {
      artifactByPlatform[key] = [];
    }
    artifactByPlatform[key].push(artifact);
  });

  const handleDownload = (platform: string) => {
    window.open(getDownloadUrl(scenarioName, platform), '_blank');
  };

  const handleCopyPath = (path?: string) => {
    if (!path) return;
    navigator.clipboard.writeText(path);
    setCopiedPath(path);
    setTimeout(() => setCopiedPath(null), 2000);
  };

  return (
    <div className="space-y-3">
      {Object.entries(artifactByPlatform).map(([platform, platformArtifacts]) => (
        <div key={platform} className="rounded-lg border border-slate-700 bg-slate-900/40 p-3 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xl">{platformIcons[platform] || '📦'}</span>
            <p className="text-sm font-semibold text-slate-100">
              {platformNames[platform] || 'Installer'}
            </p>
            <Button
              variant="outline"
              size="sm"
              className="gap-1"
              onClick={() => handleDownload(platform)}
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
                  [platform]: !prev[platform],
                }))
              }
            >
              {expanded[platform] ? 'Hide file list' : 'Show file paths'}
            </Button>
          </div>
          {expanded[platform] && (
            <div className="max-h-48 overflow-y-auto rounded border border-slate-800 bg-black/30 p-2 text-xs text-slate-400">
              {platformArtifacts.map((artifact) => (
                <div key={artifact.absolute_path} className="flex flex-wrap items-center gap-2 py-1">
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
