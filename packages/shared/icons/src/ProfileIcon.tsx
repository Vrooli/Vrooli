import { SvgBase } from './base';
import { SvgProps } from './types';

export const ProfileIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="M12 1.63477c-4.076-.057-8.00712 2.56283-9.55093 6.33045-1.61151 3.70125-.77805 8.29903 2.04758 11.18645 2.71866 2.926 7.19766 3.99283 10.948 2.62772 3.88736-1.32033 6.7535-5.13913 6.90714-9.24598.25823-3.9923-2.05558-7.98297-5.63281-9.76496C15.26575 2.0217 13.6335 1.63213 12 1.63477Zm-.0078 2.96875c2.24842-.07434 4.15895 2.18674 3.7207 4.39062-.2957 2.15594-2.69038 3.67154-4.76536 3.0022-2.10359-.5393-3.3293-3.0838-2.43786-5.06491.55676-1.389 1.98704-2.3417 3.48252-2.32791Zm-.07031 8.88867c2.67362-.0065 5.36733 1.06299 7.20313 3.02734-1.89406 3.0944-5.91569 4.65872-9.39754 3.61714-2.02367-.5597-3.81174-1.9104-4.90717-3.70113 1.82903-1.90812 4.47276-2.94374 7.10158-2.94335Z"
        />
    </SvgBase>
)