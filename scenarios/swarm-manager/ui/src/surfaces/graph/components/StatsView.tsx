import { useState } from "react";
import { useStats } from "../../../hooks/useStats";
import type { StatsCategory } from "../../../types/stats";
import { StatsContent } from "./StatsPanel";

export function StatsView() {
  const [activeTab, setActiveTab] = useState<StatsCategory>("dashboard");
  const { data, isLoading, error } = useStats(true);

  return (
    <div className="h-full overflow-y-auto bg-slate-950" data-testid="stats-view">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-5 px-4 py-5 sm:px-6 lg:px-8">
        <div>
          <h1 className="text-xl font-semibold text-slate-100">Stats</h1>
          <p className="mt-1 text-sm text-slate-400">Operational metrics across backlog flow, agent work, modes, sessions, and blocking.</p>
        </div>
        <StatsContent
          data={data}
          isLoading={isLoading}
          error={error}
          activeTab={activeTab}
          onTabChange={setActiveTab}
        />
      </div>
    </div>
  );
}
