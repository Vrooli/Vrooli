import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Tooltip } from "@mui/material";
import { scaleLinear } from "d3-scale";
import { curveMonotoneX, line } from "d3-shape";
import { useMemo, useState } from "react";
const toLabel = (datum) => {
    if (typeof datum === "object" && datum !== null && "label" in datum) {
        return datum.label;
    }
    return "";
};
const toValue = (datum) => {
    if (typeof datum === "number") {
        return datum;
    }
    if (typeof datum === "object" && datum !== null && "value" in datum) {
        return datum.value;
    }
    return 0;
};
export const LineGraph = ({ dims, data, lineColor = "#000", lineWidth = 2, yAxisLabel, }) => {
    const maxData = Math.max(...data.map(toValue));
    const minData = Math.min(...data.map(toValue));
    const constantValue = minData === maxData;
    const xScale = scaleLinear().domain([0, data.length - 1]).range([0, dims.width]);
    const yScale = scaleLinear().domain([constantValue ? minData - 0.5 : 0, maxData]).range([dims.height, 0]);
    const points = data.map((datum, index) => ({
        x: xScale(index),
        y: yScale(toValue(datum)),
        label: toLabel(datum),
        value: toValue(datum),
    }));
    const [selectedPoint, setSelectedPoint] = useState(null);
    const d3Line = line()
        .x((point) => point.x)
        .y((point) => point.y)
        .curve(curveMonotoneX);
    const path = d3Line(points);
    const tooltipText = useMemo(() => {
        if (!selectedPoint)
            return "";
        return `${selectedPoint.label.length > 0 ? `${selectedPoint.label}: ` : ""} ${selectedPoint.value}`;
    }, [selectedPoint]);
    const renderGrid = () => {
        const numOfHorizontalLines = 5;
        const numOfVerticalLines = data.length > 1 ? data.length - 1 : 1;
        const horizontalLines = [...Array(numOfHorizontalLines)].map((_, index) => {
            const y = (index / (numOfHorizontalLines - 1)) * dims.height;
            return (_jsx("line", { x1: 0, y1: y, x2: dims.width, y2: y, stroke: "#ccc", strokeWidth: 1 }, `h-line-${index}`));
        });
        const verticalLines = [...Array(numOfVerticalLines)].map((_, index) => {
            const x = (index / (numOfVerticalLines - 1)) * dims.width;
            return (_jsx("line", { x1: x, y1: 0, x2: x, y2: dims.height, stroke: "#ccc", strokeWidth: 1 }, `v-line-${index}`));
        });
        return (_jsxs(_Fragment, { children: [horizontalLines, verticalLines] }));
    };
    const renderDataPoints = () => {
        return points.map((point, index) => (_jsx("circle", { cx: point.x, cy: point.y, r: 3, fill: selectedPoint === point ? lineColor : "#ccc", stroke: lineColor, strokeWidth: 1, onMouseEnter: () => setSelectedPoint(point), onMouseLeave: () => setSelectedPoint(null) }, `point-${index}`)));
    };
    const renderTooltip = () => {
        if (selectedPoint === null) {
            return null;
        }
        const tooltipWidth = 100;
        const tooltipHeight = 50;
        const tooltipX = Math.min(selectedPoint.x, dims.width - tooltipWidth);
        const tooltipY = Math.min(selectedPoint.y, dims.height - tooltipHeight);
        return (_jsxs("g", { children: [_jsx("rect", { x: tooltipX, y: tooltipY, width: tooltipWidth, height: tooltipHeight, fill: "white", stroke: lineColor, strokeWidth: 1 }), _jsx("text", { x: tooltipX + 5, y: tooltipY + 20, fontSize: 14, fill: lineColor, children: selectedPoint.label }), _jsxs("text", { x: tooltipX + 5, y: tooltipY + 35, fontSize: 12, fill: lineColor, children: ["Value: ", selectedPoint.value] })] }));
    };
    return (_jsx(Tooltip, { title: tooltipText, placement: "top", sx: {}, children: _jsxs("svg", { width: dims.width, height: dims.height, children: [renderGrid(), _jsx("path", { d: path, stroke: lineColor, strokeWidth: lineWidth, fill: "none", onMouseMove: (e) => {
                        const svgRect = e.currentTarget.getBoundingClientRect();
                        const x = e.clientX - svgRect.left;
                        const y = e.clientY - svgRect.top;
                        const closest = points.reduce((acc, point) => {
                            const distance = Math.sqrt((point.x - x) ** 2 + (point.y - y) ** 2);
                            if (distance < acc.distance) {
                                return { point, distance };
                            }
                            return acc;
                        }, { point: null, distance: Number.MAX_VALUE });
                        if (closest.distance < 25) {
                            setSelectedPoint(closest.point);
                        }
                        else {
                            setSelectedPoint(null);
                        }
                    }, onMouseLeave: () => {
                        setSelectedPoint(null);
                    }, onClick: (e) => {
                        const svgRect = e.currentTarget.getBoundingClientRect();
                        const x = e.clientX - svgRect.left;
                        const y = e.clientY - svgRect.top;
                        const closest = points.reduce((acc, point) => {
                            const distance = Math.sqrt((point.x - x) ** 2 + (point.y - y) ** 2);
                            if (distance < acc.distance) {
                                return { point, distance };
                            }
                            return acc;
                        }, { point: null, distance: Number.MAX_VALUE });
                        if (closest.distance < 25) {
                            setSelectedPoint(closest.point);
                        }
                        else {
                            setSelectedPoint(null);
                        }
                    } }), renderDataPoints(), renderTooltip()] }) }));
};
//# sourceMappingURL=LineGraph.js.map