import { SyntheticEvent } from "react";
import { IconInfo } from "../../icons/Icons.js";
import { ViewDisplayType } from "../../types.js";
import { TitleProps } from "../text/types.js";

export type NavbarProps = {
    below?: JSX.Element | boolean | undefined;
    color?: string;
    help?: string | undefined;
    keepVisible?: boolean | undefined;
    options?: {
        iconInfo?: IconInfo | null | undefined;
        label: string;
        onClick: (event: SyntheticEvent) => unknown;
    }[];
    shouldHideTitle?: boolean;
    /** Sets tab title, if different than the Navbar title */
    tabTitle?: string;
    title?: string | undefined;
    titleBehavior?: "Hide" | "Show";
    /** Replaces title if provided */
    titleComponent?: JSX.Element;
}

export type SettingsTopBarProps = TopBarProps

export interface TopBarProps extends TitleProps {
    below?: JSX.Element | boolean;
    display: ViewDisplayType;
    keepVisible?: boolean;
    onClose?: () => unknown;
    startComponent?: JSX.Element;
    tabTitle?: string;
    titleBehavior?: NavbarProps["titleBehavior"];
    titleId?: string;
}
