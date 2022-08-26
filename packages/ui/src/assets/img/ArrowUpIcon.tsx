import { SvgProps } from './types';

export const ArrowUpIcon = (props: SvgProps) => (
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
            d="M12 1.0251v21.9774"
        /><path
            style={{
                fill: props.fill ?? 'white',
                stroke: props.fill ?? 'white',
                strokeWidth: "2",
                strokeLinecap: "round",
            }}
            d="M11.9435 1.0717 2.9279 8.9825M12.0672 1.072l9.0156 7.9108"
        />
    </svg>
)