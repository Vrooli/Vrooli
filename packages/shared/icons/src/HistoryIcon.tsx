import { SvgBase } from './base';
import { SvgProps } from './types';

export const HistoryIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: 'none',
                stroke: props.fill ?? 'white',
                strokeOpacity: 1,
                strokeWidth: 1.5,
                strokeLinecap: 'round',
                strokeLinejoin: 'round',
            }}
            d="M12 7.5v4.7zm0 4.7h4.7zm8.8-.2a8.8 8.8 0 0 1-8.8 8.8A8.8 8.8 0 0 1 3.2 12 8.8 8.8 0 0 1 12 3.2a8.8 8.8 0 0 1 8.8 8.8Z"
        />
    </SvgBase>
)