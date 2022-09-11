import { SvgBase } from './base';
import { SvgProps } from './types';

export const AddIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "2.3",
                strokeLinecap: "round",
            }}
            d="M19.51 12H4.49M12 4.5v15.02"
            fill="none"
        />
    </SvgBase>
)