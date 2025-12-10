import { CheckCircle, XCircle, AlertCircle, Wrench } from "lucide-react";
import type { ToolDetectionResult } from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { cn } from "../../lib/utils";

interface PrerequisitesPanelProps {
  tools: ToolDetectionResult[];
}

const PLATFORM_LABELS: Record<string, string> = {
  windows: "Windows",
  macos: "macOS",
  linux: "Linux",
  all: "All Platforms"
};

const TOOL_DESCRIPTIONS: Record<string, string> = {
  signtool: "Microsoft SignTool for authenticode signing",
  osslsigncode: "Open-source alternative for Windows signing on Linux/macOS",
  codesign: "Apple codesign utility for macOS signing",
  notarytool: "Apple notarization tool (Xcode 13+)",
  altool: "Legacy Apple notarization tool",
  gpg: "GNU Privacy Guard for Linux package signing",
  rpmsign: "RPM package signing utility",
  dpkg_sig: "Debian package signing utility"
};

export function PrerequisitesPanel({ tools }: PrerequisitesPanelProps) {
  if (!tools || tools.length === 0) {
    return (
      <Card className="border-slate-800/80 bg-slate-900/70">
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Wrench className="h-5 w-5 text-slate-400" />
            Signing Tools
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-slate-400 text-center py-4">
            No tool detection results available.
          </p>
        </CardContent>
      </Card>
    );
  }

  // Group tools by platform
  const toolsByPlatform = tools.reduce((acc, tool) => {
    const platform = tool.platform || "all";
    if (!acc[platform]) {
      acc[platform] = [];
    }
    acc[platform].push(tool);
    return acc;
  }, {} as Record<string, ToolDetectionResult[]>);

  const platformOrder = ["windows", "macos", "linux", "all"];
  const sortedPlatforms = platformOrder.filter(p => toolsByPlatform[p]);

  return (
    <Card className="border-slate-800/80 bg-slate-900/70">
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <Wrench className="h-5 w-5 text-slate-400" />
          Signing Tools
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-slate-400">
          The following signing tools have been detected on this system:
        </p>

        <div className="grid gap-4 md:grid-cols-3">
          {sortedPlatforms.map(platform => (
            <div
              key={platform}
              className="p-3 rounded-lg border border-slate-800 bg-slate-950/30"
            >
              <h4 className="font-medium text-sm mb-3">
                {PLATFORM_LABELS[platform] || platform}
              </h4>
              <div className="space-y-2">
                {toolsByPlatform[platform].map(tool => (
                  <ToolStatus key={`${platform}-${tool.tool}`} tool={tool} />
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* Legend */}
        <div className="flex items-center gap-4 pt-3 border-t border-slate-800 text-xs text-slate-500">
          <span className="flex items-center gap-1">
            <CheckCircle className="h-3 w-3 text-green-400" />
            Installed
          </span>
          <span className="flex items-center gap-1">
            <XCircle className="h-3 w-3 text-slate-500" />
            Not found
          </span>
          <span className="flex items-center gap-1">
            <AlertCircle className="h-3 w-3 text-amber-400" />
            Issue detected
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function ToolStatus({ tool }: { tool: ToolDetectionResult }) {
  const description = TOOL_DESCRIPTIONS[tool.tool] || tool.tool;

  return (
    <div
      className={cn(
        "p-2 rounded border",
        tool.installed
          ? "border-green-800/50 bg-green-950/20"
          : tool.error
          ? "border-amber-800/50 bg-amber-950/20"
          : "border-slate-800 bg-slate-950/20"
      )}
    >
      <div className="flex items-start gap-2">
        {tool.installed ? (
          <CheckCircle className="h-4 w-4 text-green-400 mt-0.5 flex-shrink-0" />
        ) : tool.error ? (
          <AlertCircle className="h-4 w-4 text-amber-400 mt-0.5 flex-shrink-0" />
        ) : (
          <XCircle className="h-4 w-4 text-slate-500 mt-0.5 flex-shrink-0" />
        )}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-mono text-xs">{tool.tool}</span>
            {tool.version && (
              <span className="text-xs text-slate-500">v{tool.version}</span>
            )}
          </div>
          <p className="text-xs text-slate-500 mt-0.5 truncate" title={description}>
            {description}
          </p>
          {tool.path && (
            <p className="text-xs text-slate-600 mt-0.5 font-mono truncate" title={tool.path}>
              {tool.path}
            </p>
          )}
          {tool.error && (
            <p className="text-xs text-amber-300 mt-1">{tool.error}</p>
          )}
          {tool.remediation && !tool.installed && (
            <p className="text-xs text-slate-400 mt-1">{tool.remediation}</p>
          )}
        </div>
      </div>
    </div>
  );
}
