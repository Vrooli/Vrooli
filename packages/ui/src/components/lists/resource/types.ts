import { Resource, ResourceList, ResourceListFor } from "@local/shared";
import { DraggableProvidedDraggableProps, DraggableProvidedDragHandleProps } from "react-beautiful-dnd";

export interface ResourceCardProps {
    data: Resource;
    dragProps: DraggableProvidedDraggableProps;
    dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
    /** 
     * Hides edit and delete icons when in edit mode, 
     * making only drag'n'drop and the context menu available.
     **/
    index: number;
    isEditing: boolean;
    onContextMenu: (target: EventTarget, index: number) => unknown;
    onEdit: (index: number) => unknown;
    onDelete: (index: number) => unknown;
}

export interface ResourceListHorizontalProps {
    title?: string;
    canUpdate?: boolean;
    handleUpdate?: (updatedList: ResourceList) => unknown;
    id?: string;
    list: ResourceList | null | undefined;
    loading?: boolean;
    mutate?: boolean;
    parent: { __typename: ResourceListFor | `${ResourceListFor}`, id: string };
}

export interface ResourceListVerticalProps {
    canUpdate?: boolean;
    handleUpdate?: (updatedList: ResourceList) => unknown;
    list: ResourceList | null | undefined;
    loading: boolean;
    mutate: boolean;
    parent: { __typename: ResourceListFor | `${ResourceListFor}`, id: string };
}

export interface ResourceListItemProps {
    canUpdate: boolean;
    data: Resource;
    handleContextMenu: (target: EventTarget, index: number) => unknown;
    handleEdit: (index: number) => unknown;
    handleDelete: (index: number) => unknown;
    index: number;
    loading: boolean;
}

export interface ResourceListItemContextMenuProps {
    canUpdate: boolean;
    id: string;
    anchorEl: HTMLElement | null;
    index: number | null;
    onClose: () => unknown;
    onAddBefore: (index: number) => unknown;
    onAddAfter: (index: number) => unknown;
    onEdit: (index: number) => unknown;
    onDelete: (index: number) => unknown;
    onMove: (index: number) => unknown;
    resource: Resource | null;
}
