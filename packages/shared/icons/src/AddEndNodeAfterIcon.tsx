import { SvgBase } from './base';
import { SvgProps } from './types';

export const AddEndNodeAfterIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <g fill="none">
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M20.5 12a1.92 1.92 0 0 1-1.92 1.92A1.92 1.92 0 0 1 16.66 12a1.92 1.92 0 0 1 1.92-1.92A1.92 1.92 0 0 1 20.5 12Z"
            />
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M9.28 11.89h3.91M.87 8.2h7.57v7.58H.87ZM23.18 12a4.6 4.6 0 0 1-4.6 4.6 4.6 4.6 0 0 1-4.6-4.6 4.6 4.6 0 0 1 4.6-4.6 4.6 4.6 0 0 1 4.6 4.6z"
            />
        </g>
    </SvgBase>
)