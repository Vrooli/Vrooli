import { SvgBase } from './base';
import { SvgProps } from './types';

export const RedoIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                stroke: props.fill ?? 'white',
                strokeWidth: 1.89,
                strokeLinecap: 'round',
            }}
            fill="none"
            d="m19.93 14.2.05-1.63.05-1.63-2.06 1.6-2.06 1.59 2 .04zm-.5-.01c-8.32-9.93-15.52 0-15.52 0"
        />
    </SvgBase>
)