import { CommonProps } from 'types';

export type BottomNavProps = Pick<CommonProps, 'session'>

export interface ContactInfoProps {
    className?: string;
}

export type HamburgerProps = Pick<CommonProps, 'session'>

export type NavbarProps = Pick<CommonProps, 'session'>

export type NavListProps = Pick<CommonProps, 'session'>

export interface TabPanelProps {
    children?: React.ReactNode[] | React.ReactNode;
    index: number;
    value: number;
}

export interface HideOnScrollProps {
    target?: any;
    children: JSX.Element;
}