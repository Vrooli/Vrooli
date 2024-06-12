import { ProjectVersionDirectory } from "@local/shared";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { CodeVersionShape } from "utils/shape/models/codeVersion";
import { NoteVersionShape } from "utils/shape/models/noteVersion";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { StandardVersionShape } from "utils/shape/models/standardVersion";
import { TeamShape } from "utils/shape/models/team";
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
