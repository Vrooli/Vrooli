import { strings } from "../../consts/strings";
import { encodeScenarioPath } from "../../hooks/useScenarioPath";

export type WorkspaceSubNavKey = "graph" | "domains" | "conflicts" | "campaign" | "apply" | "analytics";

export interface WorkspaceSubNavItem {
  readonly key: WorkspaceSubNavKey;
  readonly labelKey: (typeof strings.layout.subnav)[Exclude<keyof typeof strings.layout.subnav, "label">];
  readonly available: boolean;
  readonly subPath: string;
}

export const WORKSPACE_SUBNAV: readonly WorkspaceSubNavItem[] = [
  { key: "graph", labelKey: strings.layout.subnav.graph, available: true, subPath: "graph" },
  { key: "domains", labelKey: strings.layout.subnav.domains, available: true, subPath: "domains" },
  { key: "conflicts", labelKey: strings.layout.subnav.conflicts, available: true, subPath: "conflicts" },
  { key: "campaign", labelKey: strings.layout.subnav.campaign, available: true, subPath: "campaign" },
  { key: "apply", labelKey: strings.layout.subnav.apply, available: true, subPath: "apply" },
  { key: "analytics", labelKey: strings.layout.subnav.analytics, available: true, subPath: "analytics" },
];

export function buildWorkspaceSubPath(scenario: string, item: WorkspaceSubNavItem): string {
  return `/targets/${encodeScenarioPath(scenario)}/${item.subPath}`;
}
