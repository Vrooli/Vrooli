import { SvgBase } from './base';
import { SvgProps } from './types';

export const DeleteNodeIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <g fill="none">
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M6.39 6.39h11.22v11.22H6.39z"
            />
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="m14.31 9.8-4.62 4.5m.05-4.61 4.5 4.62" />
        </g>
    </SvgBase>
)