import { SyntheticEvent } from "react";
import { IconInfo } from "../../icons/Icons.js";
import { ViewDisplayType } from "../../types.js";
import { TitleProps } from "../text/types.js";

export interface PartialNavbarProps {
    /** Optional children to render in the navbar */
    children?: React.ReactNode;
    /** Optional color for the navbar text and icons */
    color?: string;
    /** Whether to keep the navbar visible when scrolling */
    keepVisible?: boolean;
}

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
}

export interface TopBarProps extends TitleProps {
    below?: JSX.Element | boolean;
    display: ViewDisplayType | `${ViewDisplayType}`
    keepVisible?: boolean;
    onClose?: () => unknown;
    startComponent?: JSX.Element;
    tabTitle?: string;
    titleBehavior?: NavbarProps["titleBehavior"];
    titleId?: string;
}
