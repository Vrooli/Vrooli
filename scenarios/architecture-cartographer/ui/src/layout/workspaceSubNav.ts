import { strings } from "../consts/strings";
import { encodeScenarioPath } from "../hooks/useScenarioPath";

/**
 * Per-target sub-nav entries. Mirrors `NAV_ITEMS` shape: `key` is the
 * selector parameter (stable across locales); `path` is built lazily so we
 * can encode the scenario at call time without leaking that responsibility
 * into the consumer.
 *
 * Sections that haven't shipped yet (Graph, Domains, Apply, Analytics)
 * carry `available: false` so the sub-nav renders them as disabled chips
 * with no router target. They flip to `available: true` in their phase.
 */
export type WorkspaceSubNavKey = "graph" | "domains" | "conflicts" | "campaign" | "apply" | "analytics";

export interface WorkspaceSubNavItem {
  readonly key: WorkspaceSubNavKey;
  readonly labelKey: (typeof strings.layout.subnav)[Exclude<keyof typeof strings.layout.subnav, "label">];
  /** When false, the item renders as a disabled chip (no link). */
  readonly available: boolean;
  /**
   * Sub-path relative to the workspace root (`/targets/:encodedPath/`).
   * Empty string for the workspace landing page itself (not used today).
   */
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

/** Build the route path for a sub-nav item, encoding the scenario for the URL. */
export function buildWorkspaceSubPath(scenario: string, item: WorkspaceSubNavItem): string {
  return `/targets/${encodeScenarioPath(scenario)}/${item.subPath}`;
}
