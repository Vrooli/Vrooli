import { ProjectVersionDirectory } from "@local/shared";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { NoteVersionShape } from "utils/shape/models/noteVersion";
import { OrganizationShape } from "utils/shape/models/organization";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { StandardVersionShape } from "utils/shape/models/standardVersion";
import { ObjectListActions } from "../types";

export type DirectoryListSortBy = "DateCreatedAsc" | "DateCreatedDesc" | "DateUpdatedAsc" | "DateUpdatedDesc" | "NameAsc" | "NameDesc";

export type DirectoryItem = ApiVersionShape |
    NoteVersionShape |
    OrganizationShape |
    ProjectVersionShape |
    RoutineVersionShape |
    SmartContractVersionShape |
    StandardVersionShape;

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
    sortBy: DirectoryListSortBy;
    title?: string;
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
