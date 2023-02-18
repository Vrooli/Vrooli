import { Resource, Session } from "@shared/consts";
import { LineGraphProps } from "components/graphs/types";

export interface ResourceCardProps {
    canUpdate: boolean;
    data: Resource;
    index: number;
    onContextMenu: (target: EventTarget, index: number) => void;
    onEdit: (index: number) => void;
    onDelete: (index: number) => void;
    session: Session;
}

export interface LineGraphCardProps extends Omit<LineGraphProps, 'dims'> {
    title?: string;
    index: number;
}