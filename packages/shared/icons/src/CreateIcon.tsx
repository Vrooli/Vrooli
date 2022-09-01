import { SvgBase } from './base';
import { SvgProps } from './types';

export const CreateIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.5",
                strokeLinecap: "round",
            }}
            d="m15.96851 11.99532-7.93702.0094M12 8.0315v7.93702M20.82306 12A8.82306 8.82306 0 0 1 12 20.82306 8.82306 8.82306 0 0 1 3.17694 12 8.82306 8.82306 0 0 1 12 3.17694 8.82306 8.82306 0 0 1 20.82306 12Z"
            fill="none"
        />
    </SvgBase>
)