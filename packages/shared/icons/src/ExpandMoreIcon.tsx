import { SvgBase } from './base';
import { SvgProps } from './types';

export const ExpandMoreIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                stroke: props.fill ?? 'white',
                strokeWidth: 1.89,
                strokeLinecap: 'round',
                strokeOpacity: 1,
                fill: 'none',
            }}
            d="m3.2 8 8.84 8 8.75-7.78"
        />
    </SvgBase>
)