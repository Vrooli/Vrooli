import { SvgProps } from './types';

export const RoutineValidIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
        <path
            style={{
                fill: props.fill ?? 'white',
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
            d="M16.55971 10.32137a1.07117 1.07117 0 0 1-1.07117 1.07117 1.07117 1.07117 0 0 1-1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117-1.07117 1.07117 1.07117 0 0 1 1.07117 1.07117zm-6.97708 0a1.07117 1.07117 0 0 1-1.07117 1.07117 1.07117 1.07117 0 0 1-1.07117-1.07117A1.07117 1.07117 0 0 1 8.51146 9.2502a1.07117 1.07117 0 0 1 1.07117 1.07117Z"
        />
        <path
            style={{
                fill: props.fill ?? 'white',
                stroke: props.fill ?? 'white',
                strokeWidth: "1",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M17.28145 14.26879H6.71855a5.28154 3.9889 0 0 0 5.28232 3.9895 5.28154 3.9889 0 0 0 5.28058-3.9895z"
        />
    </svg>
)