import { SvgProps } from './types';

export const AddOutgoingLinkIcon = (props: SvgProps) => (
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
                d="M18.02 12h-5.6m-.04-5.61H1.16V17.6h11.22ZM18.25 12a2.25 2.25 0 0 0 2.25 2.25A2.25 2.25 0 0 0 22.75 12a2.25 2.25 0 0 0-2.25-2.25A2.25 2.25 0 0 0 18.25 12Z"
            />
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M14.5 4.65h6.44M17.69 1.4v6.44"
            />
        </g>
    </svg>
)