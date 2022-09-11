import { SvgBase } from './base';
import { SvgProps } from './types';

export const InvisibleIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: props.fill ?? 'white',
                fillOpacity: 1,
                strokeWidth: 0,
            }}
            d="M11.97 6.07a9.68 9.68 0 0 0-3.18.56l1.28 1.28A4.55 4.55 0 0 1 12 7.46 4.55 4.55 0 0 1 16.54 12a4.55 4.55 0 0 1-.45 1.93L18 15.84a9.68 9.68 0 0 0 2.91-3.8 9.68 9.68 0 0 0-8.94-5.97ZM6.92 7.52a9.68 9.68 0 0 0-3.83 4.42 9.68 9.68 0 0 0 8.93 6 9.68 9.68 0 0 0 4.28-1.04l-1.4-1.4a4.55 4.55 0 0 1-2.9 1.04A4.55 4.55 0 0 1 7.46 12 4.55 4.55 0 0 1 8.5 9.1ZM12 9.08a2.92 2.92 0 0 0-.68.08l3.52 3.52a2.92 2.92 0 0 0 .08-.68A2.92 2.92 0 0 0 12 9.08Zm-2.34 1.18A2.92 2.92 0 0 0 9.08 12 2.92 2.92 0 0 0 12 14.92a2.92 2.92 0 0 0 1.74-.58Z"
        />
        <path
            style={{
                stroke: props.fill ?? 'white',
                fillOpacity: 0,
                strokeWidth: 1.9,
                strokeLinecap: "round",
                strokeOpacity: 1,
            }}
            d="m5.04 5.04 13.92 13.92"
        />
    </SvgBase>
)