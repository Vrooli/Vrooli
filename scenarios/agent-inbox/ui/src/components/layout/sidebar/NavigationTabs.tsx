import { navItems } from "./types";
import type { View } from "./types";

interface NavigationTabsProps {
  currentView: View;
  onViewChange: (view: View) => void;
  chatCounts?: {
    inbox: number;
    starred: number;
    archived: number;
  };
  testId: string;
}

export function NavigationTabs({ currentView, onViewChange, chatCounts, testId }: NavigationTabsProps) {
  return (
    <div className="px-3 py-2 border-b border-white/10 shrink-0" data-testid={testId}>
      <div className="flex gap-1">
        {navItems.map(({ id, label, icon: Icon }) => {
          const count = chatCounts?.[id];
          const isActive = currentView === id;

          return (
            <button
              key={id}
              onClick={() => onViewChange(id)}
              className={`flex-1 flex items-center justify-center gap-1.5 px-2 py-2 rounded-lg text-xs font-medium transition-colors ${
                isActive
                  ? "bg-white/10 text-white"
                  : "text-slate-400 hover:text-white hover:bg-white/5"
              }`}
              data-testid={`nav-${id}`}
            >
              <Icon className={`h-3.5 w-3.5 ${isActive && id === "starred" ? "text-yellow-400" : ""}`} />
              <span className="hidden sm:inline">{label}</span>
              {count !== undefined && count > 0 && (
                <span
                  className={`text-[10px] px-1 py-0.5 rounded-full ${
                    isActive ? "bg-white/20" : "bg-white/10"
                  }`}
                >
                  {count}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
