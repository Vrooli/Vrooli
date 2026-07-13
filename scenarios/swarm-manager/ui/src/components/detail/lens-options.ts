import { KanbanSquare, Network, type LucideIcon } from "lucide-react";
import type { AppGraphLens } from "../../app/routes/route-paths";

export interface LensOption {
  lens: AppGraphLens;
  label: string;
  icon: LucideIcon;
  iconColorClass: string;
}

const PLAN_LENS: LensOption = { lens: "plan", label: "Plan", icon: KanbanSquare, iconColorClass: "text-cyan-400" };

// "Graph" drills into the graph's focus mode (centered on this entity), but
// the label names the destination surface, matching the workspace lens nav.
const GRAPH_LENS: LensOption = { lens: "focus", label: "Graph", icon: Network, iconColorClass: "text-emerald-400" };

const PLAN_GRAPH_LENSES: LensOption[] = [PLAN_LENS, GRAPH_LENS];

// Goals are a board-scoping concept, not a graph filter: the Plan pill scopes
// the board to the goal (?goal=), and there is no Graph pill.
const PLAN_ONLY_LENSES: LensOption[] = [PLAN_LENS];

export const BACKLOG_LENSES: LensOption[] = PLAN_GRAPH_LENSES;
export const INITIATIVE_LENSES: LensOption[] = PLAN_GRAPH_LENSES;
export const GOAL_LENSES: LensOption[] = PLAN_ONLY_LENSES;
export const EXECUTION_LENSES: LensOption[] = PLAN_GRAPH_LENSES;
export const SCENARIO_LENSES: LensOption[] = PLAN_GRAPH_LENSES;
