import { SvgBase } from './base';
import { SvgProps } from './types';

export const AddLinkIcon = (props: SvgProps) => (
    <SvgBase props={props}>
            <path
                style={{
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.75748",
                    strokeLinecap: "round",
                }}
                d="M6.3 14.62h11.4m5.18 0a2.25 2.25 0 0 1-2.25 2.26 2.25 2.25 0 0 1-2.26-2.25 2.25 2.25 0 0 1 2.25-2.26 2.25 2.25 0 0 1 2.26 2.26Zm-17.22 0a2.25 2.25 0 0 1-2.25 2.26 2.25 2.25 0 0 1-2.25-2.25 2.25 2.25 0 0 1 2.25-2.26 2.25 2.25 0 0 1 2.25 2.26Zm9.26-6.26H9.08M12 5.4v5.84"
                fill="none"
            />
    </SvgBase>
)