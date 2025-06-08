import Typography from "@mui/material/Typography";
import { scaleLinear } from "d3-scale";
import { curveMonotoneX, line } from "d3-shape";
import { useMemo, useRef, useState } from "react";
import { PopoverWithArrow } from "../../dialogs/PopoverWithArrow/PopoverWithArrow.js";
import { type LineGraphProps } from "../types.js";

type Point = {
    x: number,
    y: number,
    label: string,
    value: number,
}

// type XAxisLabelsProps = {
//     data: any[];
//     xScale: any;
//     height: number;
// };

type YAxisLabelsProps = {
    yScale: any;
    numOfHorizontalLines: number;
    width: number;
};

function getRoundingPrecision(minValue: number, maxValue: number, numOfLines: number) {
    const range = maxValue - minValue;
    const increment = range / (numOfLines - 1);

    if (increment >= 1) {
        return Math.max(0, Math.ceil(Math.log10(increment)) - 1);
    } else {
        return Math.ceil(-Math.log10(increment));
    }
}

// const XAxisLabels: React.FC<XAxisLabelsProps> = ({ data, xScale, height }) => {
//     return (
//         <g>
//             {data.map((datum, index) => (
//                 <text
//                     key={`x-label-${index}`}
//                     x={xScale(index)}
//                     y={height + 15}
//                     fontSize={10}
//                     textAnchor="middle"
//                 >
//                     {toLabel(datum)}
//                 </text>
//             ))}
//         </g>
//     );
// };

function YAxisLabels({
    yScale,
    numOfHorizontalLines,
    width,
}: YAxisLabelsProps) {
    const minValue = yScale.domain()[0];
    const maxValue = yScale.domain()[1];

    const precision = getRoundingPrecision(minValue, maxValue, numOfHorizontalLines);

    const labels = [...Array(numOfHorizontalLines)].map((_, index) => {
        const value = (index / (numOfHorizontalLines - 1)) * (maxValue - minValue) + minValue;
        return (
            <g key={`y-label-${index}`} transform={`translate(-30, ${yScale(value)})`}>
                <foreignObject width="40" height="20" y={-10}>
                    <Typography
                        variant="body2"
                        component="span"
                        sx={{
                            textAnchor: "end",
                            dominantBaseline: "central",
                            display: "block",
                        }}
                    >
                        {value.toFixed(precision)}
                    </Typography>
                </foreignObject>
            </g>
        );
    });

    return <g>{labels}</g>;
}

// Function to extract the label from a data point
function toLabel(datum: any): string {
    if (typeof datum === "object" && datum !== null && "label" in datum) {
        return datum.label;
    }
    return "";
}

// Function to extract the value from a data point
function toValue(datum: any): number {
    if (typeof datum === "number") {
        return datum;
    }
    if (typeof datum === "object" && datum !== null && "value" in datum) {
        return datum.value;
    }
    return 0;
}

const padding = { top: 10, right: 35, bottom: 10, left: 10 };

/**
 * A line graph component to represent a list of numerical data as a line.
 */
