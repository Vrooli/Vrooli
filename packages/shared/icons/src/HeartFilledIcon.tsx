import { SvgBase } from './base';
import { SvgProps } from './types';

export const HeartFilledIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="M7.79883 3.35724c-1.41849 0-2.84044.54943-3.92188 1.63086-2.1636 2.1636-2.1636 5.68015 0 7.84375l.00391.0039c1.0031.98773 7.58399 7.58399 7.58399 7.58399.2948.29657.77439.29747 1.0703.002l7.5879-7.58985c2.1636-2.1636 2.1636-5.68015 0-7.84375-2.1636-2.1636-5.68015-2.1636-7.84375 0L12 5.2674l-.2793-.2793c-1.08144-1.08143-2.5034-1.63086-3.92187-1.63086Z"
        />
    </SvgBase>
)