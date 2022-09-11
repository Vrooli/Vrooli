import { SvgBase } from './base';
import { SvgProps } from './types';

export const SearchIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                stroke: props.fill ?? 'white',
                strokeWidth: "1.37952",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="m19.685 19.685-6.622-6.622m-.22-7.288a4.999 4.999 0 0 1 0 7.069 4.999 4.999 0 0 1-7.068 0 4.999 4.999 0 0 1 0-7.07 4.999 4.999 0 0 1 7.069 0z"
            fill="none"
        />
    </SvgBase>
)