export function LineGraph({
    dims,
    data,
    lineColor = "#000",
    lineWidth = 2,
    yAxisLabel,
}: LineGraphProps) {
    // Find the maximum and minimum value in the data array
    const maxData = Math.max(...data.map(toValue));
    const minData = Math.min(...data.map(toValue));
    const numOfHorizontalLines = 5;

    const xScale = scaleLinear()
        .domain([0, data.length - 1])
        .range([padding.left, dims.width - padding.right]);
    const yScale = scaleLinear()
        .domain([Math.floor(minData * 0.9), maxData === 0 ? 1 : Math.ceil(maxData * 1.1)])
        .range([dims.height - padding.bottom, padding.top]);

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

    // Anchor for popover that displays a point's label/value
    const [anchorEl, setAnchorEl] = useState<Element | null>(null);


    const grid = useMemo(() => {
        const numOfHorizontalLines = 5;
        const numOfVerticalLines = data.length > 1 ? data.length - 1 : 1;

        const horizontalLines = [...Array(numOfHorizontalLines)].map((_, index) => {
            const y = padding.top + ((dims.height - padding.top - padding.bottom) * index) / (numOfHorizontalLines - 1);
            return (
                <line
                    key={`h-line-${index}`}
                    x1={padding.left}
                    y1={y}
                    x2={dims.width - padding.right}
                    y2={y}
                    stroke="#ccc"
                    strokeWidth={1}
                />
            );
        });

        const verticalLines = [...Array(numOfVerticalLines)].map((_, index) => {
            const x = padding.left + ((dims.width - padding.left - padding.right) * index) / (numOfVerticalLines - 1);
            return (
                <line
                    key={`v-line-${index}`}
                    x1={x}
                    y1={padding.top}
                    x2={x}
                    y2={dims.height - padding.bottom}
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
    }, [data, dims.height, dims.width]);

    const closeTimeout = useRef<number | null>(null);

    const dataPoints = useMemo(() => {
        return points.map((point, index) => (
            <g key={`point-${index}`}>
                {/* Actual displayed circle */}
                <circle
                    cx={point.x}
                    cy={point.y}
                    r={3}
                    fill={selectedPoint === point ? lineColor : "#ccc"}
                    stroke={lineColor}
                    strokeWidth={1}
                />
                {/* Invisible circle that is used to increase the size of the hover area */}
                <circle
                    cx={point.x}
                    cy={point.y}
                    r={12} // Increase the radius to create a larger hover area
                    fill="transparent" // Make the circle invisible
                    onMouseEnter={(e) => {
                        // Clear the timeout if it exists
                        if (closeTimeout.current) {
                            clearTimeout(closeTimeout.current);
                            closeTimeout.current = null;
                        }
                        setSelectedPoint(point);
                        setAnchorEl(e.currentTarget as unknown as HTMLElement);
                    }}
                    onMouseLeave={() => {
                        // Delay the tooltip removal
                        closeTimeout.current = window.setTimeout(() => {
                            setSelectedPoint(null);
                            setAnchorEl(null); // Remove the anchor element
                        }, 500);
                    }}
                />
            </g>
        ));
    }, [lineColor, points, selectedPoint]);

    return (
        <>
            {/* The line graph */}
            <svg width={dims.width} height={dims.height}>
                {grid}
                {/* <g transform={`translate(0, ${dims.height})`}>
                    <XAxisLabels data={data} xScale={xScale} height={dims.height} />
                </g> */}
                <g transform={`translate(${dims.width}, 0)`}>
                    <YAxisLabels
                        yScale={yScale}
                        numOfHorizontalLines={numOfHorizontalLines}
                        width={dims.width}
                    />
                </g>
                <path
                    d={path}
                    stroke={lineColor}
                    strokeWidth={lineWidth}
                    fill="none"
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
                        if (closest.distance < 40) {
                            // If a data point is close enough to the mouse cursor, display a selectedPoint with its value
                            setSelectedPoint(closest.point);
                        } else {
                            // Otherwise, hide the selectedPoint
                            setSelectedPoint(null);
                        }
                    }}
                />
                {dataPoints}
            </svg>
            {selectedPoint && anchorEl && (
                <PopoverWithArrow
                    anchorEl={anchorEl}
                    handleClose={() => {
                        setSelectedPoint(null);
                        setAnchorEl(null);
                    }}
                    onMouseEnter={() => {
                        // Clear the timeout when the mouse enters the tooltip
                        if (closeTimeout.current) {
                            clearTimeout(closeTimeout.current);
                            closeTimeout.current = null;
                        }
                    }}
                    onMouseMove={(event) => {
                        // If the mouse moves far enough away from the data point, hide the tooltip
                        const svgRect = anchorEl.getBoundingClientRect();
                        const x = event.clientX - svgRect.left;
                        const y = event.clientY - svgRect.top;
                        const distance = Math.sqrt(x ** 2 + y ** 2);
                        console.log("on mouse moveee 1", event.clientX, event.clientY, svgRect.left, svgRect.top);
                        console.log("on mouse moveee 2", x, y, distance, "\n");
                        if (distance > 40) {
                            setSelectedPoint(null);
                            setAnchorEl(null);
                        }
                    }}
                >
                    <Typography variant="body2" color="textPrimary">
                        {tooltipText}
                    </Typography>
                </PopoverWithArrow>
            )}

        </>
    );
}
