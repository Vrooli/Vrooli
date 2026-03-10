/**
 * Briefs Page
 *
 * Displays daily briefs - morning and evening summaries across all domains.
 * This implements the daily brief system for consolidated domain insights.
 *
 * [REQ:LD-BRIEF-MORNING] - Morning brief with yesterday summary
 * [REQ:LD-BRIEF-EVENING] - Evening review with today's events
 * [REQ:LD-BRIEF-CONSOLIDATE] - Cross-domain consolidation
 */
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Sun, Moon, Calendar, RefreshCw } from "lucide-react";

import { BriefCard } from "../components/dashboard";
import { ErrorAlert } from "../components/ErrorAlert";
import { Card } from "../components/ui";
import { DATA_SELECTORS } from "../consts/selectors";

import {
  fetchCurrentBrief,
  fetchMorningBrief,
  fetchEveningBrief,
} from "../lib/api";

type BriefTab = "current" | "morning" | "evening";

export default function BriefsPage() {
  const [activeTab, setActiveTab] = useState<BriefTab>("current");
  const [selectedDate, setSelectedDate] = useState(() => {
    return new Date().toISOString().split("T")[0];
  });

  const currentQuery = useQuery({
    queryKey: ["brief", "current"],
    queryFn: fetchCurrentBrief,
    refetchInterval: 60000,
    enabled: activeTab === "current",
  });

  const morningQuery = useQuery({
    queryKey: ["brief", "morning", selectedDate],
    queryFn: () => fetchMorningBrief(selectedDate),
    enabled: activeTab === "morning",
  });

  const eveningQuery = useQuery({
    queryKey: ["brief", "evening", selectedDate],
    queryFn: () => fetchEveningBrief(selectedDate),
    enabled: activeTab === "evening",
  });

  const activeQuery = activeTab === "current"
    ? currentQuery
    : activeTab === "morning"
    ? morningQuery
    : eveningQuery;

  const handleRefresh = () => {
    activeQuery.refetch();
  };

  const tabs: { id: BriefTab; label: string; icon: typeof Sun }[] = [
    { id: "current", label: "Current", icon: Calendar },
    { id: "morning", label: "Morning", icon: Sun },
    { id: "evening", label: "Evening", icon: Moon },
  ];

  return (
    <div className="space-y-6" data-testid={DATA_SELECTORS.BRIEFS_PAGE}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Daily Briefs</h1>
          <p className="text-gray-400 mt-1">
            Morning and evening summaries across all your domains
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={activeQuery.isFetching}
          className="flex items-center gap-2 px-3 py-2 text-sm bg-white/10 hover:bg-white/15 rounded-lg transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${activeQuery.isFetching ? "animate-spin" : ""}`} />
          Refresh
        </button>
      </div>

      {/* Error state */}
      <ErrorAlert
        error={activeQuery.error as Error | null}
        onRetry={handleRefresh}
      />

      {/* Tab navigation */}
      <div className="flex gap-2">
        {tabs.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            data-testid={id === "morning" ? DATA_SELECTORS.BRIEF_MORNING_TAB : id === "evening" ? DATA_SELECTORS.BRIEF_EVENING_TAB : undefined}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
              activeTab === id
                ? "bg-blue-600 text-white"
                : "bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white"
            }`}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        ))}
      </div>

      {/* Date picker for morning/evening tabs */}
      {activeTab !== "current" && (
        <Card className="flex items-center gap-3 p-3">
          <Calendar className="w-4 h-4 text-gray-400" />
          <label className="text-sm text-gray-400">Select date:</label>
          <input
            type="date"
            value={selectedDate}
            onChange={(e) => {
              const dateValue = e.target.value;
              if (dateValue) {
                setSelectedDate(dateValue);
              }
            }}
            max={new Date().toISOString().split("T")[0]}
            className="bg-white/5 border border-white/10 rounded px-3 py-1 text-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </Card>
      )}

      {/* Brief content */}
      <BriefCard
        brief={activeQuery.data?.brief ?? null}
        isLoading={activeQuery.isLoading}
      />

      {/* Config info */}
      {activeQuery.data?.config && (
        <Card className="text-xs text-gray-500 flex items-center justify-center gap-4 p-3">
          <span className="flex items-center gap-1">
            <Sun className="w-3 h-3" />
            Morning: {activeQuery.data.config.morning_hour}:00
          </span>
          <span className="flex items-center gap-1">
            <Moon className="w-3 h-3" />
            Evening: {activeQuery.data.config.evening_hour}:00
          </span>
        </Card>
      )}
    </div>
  );
}
