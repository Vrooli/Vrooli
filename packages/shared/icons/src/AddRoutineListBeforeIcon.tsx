import { SvgBase } from './base';
import { SvgProps } from './types';

export const AddRoutineListBeforeIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <g fill="none">
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M9.16 12h5.6M.85 8.21h7.58v7.58H.87Zm14.54 0h7.57v7.58H15.4z"
            />
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.37953",
                    strokeLinecap: "round",
                }}
                d="M2.8 12.09h3.73M4.65 10.2v3.73"
            />
        </g>
    </SvgBase>
)