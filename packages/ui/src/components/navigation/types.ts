import { SvgComponent } from "@local/shared";
import { TitleProps } from "components/text/types";
import { ViewDisplayType } from "views/types";

export interface ContactInfoProps {
    sx?: { [key: string]: any };
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
    title?: string | undefined;
    /** Replaces title if provided */
    titleComponent?: JSX.Element;
}

export type NavbarLogoState = "full" | "icon" | "none";
export interface NavbarLogoProps {
    onClick: () => void;
    state: NavbarLogoState;
}

export interface HideOnScrollProps {
    target?: any;
    children: JSX.Element;
}

export type SettingsTopBarProps = Omit<TopBarProps, "below">

export interface TopBarProps extends Omit<TitleProps, "variant"> {
    display: ViewDisplayType
    onClose?: () => void,
    below?: JSX.Element | boolean
    hideTitleOnDesktop?: boolean,
    titleId?: string
}
