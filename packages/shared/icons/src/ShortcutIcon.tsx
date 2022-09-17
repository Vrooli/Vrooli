import { SvgBase } from './base';
import { SvgProps } from './types';

export const ShortcutIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                stroke: props.fill ?? 'white',
                strokeOpacity: 1,
                strokeWidth: 3.4,
                strokeLinecap: 'round',
                strokeLinejoin: 'round',
                fill: 'none',
            }}
            d="m19 12.5-3.5 3.7m3.6-3.7-3.6-3.7m-10.6-1v4.6H19"
        />
    </SvgBase>
)