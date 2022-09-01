import { SvgProps } from "./types";

export interface SvgBaseProps {
    props: SvgProps;
    children: React.ReactNode;
}

export const SvgBase = ({ props, children }: SvgBaseProps): JSX.Element => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width === null ? '24px' : props.width === 'unset' ? undefined : props.width}
        height={props.height === null ? '24px' : props.height === 'unset' ? undefined : props.height}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
    >
        {children}
    </svg>
);