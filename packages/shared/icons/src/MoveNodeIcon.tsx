import { SvgProps } from './types';

export const MoveNodeIcon = (props: SvgProps) => (
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
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.37953",
                strokeLinecap: "round",
            }}
            d="M14.18 19.33 12 20.96m-2.18-1.63L12 20.96m-.04-.07v-4.56m-7.29-2.15L3.04 12m1.63-2.18L3.04 12m.08-.04h4.55m11.7 2.22L21.02 12m-1.63-2.18L21 12m-.07-.04h-4.56m-2.2-7.29L12 3.04M9.82 4.67 12 3.04m-.04.07v4.56m-4.28.01h8.64v8.64H7.68Z"
            fill="none"
        />
    </svg>
)