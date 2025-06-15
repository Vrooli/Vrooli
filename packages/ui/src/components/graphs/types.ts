/* c8 ignore start */
export interface LineGraphProps {
    /** Dimensions of the line graph */
    dims: { width: number, height: number };
    /** Chronological list of data points to display in the line graph, optionally with labels */
    data: (number | { label: string, value: number })[];
    /** Color of the line */
    lineColor?: string;
    /** Width of the line */
    lineWidth?: number;
    /** Label for the y-axis of the line graph */
    yAxisLabel?: string;
    /** Whether to hide Y-axis labels */
    hideAxes?: boolean;
    /** Whether to hide tooltips */
    hideTooltips?: boolean;
    /** Color for data point dots */
    dotColor?: string;
}
