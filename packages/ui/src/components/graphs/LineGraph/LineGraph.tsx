import { Box, Tooltip, Typography } from "@mui/material";
import { useMemo, useState } from "react";
import { LineGraphProps } from "../types";

type Point = {
    x: number,
    y: number,
    label: string,
    value: number,
}

const toLabel = (datum: number | { label: string, value: number }) => {
    return typeof datum === 'number' ? '' : datum.label;
};

const toValue = (datum: number | { label: string, value: number }) => {
    return typeof datum === 'number' ? datum : datum.value;
};

/**
 * A line graph component to represent a list of numerical data as a line.
 */
export const LineGraph = ({
    dims,
    data,
    lineColor = '#000',
    lineWidth = 2,
    yAxisLabel,
}: LineGraphProps) => {
    // Find the maximum value in the data array
    const maxData = Math.max(...data.map(toValue));

    // Calculate the x and y coordinates of each point in the data array
    const points: Point[] = data.map((datum, index) => ({
        x: (index / (data.length - 1)) * dims.width,
        y: maxData > 0 ? (dims.height - (toValue(datum) / maxData) * dims.height) : dims.height / 2,
        label: toLabel(datum),
        value: toValue(datum),
    }));

    // Set up state for the selected point (i.e. hovered or pressed)
    const [selectedPoint, setSelectedPoint] = useState<Point | null>(null);

    // Generate the path string for the line element
    const path = points.reduce((acc, point, index) => {
        const prefix = index === 0 ? 'M' : 'L';
        return `${acc} ${prefix}${point.x},${point.y}`;
    }, '');

    // A tooltip is displayed when the user hovers over a data point
    const tooltipText = useMemo(() => {
        if (!selectedPoint) return '';
        return `${selectedPoint.label.length > 0 ? `${selectedPoint.label}: ` : ''} ${selectedPoint.value}`;
    }, [selectedPoint]);

    return (
        <Tooltip title={tooltipText} placement="top" sx={{
            // Custom placement of the tooltip to align with the data point
            // '& .MuiTooltip-tooltip': {
            //     transform: `translateX(${selectedPoint ? selectedPoint.x - 10 : 0}px)`,
            // },
        }}>
            {/* The line graph */}
            <svg width={dims.width} height={dims.height}>
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
                        console.log('closest', closest.distance, closest.point)
                        if (closest.distance < 25) {
                            // If a data point is close enough to the mouse cursor, display a selectedPoint with its value
                            setSelectedPoint(closest.point)
                        } else {
                            console.log('setSelectedPoint(null)')
                            // Otherwise, hide the selectedPoint
                            setSelectedPoint(null);
                        }
                    }}
                    onMouseLeave={() => {
                        console.log('mouse leave')
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
                        console.log('closest', closest.distance, closest.point)
                        if (closest.distance < 25) {
                            // If a data point is close enough to the mouse cursor, display a selectedPoint with its value
                            setSelectedPoint(closest.point)
                        } else {
                            console.log('setSelectedPoint(null)')
                            // Otherwise, hide the selectedPoint
                            setSelectedPoint(null);
                        }
                    }}
                />
            </svg>
        </Tooltip>
    );
};