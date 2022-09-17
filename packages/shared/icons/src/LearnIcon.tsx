import { SvgBase } from './base';
import { SvgProps } from './types';

export const LearnIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <g>
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 1,
                }}
                d="M19.07 13.08c.02.85-1.43.53-.98-.26.26-.48 1-.25.98.26zm-1.06-4.76h1.05v4.82h-1.05V8.32zM12 4.76l7.07 3.57L12 11.89 4.93 8.33 12 4.76Z"
                transform="translate(-5 -2.87) scale(1.41668)"
            />
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 1,
                }}
                d="M6.98 13.68C8.3 15 10 16.13 11.91 16.22c2.01.1 3.92-1 5.18-2.5m-10.11-3.1v3.05l5.1 2.46 5.02-2.41v-3.05l-5.01 2.41-5.11-2.46Z"
                transform="translate(-5 -2.87) scale(1.41668)" />
        </g>
    </SvgBase>
)