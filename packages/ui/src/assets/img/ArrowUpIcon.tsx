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
            d="m12.05 3.2 7.22 6.36M11.95 3.2 4.73 9.56M12 3.16v17.67"
        />
    </svg>
)