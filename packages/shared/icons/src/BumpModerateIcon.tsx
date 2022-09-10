import { SvgBase } from './base';
import { SvgProps } from './types';

export const BumpModerateIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: 'none',
                stroke: props.fill ?? 'white',
                strokeWidth: "1.89",
                strokeLinecap: "round",
                strokeLinejoin: "round",
            }}
            d="M5.3 11 12 5l6.7 5.8M5.3 18.4l6.7-5.9 6.7 5.7"
        />
    </SvgBase>
)