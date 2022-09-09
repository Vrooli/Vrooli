import { SvgBase } from './base';
import { SvgProps } from './types';

export const RoutineIncompleteIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.37953",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M20.82306 12A8.82306 8.82306 0 0 1 12 20.82306 8.82306 8.82306 0 0 1 3.17694 12 8.82306 8.82306 0 0 1 12 3.17694 8.82306 8.82306 0 0 1 20.82306 12Z"
            fill="none"
        />
        <path
            style={{
                fill: props.fill ?? 'white',
                stroke: props.fill ?? 'white',
                strokeWidth: "1",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M16.55971 11.04513a1.07117 1.07117 0 0 1-1.07117 1.07117 1.07117 1.07117 0 0 1-1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117 1.07117zm-6.97708 0a1.07117 1.07117 0 0 1-1.07117 1.07117 1.07117 1.07117 0 0 1-1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117 1.07117Z"
        />
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.1",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M7.61994 15.83595h8.76012"
        />
    </SvgBase>
)