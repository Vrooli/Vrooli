import { Resource } from "@shared/consts";
import { SvgComponent } from "@shared/icons";
import { LineGraphProps } from "components/graphs/types";
import { AwardDisplay } from "types";

export interface AwardCardProps {
    award: AwardDisplay;
}

export interface CardGridProps {
    children: JSX.Element | (JSX.Element | null)[] | null;
    disableMargin?: boolean;
    minWidth: number;
}

export interface LineGraphCardProps extends Omit<LineGraphProps, 'dims'> {
    title?: string;
    index: number;
}

export interface ResourceCardProps {
    canUpdate: boolean;
    data: Resource;
    index: number;
    onContextMenu: (target: EventTarget, index: number) => void;
    onEdit: (index: number) => void;
    onDelete: (index: number) => void;
}

export interface TIDCardProps {
    buttonText: string;
    description: string;
    key: string | number;
    Icon: SvgComponent;
    onClick: (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => void;
    title: string;
}