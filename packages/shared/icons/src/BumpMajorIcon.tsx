import { SvgBase } from './base';
import { SvgProps } from './types';

export const BumpMajorIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: 'none',
                stroke: props.fill ?? 'white',
                strokeWidth: "1.89",
                strokeLinecap: "round",
                strokeLinejoin: "round",
            }}
            d="M5.3 8.6 12 2.8l6.7 5.7M5.3 15 12 9l6.7 5.8M5.3 21.3l6.7-5.9 6.7 5.7"
        />
    </SvgBase>
)