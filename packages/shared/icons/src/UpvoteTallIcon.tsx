import { SvgBase } from './base';
import { SvgProps } from './types';

export const UpvoteTallIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: props.fill ?? 'white',
                strokeWidth: 0,
            }}
            d="M18.68 19.13H5.32l3.34-6.56L12 6l3.34 6.56Z"
            transform="matrix(1.22093 0 0 1.08273 -2.65 -1.6)"
        />
    </SvgBase>
)