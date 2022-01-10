import React, { useMemo } from 'react';
import { letterFrequency } from '@visx/mock-data';
import { Group } from '@visx/group';
import { Bar } from '@visx/shape';
import { scaleLinear, scaleBand } from '@visx/scale';
import { AxisLeft, AxisBottom } from '@visx/axis'
import { BarGraphProps } from '../types';

export interface Dimensions {
    width: number | undefined;
    height: number | undefined;
}

export interface Margins {
    top: number;
    bottom: number;
    left: number;
    right: number;
}

const DEFAULT_WIDTH = 500;
const DEFAULT_HEIGHT = 500;
const DEFAULT_MARGINS: Margins = { top: 20, bottom: 20, left: 20, right: 20 };

// Finally we'll embed it all in an SVG
export const BarGraph = ({
    className,
    data = letterFrequency,
    dimensions,
    margins = DEFAULT_MARGINS,
    style,
}: BarGraphProps) => {

    const state = useMemo(() => {
        // Determine the graph dimensions
        const width = dimensions?.width || DEFAULT_WIDTH;
        const height = dimensions?.height || DEFAULT_HEIGHT;
        // Create bounds based on available dimensions and margins
        const xMax = width - margins.left - margins.right;
        const yMax = height - margins.top - margins.bottom;
        // We'll make some helpers to get at the data we want
        const x = d => d.letter;
        const y = d => +d.frequency * 100;
        // Scale the graph by our data
        const xScale = scaleBand({
            range: [0, xMax],
            round: true,
            domain: data.map(x),
            padding: 0.4,
        });
        const yScale = scaleLinear({
            range: [yMax, 0],
            round: true,
            domain: [0, Math.max(...data.map(y))],
        });
        // Compose together the scale and accessor functions to get point functions
        const compose = (scale, accessor) => data => scale(accessor(data));
        const xPoint = compose(xScale, x);
        const yPoint = compose(yScale, y);
        // Return the state we need to render the graph
        return {
            width,
            height,
            xMax,
            yMax,
            xScale,
            yScale,
            xPoint,
            yPoint,
        };
    }, [data, dimensions, margins])

    return (
        <svg className={className} width={state.width} height={state.height} style={style}>
            <Group left={20}>
                {data.map((d, i) => {
                    const barHeight = state.yMax - state.yPoint(d);
                    // console.log('barHeighttt', state.yMax, state.yPoint(d));
                    return (
                        <Group key={`bar-${i}`}>
                            <Bar
                                x={state.xPoint(d)}
                                y={state.yMax - barHeight}
                                height={barHeight}
                                width={state.xScale.bandwidth()}
                                fill="#fc2e1c"
                            />
                        </Group>
                    );
                })}
            </Group>
            <Group>
                <AxisBottom left={20} top={state.yMax} scale={state.xScale} />
            </Group>
            <Group>
                <AxisLeft left={20} scale={state.yScale} />
            </Group>
        </svg>
    );
}