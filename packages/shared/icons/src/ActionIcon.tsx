import { SvgBase } from './base';
import { SvgProps } from './types';

export const ActionIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: 'none',
                stroke: props.fill ?? 'white',
                strokeWidth: "1.51",
                strokeLinecap: "round",
            }}
            d="M20.82306 12A8.82306 8.82306 0 0 1 12 20.82306 8.82306 8.82306 0 0 1 3.17694 12 8.82306 8.82306 0 0 1 12 3.17694 8.82306 8.82306 0 0 1 20.82306 12Z"
        />
        <path
            style={{
                stroke: 'none',
                fill: props.fill ?? 'white',
            }}
            d="M16.07517 12A4.07517 4.07517 0 0 1 12 16.07517 4.07517 4.07517 0 0 1 7.92483 12 4.07517 4.07517 0 0 1 12 7.92483 4.07517 4.07517 0 0 1 16.07517 12Z"
        />
    </SvgBase>
)