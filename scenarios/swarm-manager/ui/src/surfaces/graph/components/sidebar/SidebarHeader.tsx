/**
 * SidebarHeader — Top bar for the graph sidebar.
 *
 * Shows the app title, a home button (returns to graph view), a settings
 * gear, and a collapse/close button. The home button carries a bot-count
 * badge (from `useOperationsStore`) when agents are running — the Plan board
 * behind it is where running agents are shown, so the badge doubles as the
 * "something is running" indicator without a dedicated agents chip.
 */

import { Bot, Home, PanelLeft, Settings, X } from "lucide-react";
import { DecisionsInboxButton } from "../../../../components/command-post/DecisionsInboxButton";
import { selectActiveCount, useOperationsStore } from "../../../../stores/operations-store";

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
  const activeAgentCount = useOperationsStore(selectActiveCount);

  return (
    <div className="flex h-10 shrink-0 items-center justify-between border-b border-slate-200/20 px-3">
      {/* Left: Home button + App title */}
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          onClick={onGoHome}
          className="relative rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label={
            activeAgentCount > 0
              ? `Go to graph view (${activeAgentCount} ${activeAgentCount === 1 ? "agent" : "agents"} running)`
              : "Go to graph view"
          }
          data-testid="sidebar-home"
        >
          <Home className="h-4 w-4" />
          {activeAgentCount > 0 && (
            <span
              className="absolute -right-1.5 -top-1 inline-flex items-center gap-0.5 rounded-full bg-emerald-500/20 px-1 py-px text-[10px] font-semibold leading-none text-emerald-300"
              data-testid="sidebar-home-agent-badge"
            >
              <Bot className="h-2.5 w-2.5" aria-hidden />
              {activeAgentCount}
            </span>
          )}
        </button>
        <span className="text-sm font-semibold text-slate-200">Swarm Manager</span>
      </div>

      {/* Right: decisions inbox + Settings + Collapse/Close */}
      <div className="flex items-center gap-1">
        {onOpenCommandPost && (
          <DecisionsInboxButton onOpen={onOpenCommandPost} testId="sidebar-decisions-button" />
        )}
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
