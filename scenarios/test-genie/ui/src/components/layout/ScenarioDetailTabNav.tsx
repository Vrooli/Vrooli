import { cn } from "../../lib/utils";
import type { ScenarioDetailTabKey } from "../../types";

interface ScenarioDetailTabNavProps {
  activeTab: ScenarioDetailTabKey;
  onTabChange: (tab: ScenarioDetailTabKey) => void;
}

const TABS: Array<{ key: ScenarioDetailTabKey; label: string }> = [
  { key: "overview", label: "Overview" },
  { key: "requirements", label: "Requirements" },
  { key: "history", label: "History" }
];

export function ScenarioDetailTabNav({ activeTab, onTabChange }: ScenarioDetailTabNavProps) {
  return (
    <div className="flex border-b border-white/10">
      {TABS.map((tab) => (
        <button
          key={tab.key}
          type="button"
          onClick={() => onTabChange(tab.key)}
          className={cn(
            "relative px-4 py-3 text-sm font-medium transition",
            activeTab === tab.key
              ? "text-white"
              : "text-slate-400 hover:text-slate-200"
          )}
          data-testid={`test-genie-scenario-tab-${tab.key}`}
        >
          {tab.label}
          {activeTab === tab.key && (
            <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-gradient-to-r from-purple-500 to-cyan-500" />
          )}
        </button>
      ))}
    </div>
  );
}
