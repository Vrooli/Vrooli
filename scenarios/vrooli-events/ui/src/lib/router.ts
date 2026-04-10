// DOC: docs/internal/ASSUMPTIONS.md — routing with react-router-dom
// DOC: docs/internal/EXPERIENCE-AUDIT.md#navigation-integrity
import type { LucideIcon } from "lucide-react";
import { Activity, BarChart3, Radio, Settings, GitBranch, Users, Shield, Zap, Bell, ClipboardCheck } from "lucide-react";

export type Route = "stream" | "analytics" | "events" | "settings" | "scenarios" | "traces" | "policies" | "circuit-breakers" | "subscriptions" | "compliance";

/** Navigation item definition — single source of truth for routes and their labels. */
export interface NavItem {
  id: Route;
  label: string;
  icon: LucideIcon;
}

/** All navigable routes with display metadata. */
export const NAV_ITEMS: NavItem[] = [
  { id: "stream", label: "Live Stream", icon: Radio },
  { id: "analytics", label: "Analytics", icon: BarChart3 },
  { id: "scenarios", label: "Scenario Metrics", icon: Users },
  { id: "traces", label: "Correlation Traces", icon: GitBranch },
  { id: "events", label: "Event History", icon: Activity },
  { id: "policies", label: "Policies", icon: Shield },
  { id: "circuit-breakers", label: "Circuit Breakers", icon: Zap },
  { id: "subscriptions", label: "Subscriptions", icon: Bell },
  { id: "compliance", label: "Compliance", icon: ClipboardCheck },
  { id: "settings", label: "Settings", icon: Settings },
];

/** All valid route identifiers — used for validation and scoring detection. */
export const ROUTES: Route[] = NAV_ITEMS.map((item) => item.id);
