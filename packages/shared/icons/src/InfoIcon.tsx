import { SvgProps } from './types';

export const InfoIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
        <defs>
            <path
                id="a"
                d="M-12.733-4.258h10.399v16.745h-10.399z"
            />
        </defs>
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.37953",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M20.823 12A8.823 8.823 0 0 1 12 20.823 8.823 8.823 0 0 1 3.177 12 8.823 8.823 0 0 1 12 3.177 8.823 8.823 0 0 1 20.823 12Z"
            fill="none"
        />
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "2.13543",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M12 17.564v-7.042"
        />
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.75748",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M12.47 7.39a.491.491 0 0 1-.49.491.491.491 0 0 1-.492-.49.491.491 0 0 1 .492-.492.491.491 0 0 1 .49.491z"
        />
    </svg>
)