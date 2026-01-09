// Time window context for stats - provides shared time filter state

import { createContext, useContext, useState, useCallback, useMemo } from "react";
import type { ReactNode } from "react";
import type { StatsFilter, TimePreset } from "../api/types";

interface TimeWindowContextValue {
  preset: TimePreset;
  setPreset: (preset: TimePreset) => void;
  filter: StatsFilter;
  presetOptions: readonly TimePreset[];
}

const TimeWindowContext = createContext<TimeWindowContextValue | null>(null);

const PRESET_OPTIONS: readonly TimePreset[] = ["6h", "12h", "24h", "7d", "30d"] as const;

interface TimeWindowProviderProps {
  children: ReactNode;
  defaultPreset?: TimePreset;
}

export function TimeWindowProvider({
  children,
  defaultPreset = "24h",
}: TimeWindowProviderProps) {
  const [preset, setPreset] = useState<TimePreset>(defaultPreset);

  const filter = useMemo<StatsFilter>(() => ({ preset }), [preset]);

  const value = useMemo(
    () => ({
      preset,
      setPreset,
      filter,
      presetOptions: PRESET_OPTIONS,
    }),
    [preset, filter]
  );

  return (
    <TimeWindowContext.Provider value={value}>
      {children}
    </TimeWindowContext.Provider>
  );
}

export function useTimeWindow(): TimeWindowContextValue {
  const context = useContext(TimeWindowContext);
  if (!context) {
    throw new Error("useTimeWindow must be used within a TimeWindowProvider");
  }
  return context;
}

// Hook for components that need to set custom filters (e.g., with profile/runner filtering)
export function useStatsFilter(overrides?: Partial<StatsFilter>): StatsFilter {
  const { filter } = useTimeWindow();
  return useMemo(() => ({ ...filter, ...overrides }), [filter, overrides]);
}

// Utility to get human-readable label for preset
export function getPresetLabel(preset: TimePreset): string {
  switch (preset) {
    case "6h":
      return "Last 6 hours";
    case "12h":
      return "Last 12 hours";
    case "24h":
      return "Last 24 hours";
    case "7d":
      return "Last 7 days";
    case "30d":
      return "Last 30 days";
    default:
      return preset;
  }
}

// Utility to get short label for preset
export function getPresetShortLabel(preset: TimePreset): string {
  switch (preset) {
    case "6h":
      return "6h";
    case "12h":
      return "12h";
    case "24h":
      return "24h";
    case "7d":
      return "7d";
    case "30d":
      return "30d";
    default:
      return preset;
  }
}
