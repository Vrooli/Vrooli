import { ResourceList } from "types";

export interface ResourceListHorizontalProps {
    title?: string;
    canEdit?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null;
    session: Session;
    mutate?: boolean;
}

export interface ResourceListVerticalProps {
    title?: string;
    canEdit?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null;
    session: session
    mutate?: boolean;
}

export interface ResourceListItemProps {
    session: Session;
    index: number;
    data: Resource;
    onClick?: (e: any, data: any) => void;
}

export interface ResourceListItemContextMenuProps {
    id: string;
    anchorEl: HTMLElement | null;
    index: number | null;
    onClose: () => void;
    onAddBefore: (index: number) => void;
    onAddAfter: (index: number) => void;
    onEdit: (index: number) => void;
    onDelete: (index: number) => void;
    onMove: (index: number) => void;
}