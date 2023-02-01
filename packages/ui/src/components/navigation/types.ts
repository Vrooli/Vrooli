import { CommonProps } from 'types';

export type BottomNavProps = Pick<CommonProps, 'session'>

export type CommandPaletteProps = Pick<CommonProps, 'session'>

export interface ContactInfoProps extends Pick<CommonProps, 'session'> {
    sx?: { [key: string]: any };
}

export type FindInPageProps = Pick<CommonProps, 'session'>

export type FooterProps = Pick<CommonProps, 'session'>

export type NavbarProps = Pick<CommonProps, 'session' | 'sessionChecked'>

export type NavListProps = Pick<CommonProps, 'session' | 'sessionChecked'>

export interface HideOnScrollProps {
    target?: any;
    children: JSX.Element;
}