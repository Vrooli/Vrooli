import { type KeyboardEvent } from "react";
import { Card, CardContent } from "../ui/card";
import { Badge } from "../ui/badge";
import { Monitor, Package, CheckCircle, XCircle, Clock, Download } from "lucide-react";
import { formatBytes, platformIcons } from "./utils";
import { Button } from "../ui/button";
import type { ScenarioDesktopStatus, DesktopBuildArtifact } from "./types";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface ScenarioCardProps {
  scenario: ScenarioDesktopStatus;
  onSelect: (scenario: ScenarioDesktopStatus) => void;
  isSelected?: boolean;
}

export function ScenarioCard({ scenario, onSelect, isSelected }: ScenarioCardProps) {
  const artifacts = scenario.build_artifacts || [];
  const uniquePlatforms = Array.from(
    new Set(artifacts.map((artifact: DesktopBuildArtifact) => artifact.platform).filter(Boolean))
  ) as string[];

  const handleOpenDetails = () => {
    onSelect(scenario);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      handleOpenDetails();
    }
  };

  const downloadPlatform = (platform?: string | null) => {
    if (!platform) return;
    const query = uniquePlatforms.includes(platform) ? platform : undefined;
    if (!query) return;
    const downloadUrl = buildUrl(`/desktop/download/${scenario.name}/${platform}`);
    window.open(downloadUrl, '_blank');
  };

  return (
    <Card
      className={`border transition-all hover:border-blue-600 focus-within:border-blue-600 ${
        isSelected ? 'border-blue-600 bg-blue-950/10' : scenario.has_desktop ? 'border-green-700 bg-green-950/10' : 'border-slate-700 bg-slate-800/50'
      }`}
      role="button"
      tabIndex={0}
      onClick={handleOpenDetails}
      onKeyDown={handleKeyDown}
    >
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-4">
          <div className="flex gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-slate-800">
              {scenario.has_desktop ? (
                <Monitor className="h-5 w-5 text-green-400" />
              ) : (
                <Package className="h-5 w-5 text-slate-400" />
              )}
            </div>
            <div className="space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <h4 className="text-lg font-semibold text-slate-100">{scenario.name}</h4>
                {!scenario.has_desktop && (
                  <Badge variant="secondary" className="text-xs bg-slate-700 text-slate-300">
                    <XCircle className="mr-1 h-3 w-3" />
                    Not Generated
                  </Badge>
                )}
                {scenario.has_desktop && !scenario.built && (
                  <Badge variant="warning" className="text-xs bg-yellow-900/30 text-yellow-400 border-yellow-700">
                    <Clock className="mr-1 h-3 w-3" />
                    Wrapper Ready
                  </Badge>
                )}
                {scenario.has_desktop && scenario.built && (
                  <Badge variant="success" className="text-xs">
                    <CheckCircle className="mr-1 h-3 w-3" />
                    Built
                  </Badge>
                )}
              </div>
              {scenario.display_name && (
                <p className="text-sm text-slate-400">{scenario.display_name}</p>
              )}
              <div className="text-xs text-slate-400">
                {scenario.desktop_path ? scenario.desktop_path : 'Desktop wrapper not generated yet'}
              </div>
            </div>
          </div>
          <div className="flex flex-col items-end gap-2 text-right text-xs text-slate-400">
            {scenario.version && <span>Version {scenario.version}</span>}
            {scenario.package_size && scenario.package_size > 0 && (
              <span>{formatBytes(scenario.package_size)}</span>
            )}
            {scenario.last_modified && <span>Updated {scenario.last_modified}</span>}
          </div>
        </div>

        {uniquePlatforms.length > 0 && (
          <div className="mt-4 flex flex-wrap items-center gap-2">
            <span className="text-xs uppercase tracking-wide text-slate-400">Downloads</span>
            {uniquePlatforms.map((platform) => (
              <Button
                key={platform}
                variant="outline"
                size="sm"
                className="gap-1"
                onClick={(event) => {
                  event.stopPropagation();
                  downloadPlatform(platform);
                }}
              >
                <span>{platformIcons[platform] || '📦'}</span>
                <Download className="h-3 w-3" />
              </Button>
            ))}
            <span className="text-[11px] text-slate-500">
              Click to download instantly or open for guided steps.
            </span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
