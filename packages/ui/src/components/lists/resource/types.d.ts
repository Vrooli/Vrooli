import { ResourceList } from "types";

export interface ResourceListHorizontalProps {
    title?: string;
    canEdit?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null;
    loading?: boolean;
    session: Session;
    mutate?: boolean;
    zIndex: number;
}

export interface ResourceListVerticalProps {
    title?: string;
    canEdit?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null | undefined;
    loading: boolean;
    session: session
    mutate: boolean;
    zIndex: number;
}

export interface ResourceListItemProps {
    canEdit: boolean;
    data: Resource;
    handleEdit: (index: number) => void;
    handleDelete: (index: number) => void;
    index: number;
    loading: boolean;
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
    zIndex: number;
}