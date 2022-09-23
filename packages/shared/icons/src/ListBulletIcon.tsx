import { SvgBase } from './base';
import { SvgProps } from './types';

export const ListBulletIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="M4.44531 3.90625a1.92969 1.92969 0 0 0-1.92969 1.92969 1.92969 1.92969 0 0 0 1.9297 1.92969A1.92969 1.92969 0 0 0 6.375 5.83592a1.92969 1.92969 0 0 0-1.92969-1.92968Zm4.23438.61719v2.64648H21.25V4.52344h-1.32227ZM4.3125 10.0703A1.92969 1.92969 0 0 0 2.38281 12a1.92969 1.92969 0 0 0 1.92969 1.92969A1.92969 1.92969 0 0 0 6.24219 12a1.92969 1.92969 0 0 0-1.92969-1.92969Zm4.36719.60742v2.64454H21.25v-2.64454h-1.32227ZM4.4375 16.22656a1.92969 1.92969 0 0 0-1.92969 1.92969 1.92969 1.92969 0 0 0 1.92969 1.92969 1.92969 1.92969 0 0 0 1.92969-1.92969 1.92969 1.92969 0 0 0-1.92969-1.92969Zm4.24219.57617v2.64454H21.25v-2.64454h-1.32227z"
        />
    </SvgBase>
)