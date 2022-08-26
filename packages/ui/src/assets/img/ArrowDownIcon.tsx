import { SvgProps } from './types';

export const ArrowDownIcon = (props: SvgProps) => (
    <svg
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
        <path
            style={{
                fill: props.fill ?? 'white',
                stroke: props.fill ?? 'white',
                strokeWidth: "2",
                strokeLinecap: "round",
            }}
            d="m12.05 20.8 7.22-6.36m-7.32 6.36-7.22-6.36m7.27 6.4V3.16"
        />
    </svg>
)