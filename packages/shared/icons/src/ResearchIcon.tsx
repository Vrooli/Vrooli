import { SvgBase } from './base';
import { SvgProps } from './types';

export const ResearchIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <g transform="translate(0 -.551)">
            <rect
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 1,
                }}
                width="5.303"
                height="7.11"
                x="9.349"
                y="5.131"
                ry="0"
            />
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 1,
                }}
                d="M19.578.365a1.89 1.89 0 0 0-1.658.908l-5.195 8.553c-.756 1.243.117 2.838 1.572 2.871l10.004.223c1.454.033 2.4-1.52 1.701-2.797l-4.809-8.775a1.89 1.89 0 0 0-1.615-.983Z"
                transform="matrix(1.31588 -.0186 .02386 1.02567 -13.712 9.792)"
            />
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 1,
                }}
                d="M19.372.362a1.89 1.89 0 0 0-1.658.908l-5.195 8.552c-.756 1.244.118 2.839 1.572 2.872l10.004.222c1.454.034 2.4-1.52 1.701-2.797l-4.809-8.775a1.89 1.89 0 0 0-1.615-.982Z"
                transform="matrix(-.63075 .00708 -.01144 -.39058 24.195 7.385)"
            />
        </g>
    </SvgBase>
)