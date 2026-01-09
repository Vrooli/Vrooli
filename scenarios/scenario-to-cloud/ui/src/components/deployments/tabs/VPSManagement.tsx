import { useState } from "react";
import {
  Power,
  Square,
  Trash2,
  AlertTriangle,
  Loader2,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import { useVPSAction } from "../../../hooks/useLiveState";
import type { VPSAction, CleanupLevel } from "../../../lib/api";
import { cn } from "../../../lib/utils";
import { ConfirmationDialog } from "./ConfirmationDialog";

interface VPSManagementProps {
  deploymentId: string;
  deploymentName: string;
}

type ConfirmDialogState = {
  action: VPSAction;
  level?: CleanupLevel;
  title: string;
  description: string;
  confirmText: string;
  isDestructive: boolean;
} | null;

const CLEANUP_LEVELS: { level: CleanupLevel; title: string; description: string }[] = [
  {
    level: 1,
    title: "Remove Builds Only",
    description: "Delete scenario build artifacts. Processes keep running.",
  },
  {
    level: 2,
    title: "Stop + Remove Builds",
    description: "Stop all Vrooli processes and delete build artifacts.",
  },
  {
    level: 3,
    title: "Docker Reset",
    description: "Stop and prune all Docker containers, images, and volumes. Resets databases.",
  },
  {
    level: 4,
    title: "Remove Vrooli Installation",
    description: "Completely remove the Vrooli installation directory.",
  },
  {
    level: 5,
    title: "Full VPS Reset",
    description: "Remove Vrooli, prune Docker, and clean up system packages.",
  },
];

export function VPSManagement({ deploymentId, deploymentName }: VPSManagementProps) {
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialogState>(null);
  const [confirmInput, setConfirmInput] = useState("");
  const [result, setResult] = useState<{ ok: boolean; message: string } | null>(null);

  const vpsAction = useVPSAction(deploymentId);

  const handleAction = async () => {
    if (!confirmDialog) return;

    try {
      const response = await vpsAction.mutateAsync({
        action: confirmDialog.action,
        cleanup_level: confirmDialog.level,
        confirmation: confirmInput,
      });
      setResult({ ok: response.ok, message: response.message });
    } catch (error) {
      setResult({ ok: false, message: error instanceof Error ? error.message : "Action failed" });
    }

    setConfirmDialog(null);
    setConfirmInput("");
  };

  const openRebootDialog = () => {
    setResult(null);
    setConfirmDialog({
      action: "reboot",
      title: "Restart VPS",
      description: "This will reboot the server. The deployment will be temporarily unavailable until the server restarts.",
      confirmText: "REBOOT",
      isDestructive: true,
    });
  };

  const openStopDialog = () => {
    setResult(null);
    setConfirmDialog({
      action: "stop_vrooli",
      title: "Stop All Vrooli Processes",
      description: "This will stop all scenarios and resources managed by Vrooli. The VPS will remain running.",
      confirmText: deploymentName,
      isDestructive: true,
    });
  };

  const openCleanupDialog = (level: CleanupLevel) => {
    setResult(null);
    const levelInfo = CLEANUP_LEVELS.find((l) => l.level === level)!;
    let confirmText = deploymentName;
    if (level === 3) {
      confirmText = "DOCKER-RESET";
    } else if (level >= 4) {
      confirmText = "RESET";
    }
    setConfirmDialog({
      action: "cleanup",
      level,
      title: levelInfo.title,
      description: levelInfo.description + " This action cannot be undone.",
      confirmText,
      isDestructive: true,
    });
  };

  return (
    <div className="space-y-6">
      {/* Result Message */}
      {result && (
        <div
          className={cn(
            "flex items-center gap-3 p-4 rounded-lg border",
            result.ok
              ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-400"
              : "border-red-500/30 bg-red-500/10 text-red-400"
          )}
        >
          {result.ok ? (
            <CheckCircle2 className="h-5 w-5 flex-shrink-0" />
          ) : (
            <XCircle className="h-5 w-5 flex-shrink-0" />
          )}
          <span className="text-sm">{result.message}</span>
        </div>
      )}

      {/* Warning Banner */}
      <div className="flex items-start gap-3 p-4 rounded-lg border border-amber-500/30 bg-amber-500/5">
        <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
        <div>
          <h4 className="font-medium text-amber-400">Destructive Operations</h4>
          <p className="text-sm text-slate-400 mt-1">
            These actions can affect your deployment. Make sure you understand the consequences before proceeding.
          </p>
        </div>
      </div>

      {/* Action Cards Grid */}
      <div className="grid gap-4 md:grid-cols-2">
        {/* Restart VPS */}
        <ActionCard
          title="Restart VPS"
          description="Reboot the server. All services will restart automatically."
          icon={Power}
          color="amber"
          onClick={openRebootDialog}
          disabled={vpsAction.isPending}
        />

        {/* Stop All Vrooli */}
        <ActionCard
          title="Stop All Vrooli"
          description="Stop all scenarios and resources managed by Vrooli."
          icon={Square}
          color="red"
          onClick={openStopDialog}
          disabled={vpsAction.isPending}
        />
      </div>

      {/* Cleanup Levels */}
      <div>
        <h3 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2">
          <Trash2 className="h-4 w-4" />
          Cleanup Options
        </h3>
        <div className="grid gap-3">
          {CLEANUP_LEVELS.map(({ level, title, description }) => {
            const isHighRisk = level >= 3;
            return (
              <button
                key={level}
                onClick={() => openCleanupDialog(level)}
                disabled={vpsAction.isPending}
                className={cn(
                  "flex items-start gap-4 p-4 rounded-lg border text-left transition-colors",
                  "border-white/10 hover:border-red-500/30 hover:bg-red-500/5",
                  vpsAction.isPending && "opacity-50 cursor-not-allowed"
                )}
              >
                <div
                  className={cn(
                    "p-2 rounded-lg",
                    isHighRisk ? "bg-red-500/20" : "bg-amber-500/20"
                  )}
                >
                  <Trash2
                    className={cn(
                      "h-5 w-5",
                      isHighRisk ? "text-red-400" : "text-amber-400"
                    )}
                  />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h4 className="font-medium text-white">{title}</h4>
                    <span
                      className={cn(
                        "px-1.5 py-0.5 rounded text-xs font-medium",
                        isHighRisk
                          ? "bg-red-500/20 text-red-400"
                          : "bg-amber-500/20 text-amber-400"
                      )}
                    >
                      Level {level}
                    </span>
                  </div>
                  <p className="text-sm text-slate-400 mt-1">{description}</p>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Confirmation Dialog */}
      {confirmDialog && (
        <ConfirmationDialog
          title={confirmDialog.title}
          description={confirmDialog.description}
          confirmText={confirmDialog.confirmText}
          inputValue={confirmInput}
          onInputChange={setConfirmInput}
          onConfirm={handleAction}
          onCancel={() => {
            setConfirmDialog(null);
            setConfirmInput("");
          }}
          isPending={vpsAction.isPending}
          isDestructive={confirmDialog.isDestructive}
        />
      )}
    </div>
  );
}

interface ActionCardProps {
  title: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
  color: "amber" | "red";
  onClick: () => void;
  disabled?: boolean;
}

function ActionCard({
  title,
  description,
  icon: Icon,
  color,
  onClick,
  disabled,
}: ActionCardProps) {
  const colorClasses = {
    amber: {
      border: "border-amber-500/30",
      bg: "bg-amber-500/20",
      text: "text-amber-400",
      hover: "hover:bg-amber-500/5",
    },
    red: {
      border: "border-red-500/30",
      bg: "bg-red-500/20",
      text: "text-red-400",
      hover: "hover:bg-red-500/5",
    },
  };

  const colors = colorClasses[color];

  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "flex items-start gap-4 p-4 rounded-lg border text-left transition-colors",
        "border-white/10",
        colors.hover,
        `hover:${colors.border}`,
        disabled && "opacity-50 cursor-not-allowed"
      )}
    >
      <div className={cn("p-2 rounded-lg", colors.bg)}>
        <Icon className={cn("h-5 w-5", colors.text)} />
      </div>
      <div>
        <h4 className="font-medium text-white">{title}</h4>
        <p className="text-sm text-slate-400 mt-1">{description}</p>
      </div>
    </button>
  );
}
