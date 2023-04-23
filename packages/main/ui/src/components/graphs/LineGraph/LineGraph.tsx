import { Tooltip } from "@mui/material";
import { scaleLinear } from "d3-scale";
import { curveMonotoneX, line } from "d3-shape";
import { useMemo, useState } from "react";
import { LineGraphProps } from "../types";

type Point = {
    x: number,
    y: number,
    label: string,
    value: number,
}

// Function to extract the label from a data point
const toLabel = (datum: any): string => {
    if (typeof datum === "object" && datum !== null && "label" in datum) {
        return datum.label;
    }
    return "";
};

// Function to extract the value from a data point
const toValue = (datum: any): number => {
    if (typeof datum === "number") {
        return datum;
    }
    if (typeof datum === "object" && datum !== null && "value" in datum) {
        return datum.value;
    }
    return 0;
};

/**
 * A line graph component to represent a list of numerical data as a line.
 */
export const LineGraph = ({
    dims,
    data,
    lineColor = "#000",
    lineWidth = 2,
    yAxisLabel,
}: LineGraphProps) => {
    // Find the maximum and minimum value in the data array
    const maxData = Math.max(...data.map(toValue));
    const minData = Math.min(...data.map(toValue));
    const constantValue = minData === maxData;

    // Calculate the x and y coordinates of each point in the data array
    const xScale = scaleLinear().domain([0, data.length - 1]).range([0, dims.width]);
    const yScale = scaleLinear().domain([constantValue ? minData - 0.5 : 0, maxData]).range([dims.height, 0]);

    const points: Point[] = data.map((datum, index) => ({
        x: xScale(index),
        y: yScale(toValue(datum)),
        label: toLabel(datum),
        value: toValue(datum),
    }));

    // Set up state for the selected point (i.e. hovered or pressed)
    const [selectedPoint, setSelectedPoint] = useState<Point | null>(null);

    // Generate the path string for the line element
    const d3Line = line<Point>()
        .x((point) => point.x)
        .y((point) => point.y)
        .curve(curveMonotoneX);

    const path = d3Line(points);

    // A tooltip is displayed when the user hovers over a data point
    const tooltipText = useMemo(() => {
        if (!selectedPoint) return "";
        return `${selectedPoint.label.length > 0 ? `${selectedPoint.label}: ` : ""} ${selectedPoint.value}`;
    }, [selectedPoint]);

    const renderGrid = () => {
        const numOfHorizontalLines = 5;
        const numOfVerticalLines = data.length > 1 ? data.length - 1 : 1;

        const horizontalLines = [...Array(numOfHorizontalLines)].map((_, index) => {
            const y = (index / (numOfHorizontalLines - 1)) * dims.height;
            return (
                <line
                    key={`h-line-${index}`}
                    x1={0}
                    y1={y}
                    x2={dims.width}
                    y2={y}
                    stroke="#ccc"
                    strokeWidth={1}
                />
            );
        });

        const verticalLines = [...Array(numOfVerticalLines)].map((_, index) => {
            const x = (index / (numOfVerticalLines - 1)) * dims.width;
            return (
                <line
                    key={`v-line-${index}`}
                    x1={x}
                    y1={0}
                    x2={x}
                    y2={dims.height}
                    stroke="#ccc"
                    strokeWidth={1}
                />
            );
        });

        return (
            <>
                {horizontalLines}
                {verticalLines}
            </>
        );
    };

    const renderDataPoints = () => {
        return points.map((point, index) => (
            <circle
                key={`point-${index}`}
                cx={point.x}
                cy={point.y}
                r={3}
                fill={selectedPoint === point ? lineColor : "#ccc"}
                stroke={lineColor}
                strokeWidth={1}
                onMouseEnter={() => setSelectedPoint(point)}
                onMouseLeave={() => setSelectedPoint(null)}
            />
        ));
    };

    const renderTooltip = () => {
        if (selectedPoint === null) {
            return null;
        }

        const tooltipWidth = 100;
        const tooltipHeight = 50;
        const tooltipX = Math.min(selectedPoint.x, dims.width - tooltipWidth);
        const tooltipY = Math.min(selectedPoint.y, dims.height - tooltipHeight);

        return (
            <g>
                <rect
                    x={tooltipX}
                    y={tooltipY}
                    width={tooltipWidth}
                    height={tooltipHeight}
                    fill="white"
                    stroke={lineColor}
                    strokeWidth={1}
                />
                <text
                    x={tooltipX + 5}
                    y={tooltipY + 20}
                    fontSize={14}
                    fill={lineColor}
                >
                    {selectedPoint.label}
                </text>
                <text
                    x={tooltipX + 5}
                    y={tooltipY + 35}
                    fontSize={12}
                    fill={lineColor}
                >
                    Value: {selectedPoint.value}
                </text>
            </g>
        );
    };

    return (
        <Tooltip title={tooltipText} placement="top" sx={{
            // Custom placement of the tooltip to align with the data point
            // '& .MuiTooltip-tooltip': {
            //     transform: `translateX(${selectedPoint ? selectedPoint.x - 10 : 0}px)`,
            // },
        }}>
            {/* The line graph */}
            <svg width={dims.width} height={dims.height}>
                {renderGrid()}
                <path
                    d={path}
                    stroke={lineColor}
                    strokeWidth={lineWidth}
                    fill="none"
                    onMouseMove={(e) => {
                        // Calculate the x and y coordinates of the mouse relative to the SVG element
                        const svgRect = e.currentTarget.getBoundingClientRect();
                        const x = e.clientX - svgRect.left;
                        const y = e.clientY - svgRect.top;
                        // Find the data point closest to the mouse cursor
                        const closest = points.reduce((acc, point) => {
                            const distance = Math.sqrt((point.x - x) ** 2 + (point.y - y) ** 2);
                            if (distance < acc.distance) {
                                return { point, distance };
                            }
                            return acc;
                        }, { point: null, distance: Number.MAX_VALUE } as { point: Point | null, distance: number });
                        if (closest.distance < 25) {
                            // If a data point is close enough to the mouse cursor, display a selectedPoint with its value
                            setSelectedPoint(closest.point);
                        } else {
                            // Otherwise, hide the selectedPoint
                            setSelectedPoint(null);
                        }
                    }}
                    onMouseLeave={() => {
                        // Hide the selectedPoint when the mouse leaves the SVG element
                        setSelectedPoint(null);
                    }}
                    onClick={(e) => {
                        // Calculate the x and y coordinates of the mouse relative to the SVG element
                        const svgRect = e.currentTarget.getBoundingClientRect();
                        const x = e.clientX - svgRect.left;
                        const y = e.clientY - svgRect.top;
                        // Find the data point closest to the mouse cursor
                        const closest = points.reduce((acc, point) => {
                            const distance = Math.sqrt((point.x - x) ** 2 + (point.y - y) ** 2);
                            if (distance < acc.distance) {
                                return { point, distance };
                            }
                            return acc;
                        }, { point: null, distance: Number.MAX_VALUE } as { point: Point | null, distance: number });
                        if (closest.distance < 25) {
                            // If a data point is close enough to the mouse cursor, display a selectedPoint with its value
                            setSelectedPoint(closest.point);
                        } else {
                            // Otherwise, hide the selectedPoint
                            setSelectedPoint(null);
                        }
                    }}
                />
                {renderDataPoints()}
                {renderTooltip()}
            </svg>
        </Tooltip>
    );
};
