import { SvgProps } from './types';

export const ReorderIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
    >
        <path
            fill="none"
            style={{
                stroke: props.fill ?? 'white',
                strokeWidth: "1.89",
                strokeOpacity: 1,
                strokeLinecap: "round",
            }}
            d="M7.92 17.78v-1.35l.56.34.55.33-.55.34Zm-.03-10.2V6.21l.56.34.55.34-.55.33zm.23 9.58c-3.27 0-5.91-2.32-5.91-5.18S4.85 6.8 8.1 6.8m4.17 10.36h9.51m-9.51-5.18h9.51m-9.5-5.18h9.51"
        />
    </svg>
)