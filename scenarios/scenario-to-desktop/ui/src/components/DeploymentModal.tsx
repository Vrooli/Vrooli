import { useState } from "react";
import { createPortal } from "react-dom";
import { Cloud, Database, HardDrive, Info, Package, Server, Sparkles, Terminal, X } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { DEPLOYMENT_OPTIONS, SERVER_TYPE_OPTIONS, type DeploymentMode, type ServerType } from "../domain/deployment";

type DeploymentStatus = "recommended" | "experimental" | "coming_soon";

const DEPLOYMENT_VISUALS: Record<DeploymentMode, { icon: typeof Server; gradient: string; status: DeploymentStatus }> = {
  "external-server": {
    icon: Server,
    gradient: "from-blue-700/80 via-indigo-600/70 to-slate-700/70",
    status: "recommended"
  },
  "bundled": {
    icon: Package,
    gradient: "from-emerald-700/80 via-teal-600/70 to-slate-700/70",
    status: "experimental"
  },
  "cloud-api": {
    icon: Cloud,
    gradient: "from-amber-700/80 via-orange-600/70 to-slate-700/70",
    status: "coming_soon"
  }
};

const SERVER_TYPE_VISUALS: Record<ServerType, { icon: typeof Database; gradient: string; status: "available" | "experimental" }> = {
  external: {
    icon: Database,
    gradient: "from-blue-700/80 via-slate-700/70 to-slate-800/70",
    status: "available"
  },
  static: {
    icon: HardDrive,
    gradient: "from-slate-700/80 via-slate-600/70 to-slate-800/70",
    status: "available"
  },
  node: {
    icon: Terminal,
    gradient: "from-purple-700/80 via-indigo-700/70 to-slate-800/70",
    status: "experimental"
  },
  executable: {
    icon: Package,
    gradient: "from-rose-700/80 via-red-700/70 to-slate-800/70",
    status: "experimental"
  }
};

interface DeploymentModalProps {
  open: boolean;
  deploymentMode: DeploymentMode;
  serverType: ServerType;
  allowedServerTypes: ServerType[];
  onChange: (mode: DeploymentMode, serverType?: ServerType) => void;
  onClose: () => void;
}

