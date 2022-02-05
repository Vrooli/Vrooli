import { CommonProps } from 'types';

export type BottomNavProps = Pick<CommonProps, 'userRoles'>

export interface ContactInfoProps {
    className?: string;
}

export type HamburgerProps = Pick<CommonProps, 'userRoles'>

export type NavbarProps = Pick<CommonProps, 'userRoles'>

export type NavListProps = Pick<CommonProps, 'userRoles'>

export interface TabPanelProps {
    children?: React.ReactNode[] | React.ReactNode;
    index: number;
    value: number;
}

export interface HideOnScrollProps {
    target?: any;
    children: JSX.Element;
}