import { FileText, FileDiff, GitCommit, History } from "lucide-react";
import type { LayoutSection } from "./LayoutSettingsModal";

interface MobileNavProps {
  activePanel: LayoutSection;
  onPanelChange: (panel: LayoutSection) => void;
  stagedCount?: number;
  unstagedCount?: number;
}

const panels: Array<{
  id: LayoutSection;
  label: string;
  icon: typeof FileText;
}> = [
  { id: "changes", label: "Changes", icon: FileText },
  { id: "diff", label: "Diff", icon: FileDiff },
  { id: "commit", label: "Commit", icon: GitCommit },
  { id: "history", label: "History", icon: History }
];

export function MobileNav({
  activePanel,
  onPanelChange,
  stagedCount = 0,
  unstagedCount = 0
}: MobileNavProps) {
  const totalChanges = stagedCount + unstagedCount;

  return (
    <nav
      className="fixed bottom-0 left-0 right-0 z-40 border-t border-slate-800 bg-slate-950/95 backdrop-blur-sm pb-safe"
      data-testid="mobile-nav"
    >
      <div className="flex items-center justify-around h-16">
        {panels.map((panel) => {
          const isActive = activePanel === panel.id;
          const Icon = panel.icon;
          const showBadge = panel.id === "changes" && totalChanges > 0;
          const showCommitBadge = panel.id === "commit" && stagedCount > 0;

          return (
            <button
              key={panel.id}
              type="button"
              onClick={() => onPanelChange(panel.id)}
              className={`relative flex flex-col items-center justify-center gap-1 px-4 py-2 min-w-[64px] transition-colors touch-target ${
                isActive
                  ? "text-blue-400"
                  : "text-slate-500 hover:text-slate-300 active:text-slate-200"
              }`}
              aria-label={panel.label}
              aria-current={isActive ? "page" : undefined}
              data-testid={`mobile-nav-${panel.id}`}
            >
              <div className="relative">
                <Icon className="h-6 w-6" />
                {showBadge && (
                  <span className="absolute -top-1.5 -right-2 min-w-[18px] h-[18px] flex items-center justify-center rounded-full bg-amber-500 text-[10px] font-bold text-slate-900 px-1">
                    {totalChanges > 99 ? "99+" : totalChanges}
                  </span>
                )}
                {showCommitBadge && (
                  <span className="absolute -top-1.5 -right-2 min-w-[18px] h-[18px] flex items-center justify-center rounded-full bg-emerald-500 text-[10px] font-bold text-slate-900 px-1">
                    {stagedCount > 99 ? "99+" : stagedCount}
                  </span>
                )}
              </div>
              <span className="text-[10px] font-medium">{panel.label}</span>
              {isActive && (
                <span
                  className="absolute bottom-0 left-1/2 -translate-x-1/2 w-8 h-0.5 rounded-full bg-blue-400"
                  aria-hidden="true"
                />
              )}
            </button>
          );
        })}
      </div>
    </nav>
  );
}
