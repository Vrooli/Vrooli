import { SvgBase } from './base';
import { SvgProps } from './types';

export const CompleteIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "2",
                strokeLinecap: "round",
            }}
            d="m3.3 12.35 5.44 5.62L20.7 5.62"
        />
    </SvgBase>
)