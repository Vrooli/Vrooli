/**
 * SidebarHeader — Top bar for the graph sidebar.
 *
 * Shows the app title, a home button (returns to graph view), the
 * always-visible Operations Center trigger pill, a settings gear, and a
 * collapse/close button. The trigger reads its count from
 * `useOperationsStore`, so the sidebar header does not need activity-list
 * or stop-run plumbing.
 */

import { Home, PanelLeft, Settings, X } from "lucide-react";
import { CommandPostButton } from "../../../../components/command-post/CommandPostButton";
import { OpsTriggerButton } from "../../../../components/operations/OpsTriggerButton";
import { useCommandPostBadgeCount } from "../../../../hooks/useCommandPostBadgeCount";

export interface SidebarHeaderProps {
  onSettingsOpen: () => void;
  onCollapse: () => void;
  onGoHome: () => void;
  onOpenCommandPost?: () => void;
}

export function SidebarHeader({
  onSettingsOpen,
  onCollapse,
  onGoHome,
  onOpenCommandPost,
}: SidebarHeaderProps) {
  const commandPostBadgeCount = useCommandPostBadgeCount();

  return (
    <div className="flex h-10 shrink-0 items-center justify-between border-b border-slate-200/20 px-3">
      {/* Left: Home button + App title */}
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          onClick={onGoHome}
          className="rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Go to graph view"
          data-testid="sidebar-home"
        >
          <Home className="h-4 w-4" />
        </button>
        <span className="text-sm font-semibold text-slate-200">Swarm Manager</span>
      </div>

      {/* Right: Command Post (mobile) + Operations trigger + Settings + Collapse/Close */}
      <div className="flex items-center gap-1">
        {onOpenCommandPost && (
          <CommandPostButton
            count={commandPostBadgeCount}
            onClick={onOpenCommandPost}
            className="md:hidden border-0 bg-transparent p-1.5"
          />
        )}
        <OpsTriggerButton variant="compact" />
        <button
          type="button"
          onClick={onSettingsOpen}
          className="rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Settings"
          data-testid="sidebar-settings"
        >
          <Settings className="h-4 w-4" />
        </button>
        <button
          type="button"
          onClick={onCollapse}
          className="rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Collapse sidebar"
          data-testid="sidebar-toggle-close"
        >
          <X className="h-4 w-4 md:hidden" />
          <PanelLeft className="hidden h-4 w-4 md:block" />
        </button>
      </div>
    </div>
  );
}
