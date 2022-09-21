import { SvgBase } from './base';
import { SvgProps } from './types';

export const HeartOutlineIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                strokeOpacity: 1,
                stroke: props.fill ?? 'white',
                strokeWidth: 1.5,
                strokeLinecap: 'round',
                strokeLinejoin: 'round',
                fill: 'none',
            }}
            d="M7.7983 4.11302c-1.22462 0-2.44924.4711-3.3876 1.40948-1.87674 1.87673-1.87674 4.89847 0 6.7752C5.43448 13.30583 12 19.88702 12 19.88702l7.5893-7.5893c1.87674-1.87674 1.87674-4.89848 0-6.77521-1.87673-1.87674-4.89847-1.87674-6.7752 0L12 6.33659l-.8141-.8141c-.93836-.93836-2.16298-1.40947-3.3876-1.40947Z"
        />
    </SvgBase>
)