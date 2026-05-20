import { useCallback, useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { validateScenario, type ValidationResult } from "../../api/validation";

const RECENT_STORAGE_KEY = "ui-health.validation.recent.v1";
const RECENT_LIMIT = 25;

export type RecentRun = {
  scenario: string;
  passed: boolean;
  errors: number;
  warnings: number;
  infos: number;
  ranAt: string;
};

const validationKey = (scenario: string) => ["validation", "result", scenario] as const;

export function recentRunFromResult(result: ValidationResult): RecentRun {
  return {
    scenario: result.scenario,
    passed: result.passed,
    errors: result.summary.errors,
    warnings: result.summary.warnings,
    infos: result.summary.infos,
    ranAt: result.ranAt,
  };
}

function readRecent(): RecentRun[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(RECENT_STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isRecentRun);
  } catch {
    return [];
  }
}

function isRecentRun(value: unknown): value is RecentRun {
  if (!value || typeof value !== "object") return false;
  const v = value as Record<string, unknown>;
  return (
    typeof v.scenario === "string" &&
    typeof v.passed === "boolean" &&
    typeof v.errors === "number" &&
    typeof v.warnings === "number" &&
    typeof v.infos === "number" &&
    typeof v.ranAt === "string"
  );
}

function writeRecent(runs: RecentRun[]): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(RECENT_STORAGE_KEY, JSON.stringify(runs));
  } catch {
    // storage full or disabled — silently drop; recent runs are non-essential.
  }
}

export function upsertRecent(runs: RecentRun[], next: RecentRun): RecentRun[] {
  const withoutDup = runs.filter((r) => r.scenario !== next.scenario);
  return [next, ...withoutDup].slice(0, RECENT_LIMIT);
}

export function useRecentRuns(): {
  runs: RecentRun[];
  record: (result: ValidationResult) => void;
  clear: () => void;
} {
  const [runs, setRuns] = useState<RecentRun[]>(() => readRecent());

  useEffect(() => {
    if (typeof window === "undefined") return;
    const handler = (e: StorageEvent) => {
      if (e.key === RECENT_STORAGE_KEY) setRuns(readRecent());
    };
    window.addEventListener("storage", handler);
    return () => window.removeEventListener("storage", handler);
  }, []);

  const record = useCallback((result: ValidationResult) => {
    setRuns((prev) => {
      const next = upsertRecent(prev, recentRunFromResult(result));
      writeRecent(next);
      return next;
    });
  }, []);

  const clear = useCallback(() => {
    setRuns([]);
    writeRecent([]);
  }, []);

  return { runs, record, clear };
}

export function useValidateScenario() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (scenario: string) => validateScenario(scenario),
    onSuccess: (result) => {
      queryClient.setQueryData(validationKey(result.scenario), result);
    },
  });
}

export { validationKey };
