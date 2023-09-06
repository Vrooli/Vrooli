import { TitleProps } from "components/text/types";
import { SvgComponent, SxType } from "types";
import { ViewDisplayType } from "views/types";

export interface ContactInfoProps {
    sx?: SxType;
}

export type NavbarProps = {
    below?: JSX.Element | boolean | undefined;
    help?: string | undefined;
    options?: {
        Icon: SvgComponent;
        label: string;
        onClick: (e?: any) => void;
    }[];
    shouldHideTitle?: boolean;
    startComponent?: JSX.Element;
    /** Sets tab title, if different than the Navbar title */
    tabTitle?: string;
    title?: string | undefined;
    /** Replaces title if provided */
    titleComponent?: JSX.Element;
}

export type SettingsTopBarProps = TopBarProps

export interface TopBarProps extends TitleProps {
    display: ViewDisplayType
    onClose?: () => void,
    below?: JSX.Element | boolean
    hideTitleOnDesktop?: boolean,
    startComponent?: JSX.Element;
    tabTitle?: string,
    titleId?: string
}
