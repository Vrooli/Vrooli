import { SvgProps } from './types';

export const RefreshIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.88976",
                strokeLinecap: "round",
            }}
            d="m12.75 3.53-.41.37-.41.37V2.8l.41.36zm-1.81-.29A8.82 8.82 0 0 0 3.18 12a8.82 8.82 0 0 0 2.67 6.33m6.1 1.94.4.36.42.37v-1.47l-.41.37zm1.93.35A8.82 8.82 0 0 0 20.82 12a8.82 8.82 0 0 0-2.32-5.96"
            fill="none"
        />
    </svg>
)