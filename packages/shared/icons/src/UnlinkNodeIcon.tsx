import { SvgBase } from './base';
import { SvgProps } from './types';

export const UnlinkNodeIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <g fill="none">
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M18.02 12h-5.6m-.04-5.61H1.16V17.6h11.22ZM18.25 12a2.25 2.25 0 0 0 2.25 2.25A2.25 2.25 0 0 0 22.75 12a2.25 2.25 0 0 0-2.25-2.25A2.25 2.25 0 0 0 18.25 12Z"
            />
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="m15.46 6.93 4.56-4.55m-4.6-.02 4.55 4.57"
            />
        </g>
    </SvgBase>
)