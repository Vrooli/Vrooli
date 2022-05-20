import { ResourceList } from "types";

export interface ResourceListHorizontalProps {
    title?: string;
    canEdit?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null;
    loading: boolean;
    session: Session;
    mutate?: boolean;
}

export interface ResourceListVerticalProps {
    title?: string;
    canEdit?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null | undefined;
    loading: boolean;
    session: session
    mutate: boolean;
}

export interface ResourceListItemProps {
    data: Resource;
    index: number;
    loading: boolean;
    onClick?: (e: any, data: any) => void;
    session: Session;
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