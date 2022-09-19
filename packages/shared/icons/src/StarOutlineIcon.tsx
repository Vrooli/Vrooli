import { SvgBase } from './base';
import { SvgProps } from './types';

export const StarOutlineIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                strokeOpacity: 1,
                stroke: props.fill ?? 'white',
                strokeWidth: 2.59,
                strokeLinecap: 'round',
                strokeLinejoin: 'round',
                fill: 'none',
            }}
            d="M-1.25 20.5625.4198 9.97258l-7.77708-7.37913L3.23033.90906l4.61472-9.67672 4.8737 9.54891 10.62914 1.39859-7.5755 7.58594 1.95445 10.5411-9.55562-4.86055Z" 
            transform="matrix(.58475 .00774 -.00773 .58362 7.38177 8.4958)"
        />
    </SvgBase>
)