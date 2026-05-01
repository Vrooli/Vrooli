import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { GitPullRequestArrow, MessageSquarePlus, Plus, Workflow } from "lucide-react";
import { FloatingActionButton } from "../../../components/ui/floating-action-button";
import { cn } from "../../../lib/utils";

interface GraphActionLauncherProps {
  isBusy?: boolean;
  error?: string | null;
  onQuickCapture: () => void;
  onPlanWork: () => void;
  onAuthorOperatingMode: () => void;
}

export function GraphActionLauncher({
  isBusy = false,
  error,
  onQuickCapture,
  onPlanWork,
  onAuthorOperatingMode,
}: GraphActionLauncherProps) {
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setIsOpen(false);
    };
    const handlePointerDown = (event: PointerEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("pointerdown", handlePointerDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
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
          {error && (
            <p className="mt-1 rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1.5 text-xs text-red-200" role="alert">
              {error}
            </p>
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
