// Request processing helpers for Test Genie UI

import type { SuiteRequest } from "./api";
import { parseTimestamp } from "./formatters";

export function priorityWeight(priority: string): number {
  const normalized = priority.toLowerCase();
  if (normalized === "urgent") return 3;
  if (normalized === "high") return 2;
  if (normalized === "normal") return 1;
  return 0;
}

export function presetFromPriority(priority: string): string {
  const normalized = priority.toLowerCase();
  if (normalized === "urgent") return "comprehensive";
  if (normalized === "high") return "smoke";
  return "quick";
}

export function isActionableRequest(status: string): boolean {
  const normalized = status.toLowerCase();
  return normalized === "queued" || normalized === "delegated";
}

export function selectActionableRequest(requests: SuiteRequest[]): SuiteRequest | null {
  const actionable = requests.filter((req) => isActionableRequest(req.status));
  if (actionable.length === 0) {
    return null;
  }
  let best = actionable[0];
  for (let idx = 1; idx < actionable.length; idx += 1) {
    const candidate = actionable[idx];
    const priorityDiff = priorityWeight(candidate.priority) - priorityWeight(best.priority);
    if (priorityDiff > 0) {
      best = candidate;
      continue;
    }
    if (priorityDiff < 0) {
      continue;
    }
    const candidateTimestamp = parseTimestamp(candidate.updatedAt ?? candidate.createdAt);
    const bestTimestamp = parseTimestamp(best.updatedAt ?? best.createdAt);
    if (candidateTimestamp < bestTimestamp) {
      best = candidate;
    }
  }
  return best;
}
