import { SvgProps } from './types';

export const CloseIcon = (props: SvgProps) => (
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
            d="m1.8353 1.8353 20.3294 20.3294M22.1647 1.8353 1.8353 22.1647"
        />
    </svg>
)