import { Resource, ResourceList, ResourceListFor } from "@local/shared";
import { UsePressEvent } from "hooks/gestures";
import { DraggableProvidedDragHandleProps, DraggableProvidedDraggableProps } from "react-beautiful-dnd";
import { SxType } from "types";
import { ObjectListActions } from "../types";

export interface ResourceCardProps {
    data: Resource;
    dragProps: DraggableProvidedDraggableProps;
    dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
    /** 
     * Hides edit and delete icons when in edit mode, 
     * making only drag'n'drop and the context menu available.
     **/
    isEditing: boolean;
    onContextMenu: (target: EventTarget, data: Resource) => unknown;
    onEdit: (data: Resource) => unknown;
    onDelete: (data: Resource) => unknown;
}

export type ResourceListProps = {
    title?: string;
    canUpdate?: boolean;
    handleUpdate?: (updatedList: ResourceList) => unknown;
    horizontal?: boolean;
    id?: string;
    list: ResourceList | null | undefined;
    loading?: boolean;
    mutate?: boolean;
    parent: { __typename: ResourceListFor | `${ResourceListFor}`, id: string };
    sxs?: { list?: SxType };
}

export interface ResourceListItemProps {
    canUpdate: boolean;
    data: Resource;
    handleContextMenu: (event: UsePressEvent, index: number) => unknown;
    handleEdit: (index: number) => unknown;
    handleDelete: (index: number) => unknown;
    index: number;
    loading: boolean;
}

export type ResourceListHorizontalProps = ResourceListProps & {
    handleToggleSelect: (data: Resource) => unknown;
    isEditing: boolean;
    isSelecting: boolean;
    onAction: (action: keyof ObjectListActions<Resource>, ...data: unknown[]) => unknown;
    onClick: (data: Resource) => unknown;
    onDelete: (data: Resource) => unknown;
    openAddDialog: () => unknown;
    openUpdateDialog: (data: Resource) => unknown;
    selectedData: Resource[];
}

export type ResourceListVerticalProps = ResourceListHorizontalProps
