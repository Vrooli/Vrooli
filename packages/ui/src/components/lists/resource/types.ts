import { Resource, ResourceList, ResourceListFor } from "@local/shared";
import { DraggableProvidedDraggableProps, DraggableProvidedDragHandleProps } from "react-beautiful-dnd";

export interface ResourceCardProps {
    canUpdate: boolean;
    data: Resource;
    dragProps: DraggableProvidedDraggableProps;
    dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
    index: number;
    onContextMenu: (target: EventTarget, index: number) => void;
    onEdit: (index: number) => void;
    onDelete: (index: number) => void;
}

export interface ResourceListHorizontalProps {
    title?: string;
    canUpdate?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    id?: string;
    list: ResourceList | null | undefined;
    loading?: boolean;
    mutate?: boolean;
    parent: { __typename: ResourceListFor | `${ResourceListFor}`, id: string };
}

export interface ResourceListVerticalProps {
    canUpdate?: boolean;
    handleUpdate?: (updatedList: ResourceList) => void;
    list: ResourceList | null | undefined;
    loading: boolean;
    mutate: boolean;
    parent: { __typename: ResourceListFor | `${ResourceListFor}`, id: string };
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
}
