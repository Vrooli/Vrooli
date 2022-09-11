import { SvgBase } from './base';
import { SvgProps } from './types';

export const ArrowUpIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: props.fill ?? 'white',
                stroke: props.fill ?? 'white',
                strokeWidth: "2",
                strokeLinecap: "round",
            }}
            d="m12.05 3.2 7.22 6.36M11.95 3.2 4.73 9.56M12 3.16v17.67"
        />
    </SvgBase>
)