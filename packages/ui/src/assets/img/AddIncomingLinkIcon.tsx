import { SvgProps } from './types';

export const AddIncomingLinkIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
        <g fill="none">
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M5.9 12h5.59m.04-5.61h11.22V17.6H11.53ZM5.66 12a2.25 2.25 0 0 1-2.25 2.25A2.25 2.25 0 0 1 1.16 12 2.25 2.25 0 0 1 3.4 9.75 2.25 2.25 0 0 1 5.66 12Z"
            />
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M9.41 4.65H2.97M6.22 1.4v6.44"
            />
        </g>
    </svg>
)