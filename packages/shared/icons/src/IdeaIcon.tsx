import { SvgBase } from './base';
import { SvgProps } from './types';

export const IdeaIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
            }}
            d="M14.8 18.28c.48-1.41 1.15-2.36 1.95-3.56.93-1.75 1.47-2.8 1.13-4.79a6.08 6.08 0 0 0-3.09-3.82 7.9 7.9 0 0 0-5.62.05 6.13 6.13 0 0 0-2.9 4.05c-.32 1.62.21 3.06 1.12 4.46a12.73 12.73 0 0 1 2 3.61m.24 1.22c.07.8.1 1.79.86 2.24 1 .13 2.02.12 3.02.03.8-.22.92-1.2.99-1.9 0-.13.02-.25.03-.37"
            transform="matrix(.81414 0 0 .8975 2.17 2.32)"
        />
        <path
            style={{
                strokeOpacity: 1,
                stroke: props.fill ?? 'white',
                strokeWidth: 1.13,
                strokeLinecap: 'round',
                strokeLinejoin: 'round',
            }}
            d="m1.29 11.08 3.21.33M3.1 5.94l2.5 2.03m15.37-1.96L18.4 8M8.45 2.35l.94 3.1m6.15-3.15L14.4 5.31m8.31 6.16-3.22.25"
        />
    </SvgBase>
)