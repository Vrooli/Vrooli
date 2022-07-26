import { CommonProps } from 'types';

export type BottomNavProps = Pick<CommonProps, 'session'>

export interface ContactInfoProps {
    className?: string;
}

export type NavbarProps = Pick<CommonProps, 'session' | 'sessionChecked'>

export type NavListProps = Pick<CommonProps, 'session' | 'sessionChecked'>

export interface HideOnScrollProps {
    target?: any;
    children: JSX.Element;
}