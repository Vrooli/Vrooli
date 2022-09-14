import { SvgBase } from './base';
import { SvgProps } from './types';

export const StandardIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                stroke: props.fill ?? 'white',
                strokeOpacity: 1,
                strokeWidth: 1.5,
                strokeLinecap: 'round',
                strokeLinejoin: 'round',
                fill: 'none',
            }}
            d="m7.71189 13.9845 2.68121 2.65934 5.89501-5.8469M4.92784 9.38307h5.96114l.04462-6.07057m-.24671 0h8.38527v17.375H4.93676V9.31767c1.9467-1.97271 3.8031-4.03277 5.75013-6.00517Z"
        />
    </SvgBase>
)