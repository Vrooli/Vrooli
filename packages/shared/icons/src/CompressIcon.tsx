import { SvgBase } from './base';
import { SvgProps } from './types';

export const CompressIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                stroke: props.fill ?? 'white',
                strokeWidth: "1.89",
                strokeLinecap: "round",
            }}
            fill="none"
            d="m12.99 18.21-1.84.02.9-1.2zm-.91 3.76v-4.4m-.99-11.78 1.84-.02-.9 1.2ZM12 2.03v4.4M4.65 13.5h14.7m-14.7-2.9h14.7"
        />
    </SvgBase>
)