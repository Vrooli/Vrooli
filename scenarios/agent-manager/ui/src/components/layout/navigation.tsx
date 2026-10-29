import type { ReactNode } from "react";
import { Activity, Binoculars, ClipboardList, Gauge, GitBranch, HeartPulse, Lightbulb, Play, Search, Settings2, Target } from "lucide-react";

export type NavSection = "dashboard" | "tasks" | "runs" | "workflows" | "watches" | "efforts" | "usage" | "findings" | "investigations" | "health" | "profiles";

export interface NavDestination {
  id: NavSection;
  label: string;
  /** Label under the icon on the phone tab bar. */
  shortLabel?: string;
  href: string;
  icon: ReactNode;
  /** Which part of the job the destination serves; the desktop column is ordered by group. */
  group: "work" | "insight" | "system";
  /** Shown on the phone tab bar. Every destination stays reachable from the header menu. */
  phoneTab: boolean;
}

/**
 * One navigation model for both media. Desktop lists every destination in group
 * order (work → insight → system). The phone tab bar carries at most
 * {@link MAX_PHONE_TABS} of them; the shell's header menu lists all.
 */
export const NAV_DESTINATIONS: readonly NavDestination[] = [
  { id: "dashboard", label: "Dashboard", shortLabel: "Home", href: "/", icon: <Activity aria-hidden="true" />, group: "work", phoneTab: true },
  { id: "tasks", label: "Tasks", href: "/tasks", icon: <ClipboardList aria-hidden="true" />, group: "work", phoneTab: true },
  { id: "runs", label: "Runs", href: "/runs", icon: <Play aria-hidden="true" />, group: "work", phoneTab: true },
  { id: "workflows", label: "Flows", href: "/workflows", icon: <GitBranch aria-hidden="true" />, group: "work", phoneTab: false },
  { id: "watches", label: "Watches", href: "/watches", icon: <Binoculars aria-hidden="true" />, group: "work", phoneTab: false },
  { id: "efforts", label: "Efforts", href: "/efforts", icon: <Target aria-hidden="true" />, group: "work", phoneTab: false },
  { id: "usage", label: "Usage", href: "/usage", icon: <Gauge aria-hidden="true" />, group: "insight", phoneTab: true },
  { id: "findings", label: "Findings", href: "/findings", icon: <Lightbulb aria-hidden="true" />, group: "insight", phoneTab: true },
  { id: "investigations", label: "Investigations", href: "/investigations", icon: <Search aria-hidden="true" />, group: "insight", phoneTab: false },
  // The api-base UI server reserves `/health` for its status JSON, so the page lives at `/observability`.
  { id: "health", label: "Health", href: "/observability", icon: <HeartPulse aria-hidden="true" />, group: "system", phoneTab: false },
  { id: "profiles", label: "Profiles", href: "/profiles", icon: <Settings2 aria-hidden="true" />, group: "system", phoneTab: false },
];

export const MAX_PHONE_TABS = 5;

/** Maps a location to the destination it belongs to; unknown paths fall back to the dashboard. */
export function sectionForPath(path: string): NavSection {
  const match = NAV_DESTINATIONS.find((item) => item.href !== "/" && (path === item.href || path.startsWith(`${item.href}/`)));
  return match?.id ?? "dashboard";
}
