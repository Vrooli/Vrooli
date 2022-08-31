import { SvgProps } from './types';

export const CancelIcon = (props: SvgProps) => (
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
                strokeWidth: "1.5",
                strokeLinecap: "round",
            }}
            d="M14.80947 14.80285 9.1905 9.19718m5.61566-.00334-5.61232 5.61232M20.82306 12A8.82306 8.82306 0 0 1 12 20.82306 8.82306 8.82306 0 0 1 3.17694 12 8.82306 8.82306 0 0 1 12 3.17694 8.82306 8.82306 0 0 1 20.82306 12Z"
            fill="none"
        />
    </svg>
)