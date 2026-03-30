import { Activity, History, Network, type LucideIcon } from "lucide-react";
import type { GraphLens } from "../../surfaces/graph/stores/graph-data-store";

export interface LensOption {
  lens: GraphLens;
  label: string;
  icon: LucideIcon;
  iconColorClass: string;
}

export const BACKLOG_LENSES: LensOption[] = [
  { lens: "topology", label: "Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "flow", label: "History", icon: History, iconColorClass: "text-cyan-400" },
  { lens: "operations", label: "Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

export const INITIATIVE_LENSES: LensOption[] = [
  { lens: "topology", label: "Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "flow", label: "History", icon: History, iconColorClass: "text-cyan-400" },
  { lens: "operations", label: "Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

export const EXECUTION_LENSES: LensOption[] = [
  { lens: "flow", label: "History", icon: History, iconColorClass: "text-cyan-400" },
  { lens: "operations", label: "Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

export const SCENARIO_LENSES: LensOption[] = [
  { lens: "topology", label: "Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "operations", label: "Operations", icon: Activity, iconColorClass: "text-amber-400" },
];
