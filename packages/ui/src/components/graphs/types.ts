export interface Dimensions {
    width: number;
    height: number;
}

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
}
