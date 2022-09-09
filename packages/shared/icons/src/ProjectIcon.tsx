import { SvgBase } from './base';
import { SvgProps } from './types';

export const ProjectIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
            }}
            d="M8.4 4v7h7.2V4Zm9.4 8.7a3.7 3.7 0 0 0-3.6 3.7 3.7 3.7 0 0 0 3.6 3.6 3.7 3.7 0 0 0 3.7-3.6 3.7 3.7 0 0 0-3.7-3.7zm-11.2.2L2.5 20h8.3z"
        />
    </SvgBase>
)