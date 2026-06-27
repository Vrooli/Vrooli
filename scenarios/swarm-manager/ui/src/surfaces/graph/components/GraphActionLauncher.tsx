import { useState } from "react";
import type { ReactNode } from "react";
import { Loader2, MessageSquarePlus, Plus, X } from "lucide-react";
import { BottomSheet } from "../../../components/ui/bottom-sheet";
import { FloatingActionButton } from "../../../components/ui/floating-action-button";
import { cn } from "../../../lib/utils";
import {
  SESSION_KIND_DESCRIPTIONS,
  SESSION_KIND_ICONS,
  SESSION_KIND_LAUNCHER_LABELS,
} from "../../../components/session/session-view-model";

interface GraphActionLauncherProps {
  isBusy?: boolean;
  error?: string | null;
  status?: string | null;
  onDismissError?: () => void;
  onQuickCapture: () => void;
  onPlanWork: () => void;
  onManageSwarm: () => void;
  onAuthorOperatingMode: () => void;
}

export function GraphActionLauncher({
  isBusy = false,
  error,
  status,
  onDismissError,
  onQuickCapture,
  onPlanWork,
  onManageSwarm,
  onAuthorOperatingMode,
}: GraphActionLauncherProps) {
  const [isOpen, setIsOpen] = useState(false);

  const runAction = (action: () => void) => {
    setIsOpen(false);
    action();
  };

  return (
    <div className="fixed bottom-[calc(3rem+env(safe-area-inset-bottom))] right-4 z-30" data-testid="graph-action-launcher">
      <BottomSheet
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="Quick Capture"
        description="Choose how to capture, plan, or move Swarm Manager work forward."
        contentClassName="px-0 py-2 pb-[calc(0.5rem+env(safe-area-inset-bottom))]"
        data-testid="graph-action-menu"
      >
        <div role="menu" aria-label="Create">
          <LauncherItem
            icon={<MessageSquarePlus className="h-5 w-5" />}
            label="Quick Capture"
            description="Capture a note, task, dependency, or relationship without starting an agent session."
            onClick={() => runAction(onQuickCapture)}
          />
          <LauncherItem
            icon={<SESSION_KIND_ICONS.meta_orchestration className="h-5 w-5" />}
            label={SESSION_KIND_LAUNCHER_LABELS.meta_orchestration}
            description={SESSION_KIND_DESCRIPTIONS.meta_orchestration}
            onClick={() => runAction(onPlanWork)}
            disabled={isBusy}
          />
          <LauncherItem
            icon={<SESSION_KIND_ICONS.swarm_operations className="h-5 w-5" />}
            label={SESSION_KIND_LAUNCHER_LABELS.swarm_operations}
            description={SESSION_KIND_DESCRIPTIONS.swarm_operations}
            onClick={() => runAction(onManageSwarm)}
            disabled={isBusy}
          />
          <LauncherItem
            icon={<SESSION_KIND_ICONS.operating_mode_authoring className="h-5 w-5" />}
            label={SESSION_KIND_LAUNCHER_LABELS.operating_mode_authoring}
            description={SESSION_KIND_DESCRIPTIONS.operating_mode_authoring}
            onClick={() => runAction(onAuthorOperatingMode)}
            disabled={isBusy}
          />
        </div>
      </BottomSheet>

      {(status || error) && (
        <div className="mb-3 w-[min(18rem,calc(100vw-2rem))] rounded-lg border border-white/10 bg-slate-900/95 p-2 text-xs shadow-2xl backdrop-blur-sm">
          {status && !error && (
            <div className="flex items-center gap-2 text-slate-200" role="status" data-testid="graph-action-status">
              <Loader2 className="h-3.5 w-3.5 animate-spin text-cyan-300" />
              <span>{status}</span>
            </div>
          )}
          {error && (
            <div className="flex items-start gap-2 text-red-200" role="alert" data-testid="graph-action-error">
              <span className="min-w-0 flex-1 break-words">{error}</span>
              {onDismissError && (
                <button
                  type="button"
                  onClick={onDismissError}
                  className="shrink-0 rounded p-0.5 text-red-200/80 hover:bg-red-500/10 hover:text-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500/40"
                  aria-label="Dismiss error"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          )}
        </div>
      )}

      <FloatingActionButton
        icon={<Plus className={cn("h-5 w-5 transition-transform", isOpen && "rotate-45")} />}
        label="Create"
        onClick={() => setIsOpen((prev) => !prev)}
        aria-expanded={isOpen}
        data-testid="graph-action-fab"
      />
    </div>
  );
}

function LauncherItem({
  icon,
  label,
  description,
  onClick,
  disabled,
}: {
  icon: ReactNode;
  label: string;
  description: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      aria-label={label}
      onClick={onClick}
      disabled={disabled}
      className="flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-cyan-500/40 disabled:pointer-events-none disabled:opacity-50"
    >
      <span className="mt-0.5 shrink-0 text-cyan-300">{icon}</span>
      <span className="min-w-0">
        <span className="block text-base font-medium leading-5 text-slate-100">{label}</span>
        <span className="mt-1 block text-sm leading-5 text-slate-400">{description}</span>
      </span>
    </button>
  );
}
