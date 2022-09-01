import { SvgBase } from './base';
import { SvgProps } from './types';

export const LightModeIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fill: props.fill ?? 'white',
                fillOpacity: 1,
                stroke: props.fill ?? 'white',
                strokeWidth: "1",
                strokeLinecap: "round",
                strokeDasharray: "none",
            }}
            d="m19.105 4.98-2.343 2.314m-9.524 9.412L4.895 19.02m14.122.088-2.314-2.344m-9.406-9.53L4.983 4.891M12 22.604v-3.93m0-13.348v-3.93M16.688 12A4.688 4.688 0 0 1 12 16.688 4.688 4.688 0 0 1 7.312 12 4.688 4.688 0 0 1 12 7.312 4.688 4.688 0 0 1 16.688 12Zm5.916-.007-3.93.003m-13.348.008-3.93.003"
        />
    </SvgBase>
)