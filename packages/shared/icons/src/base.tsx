import { SvgProps } from "./types";

export interface SvgBaseProps {
    props: SvgProps;
    children: React.ReactNode;
}

export const SvgBase = ({ props, children }: SvgBaseProps): JSX.Element => (
    <svg
        id={props.id}
        style={props.style}
        // Defaults width/height to 24px. "unset" is used to scale to max. NOTE: width/height must be set for icons
        // to show in Safari, so we but a high value here instead of undefined
        width={!props.width ? '24px' : props.width === 'unset' ? '5000px' : props.width}
        height={!props.height ? '24px' : props.height === 'unset' ? '5000px' : props.height}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
    >
        {children}
    </svg>
);

export const SvgPath = ({ d, props }: Pick<SvgBaseProps, 'props'> & { d: string }): JSX.Element => (
    <svg
        id={props.id}
        style={props.style}
        // Defaults width/height to 24px. "unset" is used to scale to max. NOTE: width/height must be set for icons
        // to show in Safari, so we but a high value here instead of undefined
        width={!props.width ? '24px' : props.width === 'unset' ? '5000px' : props.width}
        height={!props.height ? '24px' : props.height === 'unset' ? '5000px' : props.height}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
    >
        <path
            style={{
                fill: props.fill ?? 'white',
                fillOpacity: 1,
            }}
            d={d}
        />
    </svg>
);
