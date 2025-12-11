import { useState } from "react";
import { CheckCircle, XCircle, AlertCircle, Wrench, RefreshCw, Info } from "lucide-react";
import type { ToolDetectionResult } from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { cn } from "../../lib/utils";

interface PrerequisitesPanelProps {
  tools: ToolDetectionResult[];
  onRefresh?: () => void;
  refreshing?: boolean;
}

const PLATFORM_LABELS: Record<string, string> = {
  windows: "Windows",
  macos: "macOS",
  linux: "Linux",
  all: "All Platforms"
};

const PLATFORM_INSTRUCTIONS: Record<
  "windows" | "macos" | "linux",
  { title: string; steps: string[]; command?: string; note?: string }
> = {
  windows: {
    title: "Windows",
    steps: [
      "Install Windows SDK (includes signtool.exe)",
      "Visual Studio installer -> Individual components -> Windows 10/11 SDK"
    ],
    command: "Download: https://aka.ms/vs/17/release/vs_buildtools.exe",
    note: "Required for Authenticode signing; EV tokens must be used on Windows."
  },
  macos: {
    title: "macOS",
    steps: ["Install Xcode Command Line Tools for codesign/notarytool"],
    command: "Run: xcode-select --install",
    note: "codesign/notarytool only exist on macOS; needed for Gatekeeper trust."
  },
  linux: {
    title: "Linux",
    steps: ["Install GPG and package signers for DEB/RPM", "Install osslsigncode if signing Windows EXEs on Linux"],
    command: "Debian/Ubuntu: sudo apt install gnupg dpkg-sig rpm-sign osslsigncode",
    note: "GPG signs DEB/RPM/AppImage; osslsigncode can sign Windows EXEs from Linux/macOS."
  }
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

function installHint(tool: ToolDetectionResult): string | undefined {
  if (tool.installed) return;
  if (tool.tool === "signtool") return "Install the Windows 10/11 SDK or Visual Studio (includes signtool.exe).";
  if (tool.tool === "osslsigncode") return "Install osslsigncode (e.g., brew install osslsigncode or apt install osslsigncode).";
  if (tool.tool === "codesign" || tool.tool === "notarytool" || tool.tool === "altool") {
    return "Install Xcode Command Line Tools: xcode-select --install (macOS only).";
  }
  if (tool.tool === "gpg") return "Install GPG (e.g., brew install gnupg or apt install gnupg).";
  if (tool.tool === "rpmsign") return "Install rpm-sign (e.g., yum install rpm-sign).";
  if (tool.tool === "dpkg_sig") return "Install dpkg-sig (e.g., apt install dpkg-sig).";
  return;
}

function installCommand(tool: ToolDetectionResult): string | undefined {
  if (tool.installed) return;
  switch (tool.tool) {
    case "signtool":
      return "Windows: Install Windows SDK (Visual Studio installer).";
    case "osslsigncode":
      return "macOS: brew install osslsigncode · Ubuntu/Debian: apt install osslsigncode";
    case "codesign":
    case "notarytool":
    case "altool":
      return "macOS: xcode-select --install";
    case "gpg":
      return "macOS: brew install gnupg · Ubuntu/Debian: apt install gnupg";
    case "rpmsign":
      return "RHEL/CentOS/Fedora: yum install rpm-sign";
    case "dpkg_sig":
      return "Ubuntu/Debian: apt install dpkg-sig";
    default:
      return;
  }
}

type PlatformKey = "windows" | "macos" | "linux" | "all";

function detectHostPlatform(): PlatformKey {
  if (typeof navigator === "undefined") return "windows";
  const ua = navigator.userAgent || "";
  if (/Mac/.test(ua)) return "macos";
  if (/Windows/.test(ua)) return "windows";
  if (/Linux/.test(ua)) return "linux";
  return "windows";
}

export function PrerequisitesPanel({ tools, onRefresh, refreshing }: PrerequisitesPanelProps) {
  const [selectedPlatform, setSelectedPlatform] = useState<PlatformKey>(detectHostPlatform());

  if (!tools || tools.length === 0) {
    const platforms: PlatformKey[] = ["windows", "macos", "linux"];
    return (
      <Card className="border-slate-800/80 bg-slate-900/70">
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Wrench className="h-5 w-5 text-slate-400" />
            Signing Tools
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-sm text-amber-200 flex items-center gap-2">
                <AlertCircle className="h-4 w-4" />
                Signing tools missing on this machine. Install the CLI for a platform, then re-scan.
              </p>
              <p className="text-xs text-slate-400 mt-1">
                Auto-selected {PLATFORM_LABELS[selectedPlatform] || "platform"} based on your OS. You can view others to prep ahead.
              </p>
            </div>
            {onRefresh && (
              <button
                onClick={onRefresh}
                className="inline-flex items-center gap-1 rounded border border-slate-700 px-3 py-1 text-xs text-slate-200 hover:border-slate-500"
                disabled={refreshing}
              >
                <RefreshCw className={cn("h-4 w-4", refreshing && "animate-spin")} />
                {refreshing ? "Re-scanning..." : "Re-scan"}
              </button>
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            {platforms.map((p) => (
              <button
                key={p}
                onClick={() => setSelectedPlatform(p)}
                className={cn(
                  "px-3 py-1 rounded-full border text-xs",
                  selectedPlatform === p
                    ? "border-blue-500 text-blue-100 bg-blue-900/40"
                    : "border-slate-700 text-slate-200 hover:border-slate-500"
                )}
              >
                {PLATFORM_LABELS[p]}
              </button>
            ))}
          </div>

          {selectedPlatform !== "all" && (
            <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-4 space-y-2">
              <div className="flex items-center gap-2 text-sm text-slate-200">
                <Info className="h-4 w-4 text-blue-300" />
                {PLATFORM_INSTRUCTIONS[selectedPlatform].title} setup
              </div>
              <ul className="list-disc list-inside text-sm text-slate-300 space-y-1">
                {PLATFORM_INSTRUCTIONS[selectedPlatform].steps.map((step) => (
                  <li key={step}>{step}</li>
                ))}
              </ul>
              {PLATFORM_INSTRUCTIONS[selectedPlatform].command && (
                <p className="text-xs font-mono text-slate-400">
                  {PLATFORM_INSTRUCTIONS[selectedPlatform].command}
                </p>
              )}
              {PLATFORM_INSTRUCTIONS[selectedPlatform].note && (
                <p className="text-xs text-slate-400">{PLATFORM_INSTRUCTIONS[selectedPlatform].note}</p>
              )}
            </div>
          )}

          <p className="text-xs text-slate-500">
            Need full details? See SIGNING.md for platform requirements and notarization/EV notes.
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
          {!tool.installed && installHint(tool) && (
            <p className="text-xs text-slate-400 mt-1">{installHint(tool)}</p>
          )}
          {!tool.installed && installCommand(tool) && (
            <p className="text-xs text-slate-500 mt-1 font-mono">{installCommand(tool)}</p>
          )}
        </div>
      </div>
    </div>
  );
}
