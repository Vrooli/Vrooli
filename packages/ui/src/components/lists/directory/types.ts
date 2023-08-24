import { ProjectVersionDirectory } from "@local/shared";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { NoteVersionShape } from "utils/shape/models/noteVersion";
import { OrganizationShape } from "utils/shape/models/organization";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { StandardVersionShape } from "utils/shape/models/standardVersion";

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
    onContextMenu: (target: EventTarget, index: number) => void;
    onDelete: (index: number) => void;
}

export interface DirectoryListHorizontalProps {
    canUpdate?: boolean;
    handleUpdate?: (updatedDirectory: ProjectVersionDirectory) => void;
    directory: ProjectVersionDirectory | null;
    loading?: boolean;
    mutate?: boolean;
}

export interface DirectoryListVerticalProps {
    canUpdate?: boolean;
    handleUpdate?: (updatedDirectory: ProjectVersionDirectory) => void;
    directory: ProjectVersionDirectory | null | undefined;
    loading: boolean;
    mutate: boolean;
}

export interface DirectoryListItemProps {
    canUpdate: boolean;
    data: DirectoryItem;
    handleContextMenu: (target: EventTarget, index: number) => void;
    handleEdit: (index: number) => void;
    handleDelete: (index: number) => void;
    index: number;
    loading: boolean;
}

export interface DirectoryListItemContextMenuProps {
    canUpdate: boolean;
    id: string;
    anchorEl: HTMLElement | null;
    index: number | null;
    onClose: () => void;
    onDelete: (index: number) => void;
    data: DirectoryItem | null;
}
