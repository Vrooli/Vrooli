import { SvgBase } from './base';
import { SvgProps } from './types';

export const HeaderIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="M13.02242 5.88231v11.568h-2.16v-5.056H6.15115v5.056h-2.16v-11.568h2.16v4.976h4.71127v-4.976Z"
            transform="translate(.04157 -4.39994) scale(1.40575)"
        />
    </SvgBase>
)