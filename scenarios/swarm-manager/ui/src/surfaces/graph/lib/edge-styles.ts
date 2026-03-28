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

/** Threshold above which edges switch to straight lines for performance. */
export const STRAIGHT_EDGE_THRESHOLD = 300;

/** Threshold above which a filter suggestion banner is shown. */
export const FILTER_SUGGESTION_THRESHOLD = 500;
