import { Settings, Keyboard } from "lucide-react";
import { Tooltip } from "../../ui/tooltip";
import { Badge } from "../../ui/badge";
import type { Label } from "./types";

interface SidebarFooterProps {
  labels: Label[];
  onManageLabels: () => void;
  onShowKeyboardShortcuts: () => void;
  onOpenSettings: () => void;
  manageLabelsTestId: string;
}

export function SidebarFooter({
  labels,
  onManageLabels,
  onShowKeyboardShortcuts,
  onOpenSettings,
  manageLabelsTestId,
}: SidebarFooterProps) {
  return (
    <>
      {/* Labels Section (collapsed into a row) */}
      {labels.length > 0 && (
        <div className="px-3 py-2 border-t border-white/10 shrink-0">
          <div className="flex items-center gap-2 overflow-x-auto scrollbar-hide">
            <span className="text-[10px] text-slate-400 uppercase tracking-wide shrink-0">Labels:</span>
            {labels.slice(0, 4).map((label) => (
              <Badge
                key={label.id}
                color={label.color}
                className="text-[10px] py-0.5 shrink-0 cursor-pointer hover:opacity-80"
                onClick={onManageLabels}
              >
                {label.name}
              </Badge>
            ))}
            {labels.length > 4 && (
              <button
                onClick={onManageLabels}
                className="text-[10px] text-slate-400 hover:text-white shrink-0"
              >
                +{labels.length - 4}
              </button>
            )}
          </div>
        </div>
      )}

      {/* Footer */}
      <div className="p-3 border-t border-white/10 shrink-0">
        <div className="hidden lg:flex items-center justify-center gap-1">
          <Tooltip content="Manage labels">
            <button
              onClick={onManageLabels}
              className="p-2 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors"
              data-testid={manageLabelsTestId}
            >
              <Badge color="#6366f1" className="h-4 w-4 p-0 flex items-center justify-center text-[8px]">
                {labels.length}
              </Badge>
            </button>
          </Tooltip>
          <Tooltip content="Keyboard shortcuts (?)">
            <button
              onClick={onShowKeyboardShortcuts}
              className="p-2 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors"
              data-testid="sidebar-shortcuts-button"
              aria-label="Open keyboard shortcuts"
            >
              <Keyboard className="h-4 w-4" />
            </button>
          </Tooltip>
          <Tooltip content="Settings">
            <button
              onClick={onOpenSettings}
              className="p-2 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors"
              data-testid="sidebar-settings-button"
              aria-label="Open settings"
            >
              <Settings className="h-4 w-4" />
            </button>
          </Tooltip>
        </div>
        <p className="text-[11px] text-slate-500 text-center lg:hidden">
          Use the top-right menu for labels, shortcuts, and settings.
        </p>
      </div>
    </>
  );
}
