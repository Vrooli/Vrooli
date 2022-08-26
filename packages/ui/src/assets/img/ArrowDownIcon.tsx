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
            d="M12 23.0006V1.0232"
        /><path
            style={{
                fill: props.fill ?? 'white',
                stroke: props.fill ?? 'white',
                strokeWidth: "2",
                strokeLinecap: "round",
            }}
            d="m11.9435 22.954-9.0156-7.9108M12.0672 22.9536l9.0156-7.9107"
        />
    </svg>
)