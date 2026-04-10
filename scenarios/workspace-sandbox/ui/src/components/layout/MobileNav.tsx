import { Box, Info, FileDiff } from "lucide-react";
import { SELECTORS } from "../../consts/selectors";

export type MobilePanel = "sandboxes" | "details" | "changes";

interface MobileNavProps {
  activePanel: MobilePanel;
  onPanelChange: (panel: MobilePanel) => void;
  changeCount?: number;
}

const navItems: Array<{
  id: MobilePanel;
  label: string;
  icon: typeof Box;
}> = [
  { id: "sandboxes", label: "Sandboxes", icon: Box },
  { id: "details", label: "Details", icon: Info },
  { id: "changes", label: "Changes", icon: FileDiff },
];

export function MobileNav({ activePanel, onPanelChange, changeCount }: MobileNavProps) {
  return (
    <nav
      className="fixed bottom-0 left-0 right-0 z-40 border-t border-slate-800 bg-slate-950/95 backdrop-blur-sm pb-safe"
      data-testid={SELECTORS.mobileNav}
    >
      <div className="flex items-center justify-around h-16">
        {navItems.map((item) => {
          const isActive = activePanel === item.id;
          const Icon = item.icon;

          return (
            <button
              key={item.id}
              type="button"
              onClick={() => onPanelChange(item.id)}
              className={`relative flex flex-col items-center justify-center gap-1 px-2 py-2 min-w-[48px] touch-target transition-colors ${
                isActive
                  ? "text-emerald-400"
                  : "text-slate-500 hover:text-slate-300 active:text-slate-300"
              }`}
              aria-label={item.label}
              aria-current={isActive ? "page" : undefined}
              data-testid={SELECTORS.mobileNavTab(item.id)}
            >
              <div className="relative">
                <Icon className="h-5 w-5" />
                {item.id === "changes" && changeCount !== undefined && changeCount > 0 && (
                  <span className="absolute -top-1.5 -right-2.5 min-w-[16px] h-4 px-1 flex items-center justify-center rounded-full bg-emerald-600 text-[10px] font-bold text-white whitespace-nowrap overflow-hidden">
                    {changeCount > 99 ? "99+" : changeCount}
                  </span>
                )}
              </div>
              <span className="text-[10px] font-medium">{item.label}</span>
              {isActive && (
                <span
                  className="absolute bottom-0 left-1/2 -translate-x-1/2 w-8 h-0.5 rounded-full bg-emerald-400"
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
