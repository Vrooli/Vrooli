/**
 * Edge Styles
 *
 * Visual differentiation for topology edge types via color + dash pattern.
 * No labels — a compact legend explains the mapping.
 */

export interface EdgeStyleConfig {
  stroke: string;
  strokeDasharray: string;
  label: string;
}

export const EDGE_STYLES: Record<string, EdgeStyleConfig> = {
  depends_on: {
    stroke: "rgb(148 163 184)",    // slate-400
    strokeDasharray: "none",
    label: "Depends on",
  },
  member_of: {
    stroke: "rgb(56 189 248)",     // sky-400
    strokeDasharray: "6 3",
    label: "Member of",
  },
  classified_as: {
    stroke: "rgb(52 211 153)",     // emerald-400
    strokeDasharray: "2 3",
    label: "Classified as",
  },
  targets: {
    stroke: "rgb(167 139 250)",    // violet-400
    strokeDasharray: "none",
    label: "Targets",
  },
  executes: {
    stroke: "rgb(251 191 36)",     // amber-400
    strokeDasharray: "none",
    label: "Executes",
  },
  follow_up: {
    stroke: "rgb(251 146 60)",     // orange-400
    strokeDasharray: "5 3",
    label: "Follow-up",
  },
  activity_for: {
    stroke: "rgb(45 212 191)",     // teal-400
    strokeDasharray: "7 3",
    label: "Activity for",
  },
  records_activity: {
    stroke: "rgb(192 132 252)",    // purple-400
    strokeDasharray: "none",
    label: "Records activity",
  },
  spawned_run: {
    stroke: "rgb(244 114 182)",    // pink-400
    strokeDasharray: "3 2",
    label: "Spawned run",
  },
  continued_run: {
    stroke: "rgb(244 114 182)",    // pink-400
    strokeDasharray: "1 3",
    label: "Continued run",
  },
};

const DEFAULT_EDGE_STYLE: EdgeStyleConfig = {
  stroke: "rgb(100 116 139 / 0.5)",
  strokeDasharray: "none",
  label: "default",
};

/**
 * Get the React Flow edge style object for a given edge type.
 */
export function getEdgeStyle(edgeType: string | undefined): React.CSSProperties {
  const config = (edgeType && EDGE_STYLES[edgeType]) || DEFAULT_EDGE_STYLE;
  return {
    stroke: config.stroke,
    strokeDasharray: config.strokeDasharray === "none" ? undefined : config.strokeDasharray,
    strokeWidth: 1.5,
  };
}

export const SECONDARY_EDGE_TYPES = new Set<string>([
  "classified_as",
  "targets",
  "follow_up",
  "records_activity",
  "continued_run",
]);

/** Threshold above which edges switch to straight lines for performance. */
export const STRAIGHT_EDGE_THRESHOLD = 300;

/** Threshold above which a filter suggestion banner is shown. */
export const FILTER_SUGGESTION_THRESHOLD = 500;
