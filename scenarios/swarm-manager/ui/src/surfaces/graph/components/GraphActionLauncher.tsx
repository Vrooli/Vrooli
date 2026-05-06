import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { GitPullRequestArrow, Loader2, MessageSquarePlus, Plus, Workflow, X } from "lucide-react";
import { FloatingActionButton } from "../../../components/ui/floating-action-button";
import { useGlobalKeyDown } from "../../../hooks/useGlobalKeyDown";
import { cn } from "../../../lib/utils";

interface GraphActionLauncherProps {
  isBusy?: boolean;
  error?: string | null;
  status?: string | null;
  onDismissError?: () => void;
  onQuickCapture: () => void;
  onPlanWork: () => void;
  onAuthorOperatingMode: () => void;
}

export function GraphActionLauncher({
  isBusy = false,
  error,
  status,
  onDismissError,
  onQuickCapture,
  onPlanWork,
  onAuthorOperatingMode,
}: GraphActionLauncherProps) {
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useGlobalKeyDown((event) => {
    if (event.key === "Escape") setIsOpen(false);
  }, { enabled: isOpen });

  useEffect(() => {
    if (!isOpen) return;

    const handlePointerDown = (event: PointerEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    window.addEventListener("pointerdown", handlePointerDown);
    return () => {
      window.removeEventListener("pointerdown", handlePointerDown);
    };
  }, [isOpen]);

  const runAction = (action: () => void) => {
    setIsOpen(false);
    action();
  };

  return (
    <div ref={menuRef} className="fixed bottom-[calc(3rem+env(safe-area-inset-bottom))] right-4 z-30" data-testid="graph-action-launcher">
      {isOpen && (
        <div
          className="mb-3 w-[min(18rem,calc(100vw-2rem))] rounded-lg border border-white/10 bg-slate-900/95 p-1.5 shadow-2xl backdrop-blur-sm"
          role="menu"
          aria-label="Create"
          data-testid="graph-action-menu"
        >
          <LauncherItem icon={<MessageSquarePlus className="h-4 w-4" />} label="Quick Capture" onClick={() => runAction(onQuickCapture)} />
          <LauncherItem icon={<Workflow className="h-4 w-4" />} label="Plan Work With Agent" onClick={() => runAction(onPlanWork)} disabled={isBusy} />
          <LauncherItem
            icon={<GitPullRequestArrow className="h-4 w-4" />}
            label="Author Operating Mode"
            onClick={() => runAction(onAuthorOperatingMode)}
            disabled={isBusy}
          />
        </div>
      )}

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
  onClick,
  disabled,
}: {
  icon: ReactNode;
  label: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      onClick={onClick}
      disabled={disabled}
      className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm font-medium text-slate-100 transition-colors hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/40 disabled:pointer-events-none disabled:opacity-50"
    >
      <span className="shrink-0 text-cyan-300">{icon}</span>
      <span className="min-w-0 truncate">{label}</span>
    </button>
  );
}
