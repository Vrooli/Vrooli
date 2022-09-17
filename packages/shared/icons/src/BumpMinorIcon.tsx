import { SvgBase } from './base';
import { SvgProps } from './types';

export const BumpMinorIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: 'none',
                stroke: props.fill ?? 'white',
                strokeWidth: "1.89",
                strokeLinecap: "round",
                strokeLinejoin: "round",
            }}
            d="M5.3 15 12 9l6.7 5.8"
        />
    </SvgBase>
)