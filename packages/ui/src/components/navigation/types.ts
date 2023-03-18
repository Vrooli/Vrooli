import { OptionalTranslation } from 'types';
import { ViewDisplayType } from 'views/types';

export interface ContactInfoProps {
    sx?: { [key: string]: any };
}

export type NavbarProps = {
    title?: string | undefined;
    help?: string | undefined;
    below?: JSX.Element | boolean | undefined;
    shouldHideTitle?: boolean;
}

export type NavbarLogoState = 'full' | 'icon' | 'none';
export interface NavbarLogoProps {
    onClick: () => void;
    state: NavbarLogoState;
}

export interface HideOnScrollProps {
    target?: any;
    children: JSX.Element;
}

export interface SettingsTopBarProps extends Omit<TopBarProps, 'below'> { }

export interface TopBarProps {
    display: ViewDisplayType
    onClose: () => void,
    titleData?: OptionalTranslation & { hideOnDesktop?: boolean },
    below?: JSX.Element | boolean
}