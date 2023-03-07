import { Session } from '@shared/consts';
import { CommonProps, OptionalTranslation } from 'types';
import { ViewDisplayType } from 'views/types';

export type BottomNavProps = CommonProps;

export type CommandPaletteProps = CommonProps;

export interface ContactInfoProps extends CommonProps {
    sx?: { [key: string]: any };
}

export type FindInPageProps = CommonProps;

export type NavbarProps = CommonProps & {
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

export type NavListProps = CommonProps;

export interface HideOnScrollProps {
    target?: any;
    children: JSX.Element;
}

export interface SettingsTopBarProps extends Omit<TopBarProps, 'below'> {}

export interface TopBarProps {
    display: ViewDisplayType
    onClose: () => void,
    session?: Session,
    titleData?: OptionalTranslation & { hideOnDesktop?: boolean },
    below?: JSX.Element | boolean
}