import { isActiveAgentSession } from "../../../../stores";
import type { AgentSession } from "../../../../types";
import type { SessionFilters, SortConfig } from "./types";

export function applySessionFilters(items: AgentSession[], filters: SessionFilters): AgentSession[] {
  return items.filter((item) => {
    if (filters.statuses.length > 0 && !filters.statuses.includes(item.status)) return false;
    if (filters.kinds.length > 0 && !filters.kinds.includes(item.kind)) return false;
    if (filters.activeOnly && !isActiveAgentSession(item)) return false;
    if (filters.hasProposals && item.proposals.length === 0) return false;
    if (filters.hasAppliedArtifacts && !item.artifacts.some((artifact) => artifact.action !== "proposed")) return false;
    return true;
  });
}

export function applySessionSort(items: AgentSession[], sort: SortConfig): AgentSession[] {
  const sorted = [...items];
  const dir = sort.direction === "asc" ? 1 : -1;

  sorted.sort((a, b) => {
    switch (sort.field) {
      case "status":
        return a.status.localeCompare(b.status) * dir;
      case "alphabetical":
        return a.title.localeCompare(b.title) * dir;
      case "recency":
      default:
        return (new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime()) * dir;
    }
  });

  return sorted;
}
