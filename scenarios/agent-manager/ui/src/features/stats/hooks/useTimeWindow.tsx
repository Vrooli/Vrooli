// Time window store for stats - provides shared time filter state
// AI_CHECK: react_coherence=1 | LAST: 2026-02-06

import { useCallback, useEffect, useMemo, useSyncExternalStore } from "react";
import type { ReactNode } from "react";
import type { StatsFilter, TimePreset } from "../api/types";

interface TimeWindowStoreValue {
  preset: TimePreset;
  setPreset: (preset: TimePreset) => void;
  filter: StatsFilter;
  presetOptions: readonly TimePreset[];
}

const PRESET_OPTIONS: readonly TimePreset[] = ["6h", "12h", "24h", "7d", "30d"] as const;
const DEFAULT_PRESET: TimePreset = "24h";

let activePreset: TimePreset = DEFAULT_PRESET;
const listeners = new Set<() => void>();

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function getSnapshot(): TimePreset {
  return activePreset;
}

function setActivePreset(nextPreset: TimePreset): void {
  if (activePreset === nextPreset) return;
  activePreset = nextPreset;
  for (const listener of listeners) {
    listener();
  }
}

interface TimeWindowProviderProps {
  children: ReactNode;
  defaultPreset?: TimePreset;
}

export function TimeWindowProvider({
  children,
  defaultPreset = DEFAULT_PRESET,
}: TimeWindowProviderProps) {
  useEffect(() => {
    const previousPreset = activePreset;
    setActivePreset(defaultPreset);
    return () => {
      setActivePreset(previousPreset);
    };
  }, [defaultPreset]);

  return <>{children}</>;
}

export function useTimeWindow(): TimeWindowStoreValue {
  const preset = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
  const setPreset = useCallback((nextPreset: TimePreset) => {
    setActivePreset(nextPreset);
  }, []);
  const filter = useMemo<StatsFilter>(() => ({ preset }), [preset]);

  return useMemo(
    () => ({
      preset,
      setPreset,
      filter,
      presetOptions: PRESET_OPTIONS,
    }),
    [filter, preset, setPreset]
  );
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
