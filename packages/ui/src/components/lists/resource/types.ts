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
