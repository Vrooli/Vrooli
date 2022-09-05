import { SvgBase } from './base';
import { SvgProps } from './types';

export const CancelIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.5",
                strokeLinecap: "round",
            }}
            d="M14.80947 14.80285 9.1905 9.19718m5.61566-.00334-5.61232 5.61232M20.82306 12A8.82306 8.82306 0 0 1 12 20.82306 8.82306 8.82306 0 0 1 3.17694 12 8.82306 8.82306 0 0 1 12 3.17694 8.82306 8.82306 0 0 1 20.82306 12Z"
            fill="none"
        />
    </SvgBase>
)