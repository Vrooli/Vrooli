import { SvgBase } from './base';
import { SvgProps } from './types';

export const EditIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: props.fill ?? 'white',
                strokeWidth: 0,
            }}
            d="M10.18 5.92v13.4L12 22.48l1.82-3.15V5.93Zm.77-4.26a.8.8 0 0 0-.8.8v2.3h3.7v-2.3a.8.8 0 0 0-.8-.8Z"
            transform="rotate(45 13.38 10.74) scale(1.10751)"
        />
    </SvgBase>
)