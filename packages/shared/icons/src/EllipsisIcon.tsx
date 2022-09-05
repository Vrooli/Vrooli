import { SvgBase } from './base';
import { SvgProps } from './types';

export const EllipsisIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: props.fill ?? 'white',
                strokeWidth: 0,
            }}
            d="M22.28 12a2.77 2.77 0 0 1-2.77 2.77A2.77 2.77 0 0 1 16.74 12a2.77 2.77 0 0 1 2.77-2.77A2.77 2.77 0 0 1 22.28 12Zm-7.56 0a2.77 2.77 0 0 1-2.77 2.77A2.77 2.77 0 0 1 9.18 12a2.77 2.77 0 0 1 2.77-2.77A2.77 2.77 0 0 1 14.72 12Zm-7.56 0a2.77 2.77 0 0 1-2.77 2.77A2.77 2.77 0 0 1 1.62 12 2.77 2.77 0 0 1 4.4 9.23 2.77 2.77 0 0 1 7.16 12Z"
        />
    </SvgBase>
)