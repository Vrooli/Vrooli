import { SvgProps } from './types';

export const VisibleIcon = (props: SvgProps) => (
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
                strokeWidth: 0,
            }}
            d="M14.92 12A2.92 2.92 0 0 1 12 14.92 2.92 2.92 0 0 1 9.08 12 2.92 2.92 0 0 1 12 9.08 2.92 2.92 0 0 1 14.92 12Zm-2.95-5.93a9.68 9.68 0 0 0-8.88 5.87 9.68 9.68 0 0 0 8.93 6 9.68 9.68 0 0 0 8.9-5.9 9.68 9.68 0 0 0-8.95-5.97ZM12 7.46A4.55 4.55 0 0 1 16.54 12 4.55 4.55 0 0 1 12 16.54 4.55 4.55 0 0 1 7.46 12 4.55 4.55 0 0 1 12 7.46Z"
        />
    </svg>
)