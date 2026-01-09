// Chart configuration and theme constants

// Color scheme consistent with vrooli-autoheal (Tailwind colors)
export const CHART_COLORS = {
  // Status colors
  success: "#10b981", // emerald-500
  warning: "#f59e0b", // amber-500
  error: "#ef4444", // red-500
  info: "#3b82f6", // blue-500
  muted: "#6b7280", // gray-500

  // Run status colors
  complete: "#10b981", // emerald-500
  failed: "#ef4444", // red-500
  running: "#3b82f6", // blue-500
  pending: "#f59e0b", // amber-500
  cancelled: "#6b7280", // gray-500
  needsReview: "#8b5cf6", // violet-500

  // Chart series colors (for multi-line charts)
  series: [
    "#3b82f6", // blue-500
    "#10b981", // emerald-500
    "#f59e0b", // amber-500
    "#ef4444", // red-500
    "#8b5cf6", // violet-500
    "#ec4899", // pink-500
    "#06b6d4", // cyan-500
    "#84cc16", // lime-500
  ],

  // Grid and axis
  grid: "rgba(107, 114, 128, 0.3)", // 30% opacity gray-500
  axis: "#9ca3af", // gray-400
  text: "#d1d5db", // gray-300
} as const;

// Default chart margins
export const CHART_MARGINS = {
  top: 10,
  right: 10,
  bottom: 30,
  left: 40,
} as const;

// Default tooltip styles
export const TOOLTIP_STYLE = {
  backgroundColor: "hsl(var(--card))",
  border: "1px solid hsl(var(--border))",
  borderRadius: "6px",
  color: "hsl(var(--foreground))",
  fontSize: "12px",
  padding: "8px 12px",
} as const;

// Gradient definitions for area charts
export const AREA_GRADIENTS = {
  success: {
    id: "gradient-success",
    startOpacity: 0.8,
    endOpacity: 0.1,
    color: CHART_COLORS.success,
  },
  error: {
    id: "gradient-error",
    startOpacity: 0.8,
    endOpacity: 0.1,
    color: CHART_COLORS.error,
  },
  info: {
    id: "gradient-info",
    startOpacity: 0.8,
    endOpacity: 0.1,
    color: CHART_COLORS.info,
  },
  warning: {
    id: "gradient-warning",
    startOpacity: 0.8,
    endOpacity: 0.1,
    color: CHART_COLORS.warning,
  },
} as const;

// Helper to get status color
export function getStatusColor(status: string): string {
  switch (status.toLowerCase()) {
    case "complete":
    case "success":
      return CHART_COLORS.complete;
    case "failed":
    case "error":
      return CHART_COLORS.failed;
    case "running":
      return CHART_COLORS.running;
    case "pending":
      return CHART_COLORS.pending;
    case "cancelled":
      return CHART_COLORS.cancelled;
    case "needs_review":
    case "needsreview":
      return CHART_COLORS.needsReview;
    default:
      return CHART_COLORS.muted;
  }
}

// Helper to get nth series color
export function getSeriesColor(index: number): string {
  return CHART_COLORS.series[index % CHART_COLORS.series.length];
}
