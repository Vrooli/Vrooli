import { SvgProps } from './types';

export const AddIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
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
    </svg>
)