import { SvgBase } from './base';
import { SvgProps } from './types';

export const PauseIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="M16.46164 3.417a1.88976 1.88976 0 0 0-1.88867 1.88866v13.38672a1.88976 1.88976 0 0 0 1.88867 1.89063 1.88976 1.88976 0 0 0 1.89063-1.89063V5.30566A1.88976 1.88976 0 0 0 16.46164 3.417Zm-8.92523 0a1.88976 1.88976 0 0 0-1.88868 1.88866v13.38672a1.88976 1.88976 0 0 0 1.88868 1.89063 1.88976 1.88976 0 0 0 1.89062-1.89063V5.30566A1.88976 1.88976 0 0 0 7.5364 3.417Z"
        />
    </SvgBase>
)