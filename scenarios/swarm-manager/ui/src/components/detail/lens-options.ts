import { Crosshair, KanbanSquare, type LucideIcon } from "lucide-react";
import type { AppGraphLens } from "../../app/routes/route-paths";

export interface LensOption {
  lens: AppGraphLens;
  label: string;
  icon: LucideIcon;
  iconColorClass: string;
}

const PLAN_FOCUS_LENSES: LensOption[] = [
  { lens: "plan", label: "Plan", icon: KanbanSquare, iconColorClass: "text-cyan-400" },
  { lens: "focus", label: "Focus", icon: Crosshair, iconColorClass: "text-emerald-400" },
];

export const BACKLOG_LENSES: LensOption[] = PLAN_FOCUS_LENSES;
export const INITIATIVE_LENSES: LensOption[] = PLAN_FOCUS_LENSES;
export const EXECUTION_LENSES: LensOption[] = PLAN_FOCUS_LENSES;
export const SCENARIO_LENSES: LensOption[] = PLAN_FOCUS_LENSES;
