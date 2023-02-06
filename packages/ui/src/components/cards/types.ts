import { Resource, Session } from "@shared/consts";

export interface ResourceCardProps {
    canUpdate: boolean;
    data: Resource;
    index: number;
    onContextMenu: (target: EventTarget, index: number) => void;
    onEdit: (index: number) => void;
    onDelete: (index: number) => void;
    session: Session;
}

export interface StatCardProps {
    title?: string;
    data: any;
    index: number;
}