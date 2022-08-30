import { SvgProps } from './types';

export const DarkModeIcon = (props: SvgProps) => (
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
                fillOpacity: 1,
                stroke: props.fill ?? 'white',
                strokeWidth: "1",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="M6.542 2.37a9.772 9.782 0 0 0-3.11 4.898 9.772 9.782 0 0 0 6.91 11.979 9.772 9.782 0 0 0 11.213-5.015 10.648 10.657 0 0 1-7.39.7A10.648 10.657 0 0 1 6.542 2.37Z"
        />
    </svg>
)