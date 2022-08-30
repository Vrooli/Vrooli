import { SvgProps } from './types';

export const AddLinkIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
            <path
                style={{
                    fill: props.fill ?? 'white',
                    fillOpacity: 0,
                    stroke: props.fill ?? 'white',
                    strokeWidth: "1.75748",
                    strokeLinecap: "round",
                }}
                d="M6.3 14.62h11.4m5.18 0a2.25 2.25 0 0 1-2.25 2.26 2.25 2.25 0 0 1-2.26-2.25 2.25 2.25 0 0 1 2.25-2.26 2.25 2.25 0 0 1 2.26 2.26Zm-17.22 0a2.25 2.25 0 0 1-2.25 2.26 2.25 2.25 0 0 1-2.25-2.25 2.25 2.25 0 0 1 2.25-2.26 2.25 2.25 0 0 1 2.25 2.26Zm9.26-6.26H9.08M12 5.4v5.84"
                fill="none"
            />
    </svg>
)