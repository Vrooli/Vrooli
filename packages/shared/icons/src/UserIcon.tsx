import { SvgBase } from './base';
import { SvgProps } from './types';

export const UserIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: props.fill ?? 'white',
                strokeWidth: 0,
            }}
            d="M2.9 21.7v-3.5s3-4.03 8.97-4.03c5.95 0 9.23 4 9.23 4v3.49zM16.63 8.09A4.63 4.63 0 0 1 12 12.7a4.63 4.63 0 0 1-4.63-4.63A4.63 4.63 0 0 1 12 3.44a4.63 4.63 0 0 1 4.63 4.64Z"
            transform="translate(0 -.57)"
        />
    </SvgBase>
)