export function DeploymentModal({
  open,
  deploymentMode,
  serverType,
  allowedServerTypes,
  onChange,
  onClose
}: DeploymentModalProps) {
  const [showHelper, setShowHelper] = useState(false);

  if (!open) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <Card className="w-full max-w-6xl border-slate-800 bg-slate-950/90 shadow-xl">
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle className="text-lg text-slate-100">Choose deployment + data source</CardTitle>
            <p className="text-sm text-slate-400">
              Pick how the desktop app connects, then decide where its UI and APIs will come from.
            </p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={onClose} className="h-8 w-8 p-0">
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-start gap-3 rounded-lg border border-slate-800/80 bg-slate-950/60 p-3 text-sm text-slate-200">
            <Info className="mt-0.5 h-5 w-5 text-blue-300" />
            <div className="space-y-1">
              <p className="font-semibold text-slate-100">Start with the deployment intent.</p>
              <p className="text-slate-300">
                Thin client is the simplest path today. Bundled is for offline experiments and requires a manifest.
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-slate-300">
              Current selection:{" "}
              <span className="font-semibold text-slate-100">
                {deploymentMode} + {serverType}
              </span>
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setShowHelper((prev) => !prev)}
              className="gap-2"
            >
              <Sparkles className="h-4 w-4" />
              Help me decide
            </Button>
          </div>

          {showHelper && (
            <div className="rounded-lg border border-blue-800/70 bg-blue-950/30 p-4 space-y-3">
              <p className="text-sm font-semibold text-slate-100">Quick guidance</p>
              <div className="text-sm text-slate-200 space-y-2">
                <p>
                  If your scenario is already running on a server, choose <strong>Thin client</strong> +{" "}
                  <strong>External</strong>.
                </p>
                <p>
                  If the app is UI-only (no API), choose <strong>Thin client</strong> + <strong>Static files</strong>.
                </p>
                <p>
                  If you want everything inside the installer, choose <strong>Bundled</strong> (experimental) and
                  prepare a bundle manifest.
                </p>
                <p>
                  <strong>Node</strong> and <strong>Executable</strong> are for advanced local services and need extra
                  setup.
                </p>
              </div>
            </div>
          )}

          <div className="space-y-3">
            <p className="text-sm font-semibold text-slate-200">Deployment intent</p>
            <div className="grid gap-4 md:grid-cols-3">
              {DEPLOYMENT_OPTIONS.map((option) => {
                const visual = DEPLOYMENT_VISUALS[option.value];
                const isSelected = deploymentMode === option.value;
                const disabled = visual.status === "coming_soon";
                const nextAllowed = option.value === "bundled" || option.value === "cloud-api"
                  ? ["external"]
                  : SERVER_TYPE_OPTIONS.map((server) => server.value);
                const badgeText =
                  visual.status === "recommended"
                    ? "Recommended"
                    : visual.status === "experimental"
                      ? "Experimental"
                      : "Coming soon";
                return (
                  <button
                    key={option.value}
                    type="button"
                    disabled={disabled}
                    onClick={() => onChange(option.value, nextAllowed[0] ?? serverType)}
                    className={`rounded-lg border border-slate-800/80 bg-slate-950/60 p-4 text-left transition-all ${
                      isSelected ? "shadow-[inset_0_0_0_2px_rgba(59,130,246,0.9)]" : ""
                    } ${disabled ? "opacity-60 cursor-not-allowed" : "hover:scale-[1.01]"}`}
                  >
                    <div className={`mb-3 flex items-center gap-3 rounded-md bg-gradient-to-r ${visual.gradient} p-3`}>
                      <visual.icon className="h-6 w-6 text-white/90" />
                      <div>
                        <p className="text-sm font-semibold text-white">{option.label}</p>
                        <p className="text-xs text-slate-100/90">{option.value}</p>
                      </div>
                    </div>
                    <p className="text-xs text-slate-300">{option.description}</p>
                    <div className="mt-3">
                      <Badge variant="outline" className="text-xs text-slate-300">
                        {badgeText}
                      </Badge>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-sm font-semibold text-slate-200">Where should this desktop build get its data?</p>
              {deploymentMode === "bundled" && (
                <Badge variant="outline" className="text-xs text-amber-200 border-amber-700/60">
                  Bundled limits this selection
                </Badge>
              )}
            </div>
            <div className="grid gap-4 md:grid-cols-4">
              {SERVER_TYPE_OPTIONS.map((option) => {
                const visual = SERVER_TYPE_VISUALS[option.value];
                const isSelected = serverType === option.value;
                const isAllowed = allowedServerTypes.includes(option.value);
                const statusBadge =
                  visual.status === "experimental" ? (
                    <Badge variant="outline" className="text-xs text-amber-200 border-amber-700/60">
                      Experimental
                    </Badge>
                  ) : null;
                return (
                  <button
                    key={option.value}
                    type="button"
                    disabled={!isAllowed}
                    onClick={() => onChange(deploymentMode, option.value)}
                    className={`rounded-lg border border-slate-800/80 bg-slate-950/60 p-4 text-left transition-all ${
                      isSelected ? "shadow-[inset_0_0_0_2px_rgba(59,130,246,0.9)]" : ""
                    } ${isAllowed ? "hover:scale-[1.01]" : "opacity-50 cursor-not-allowed"}`}
                  >
                    <div className={`mb-3 flex items-center gap-3 rounded-md bg-gradient-to-r ${visual.gradient} p-3`}>
                      <visual.icon className="h-5 w-5 text-white/90" />
                      <div>
                        <p className="text-sm font-semibold text-white">{option.label}</p>
                        <p className="text-xs text-slate-100/90">{option.value}</p>
                      </div>
                    </div>
                    <p className="text-xs text-slate-300">{option.description}</p>
                    <div className="mt-3 flex items-center gap-2">
                      {!isAllowed && (
                        <Badge variant="outline" className="text-xs text-slate-400">
                          Not available here
                        </Badge>
                      )}
                      {statusBadge}
                    </div>
                  </button>
                );
              })}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>,
    document.body
  );
}
