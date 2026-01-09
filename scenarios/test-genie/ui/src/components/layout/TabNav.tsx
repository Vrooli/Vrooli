import { cn } from "../../lib/utils";
import { DASHBOARD_TABS } from "../../lib/constants";
import type { DashboardTabKey } from "../../types";
import { selectors } from "../../consts/selectors";

interface TabNavProps {
  activeTab: DashboardTabKey;
  onTabChange: (tab: DashboardTabKey) => void;
}

export function TabNav({ activeTab, onTabChange }: TabNavProps) {
  return (
    <nav className="flex flex-wrap gap-2" data-testid={selectors.tabs.nav}>
      {DASHBOARD_TABS.map((tab) => (
        <button
          key={tab.key}
          type="button"
          onClick={() => onTabChange(tab.key)}
          className={cn(
            "rounded-full border px-4 py-2 text-sm transition",
            activeTab === tab.key
              ? "border-cyan-400 bg-cyan-400/20 text-white"
              : "border-white/20 text-slate-300 hover:border-white/50"
          )}
          data-testid={selectors.tabs[tab.key]}
        >
          {tab.label}
        </button>
      ))}
    </nav>
  );
}
