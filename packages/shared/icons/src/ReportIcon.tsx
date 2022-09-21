import { SvgBase } from './base';
import { SvgProps } from './types';

export const ReportIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="M4.21474 2.7357v18.5286h1.98364v-9.57794h6.02961v1.56006h7.55727V4.29575h-6.03887V2.7357Z"
        />
    </SvgBase>
)