import { SvgBase } from './base';
import { SvgProps } from './types';

export const ThoughtIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <g transform="translate(.4 1.03)">
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 1,
                }}
                d="M15.79 5.23c-2.07-1.22-4.13-.2-5.32.65-.76-1.03-2.73-1.67-4.59-1.47-1.12.1-2.92 1.18-3.2 2.29-.34.61.14 1.55-.63 1.84-.66.44-.83 1.45-.14 1.95.52.44 1.16.6 1.12 1.38.2.84 1.13 1.3 1.95 1.1.64-.04 1.3-.7 1.63.17a5.14 5.14 0 0 0 5.57.87c.55-.31.9-.8 1.6-.46 1 .1 2.2.2 3-.54.6-.42-.02-1.49.87-1.43 1.66-.55 2.84-2.57 2.07-4.25-.91-1.88-1.7-2.56-3.93-2.1z"
            />
            <circle
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 1,
                }}
                cx="18.34"
                cy="14.29"
                r="1.24"
            />
            <circle
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 1,
                }}
                cx="20.77"
                cy="16.59"
                r=".97"
            />
        </g>
    </SvgBase>
)