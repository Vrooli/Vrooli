import { SvgBase } from './base';
import { SvgProps } from './types';

export const CloseIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: props.fill ?? 'white',
                stroke: props.fill ?? 'white',
                strokeWidth: "2",
                strokeLinecap: "round",
            }}
            d="m4.24 4.24 15.52 15.52m0-15.52L4.24 19.76"
        />
    </SvgBase>
)