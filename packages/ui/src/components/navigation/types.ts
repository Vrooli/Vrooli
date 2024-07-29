import { TitleProps } from "components/text/types";
import { SvgComponent, SxType, ViewDisplayType } from "types";

export interface ContactInfoProps {
    sx?: SxType;
}

export type NavbarProps = {
    below?: JSX.Element | boolean | undefined;
    help?: string | undefined;
    keepVisible?: boolean | undefined;
    options?: {
        Icon: SvgComponent;
        label: string;
        onClick: (e?: any) => unknown;
    }[];
    shouldHideTitle?: boolean;
    startComponent?: JSX.Element;
    /** Sets tab title, if different than the Navbar title */
    tabTitle?: string;
    title?: string | undefined;
    /**
     * Decides where title should be on desktop. Options:
     * - Hide: Title is hidden
     * - ShowUnder: Title is shown under the navbar (and above the `below` content)
     * - ShowIn: Title is shown in the navbar, the same as on mobile
     */
    titleBehaviorDesktop?: "Hide" | "ShowBelow" | "ShowIn";
    /**
     * Decides where title should be on mobile. Options:
     * - Hide: Title is hidden
     * - ShowIn: Title is shown in the navbar
     */
    titleBehaviorMobile?: "Hide" | "ShowIn";
    /** Replaces title if provided */
    titleComponent?: JSX.Element;
    sxs?: {
        appBar?: SxType,
        root?: SxType,
    }
}

export type SettingsTopBarProps = TopBarProps

export interface TopBarProps extends TitleProps {
    below?: JSX.Element | boolean;
    display: ViewDisplayType;
    keepVisible?: boolean;
    onClose?: () => unknown;
    startComponent?: JSX.Element;
    tabTitle?: string;
    titleBehaviorDesktop?: NavbarProps["titleBehaviorDesktop"];
    titleBehaviorMobile?: NavbarProps["titleBehaviorMobile"];
    titleId?: string;
    sxsNavbar?: NavbarProps["sxs"];
}
