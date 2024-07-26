import { ApiVersionShape, CodeVersionShape, NoteVersionShape, ProjectVersionDirectory, ProjectVersionShape, RoutineVersionShape, StandardVersionShape, TeamShape } from "@local/shared";
import { ObjectListActions } from "../types";

export type DirectoryItem = ApiVersionShape |
    CodeVersionShape |
    NoteVersionShape |
    ProjectVersionShape |
    RoutineVersionShape |
    StandardVersionShape |
    TeamShape;

export interface DirectoryCardProps {
    canUpdate: boolean;
    data: DirectoryItem;
    onContextMenu: (target: EventTarget, data: DirectoryItem) => unknown;
    onDelete: (data: DirectoryItem) => unknown;
}

export interface DirectoryListProps {
    canUpdate?: boolean;
    handleUpdate?: (updatedDirectory: ProjectVersionDirectory) => unknown;
    directory: ProjectVersionDirectory | null;
    loading?: boolean;
    mutate?: boolean;
}

export type DirectoryListHorizontalProps = DirectoryListProps & {
    handleToggleSelect: (data: DirectoryItem) => unknown;
    isEditing: boolean;
    isSelecting: boolean;
    list: DirectoryItem[];
    onAction: (action: keyof ObjectListActions<DirectoryItem>, ...data: unknown[]) => unknown;
    onClick: (data: DirectoryItem) => unknown;
    onDelete: (data: DirectoryItem) => unknown;
    openAddDialog: () => unknown;
    selectedData: DirectoryItem[];
}

export type DirectoryListVerticalProps = DirectoryListHorizontalProps
