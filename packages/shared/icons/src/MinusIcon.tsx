import { SvgBase } from './base';
import { SvgProps } from './types';

export const MinusIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="M4.9121 10.48828A1.29886 1.51181 0 0 0 3.61329 12a1.29886 1.51181 0 0 0 1.29883 1.51172H19.0879A1.29886 1.51181 0 0 0 20.38672 12a1.29886 1.51181 0 0 0-1.29883-1.51172Z"
        />
    </SvgBase>
)