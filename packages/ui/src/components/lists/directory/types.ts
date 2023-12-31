import { ProjectVersionDirectory } from "@local/shared";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { NoteVersionShape } from "utils/shape/models/noteVersion";
import { OrganizationShape } from "utils/shape/models/organization";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { StandardVersionShape } from "utils/shape/models/standardVersion";

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
    index: number;
    onContextMenu: (target: EventTarget, index: number) => unknown;
    onDelete: (index: number) => unknown;
}

export interface DirectoryListProps {
    canUpdate?: boolean;
    handleUpdate?: (updatedDirectory: ProjectVersionDirectory) => unknown;
    directory: ProjectVersionDirectory | null;
    loading?: boolean;
    mutate?: boolean;
    sortBy: DirectoryListSortBy;
}

export interface DirectoryListItemProps {
    canUpdate: boolean;
    data: DirectoryItem;
    handleContextMenu: (target: EventTarget, index: number) => unknown;
    handleEdit: (index: number) => unknown;
    handleDelete: (index: number) => unknown;
    index: number;
    loading: boolean;
}

export interface DirectoryListItemContextMenuProps {
    canUpdate: boolean;
    id: string;
    anchorEl: HTMLElement | null;
    index: number | null;
    onClose: () => unknown;
    onDelete: (index: number) => unknown;
    data: DirectoryItem | null;
}
