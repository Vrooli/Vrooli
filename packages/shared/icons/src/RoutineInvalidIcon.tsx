import { SvgBase } from './base';
import { SvgProps } from './types';

export const RoutineInvalidIcon = (props: SvgProps) => (
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
            d="M16.55971 11.37267a1.07117 1.07117 0 0 1-1.07117 1.07117 1.07117 1.07117 0 0 1-1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117 1.07117zm-6.97708 0a1.07117 1.07117 0 0 1-1.07117 1.07117 1.07117 1.07117 0 0 1-1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117 1.07117Z"
        />
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.3",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M8.13446 16.66468c4.01692-3.56112 7.73108.0327 7.73108.0327"
        />
    </SvgBase>
)