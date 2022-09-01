import { SvgProps } from './types';

export const RedoIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
    >
        <path
            style={{
                stroke: props.fill ?? 'white',
                strokeWidth: 1.89,
                strokeLinecap: 'round',
            }}
            fill="none"
            d="m19.93 14.2.05-1.63.05-1.63-2.06 1.6-2.06 1.59 2 .04zm-.5-.01c-8.32-9.93-15.52 0-15.52 0"
        />
    </svg>
)