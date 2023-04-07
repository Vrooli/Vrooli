import { Resource, ResourceList } from "@shared/consts";

export interface ResourceListHorizontalProps {
    title?: string;
    canUpdate?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null;
    loading?: boolean;
    mutate?: boolean;
    zIndex: number;
}

export interface ResourceListVerticalProps {
    canUpdate?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null | undefined;
    loading: boolean;
    mutate: boolean;
    zIndex: number;
}

export interface ResourceListItemProps {
    canUpdate: boolean;
    data: Resource;
    handleContextMenu: (target: EventTarget, index: number) => void;
    handleEdit: (index: number) => void;
    handleDelete: (index: number) => void;
    index: number;
    loading: boolean;
}

export interface ResourceListItemContextMenuProps {
    canUpdate: boolean;
    id: string;
    anchorEl: HTMLElement | null;
    index: number | null;
    onClose: () => void;
    onAddBefore: (index: number) => void;
    onAddAfter: (index: number) => void;
    onEdit: (index: number) => void;
    onDelete: (index: number) => void;
    onMove: (index: number) => void;
    resource: Resource | null;
    zIndex: number;
}