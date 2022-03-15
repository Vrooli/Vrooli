import { DialogProps } from '@mui/material';
import { HelpButtonProps } from "components/buttons/types";
import { SvgIconComponent } from '@mui/icons-material';
import { ReportFor, ResourceFor } from '@local/shared';
import { Node, NodeLink, Organization, Project, Resource, Routine, RoutineStep, Session, Standard, User } from 'types';

export interface AlertDialogProps extends DialogProps { };

export interface FormDialogProps {
    title: string;
    children: JSX.Element;
    maxWidth?: string | number;
    onClose: () => void;
}

export interface ListMenuItemData<T> {
    label: string; // Text to display
    value: T; // Value to pass back
    Icon?: SvgIconComponent; // Icon to display
    iconColor?: string; // Color of icon, if different than text
    helpData?: HelpButtonProps; // If set, displays help button with data
}
export interface ListMenuProps<T> {
    id: string;
    anchorEl: HTMLElement | null;
    onSelect: (value: T) => void;
    onClose: () => void;
    title?: string;
    data?: ListMenuItemData<T>[];
}

export enum ObjectDialogState {
    Add = 'Add',
    Edit = 'Edit',
    View = 'View',
}
export enum ObjectDialogAction {
    Add = 'Add',
    Cancel = 'Cancel',
    Close = 'Close',
    Edit = 'Edit',
    Next = 'Next',
    Previous = 'Previous',
    Save = 'Save',
}

export interface BaseObjectDialogProps extends DialogProps {
    title: string;
    open: boolean;
    hasPrevious?: boolean;
    hasNext?: boolean;
    onAction: (state: ObjectDialogAction) => any; // Callback when option button or close button is pressed
    children: JSX.Element | JSX.Element[];
};

export interface DeleteRoutineDialogProps {
    handleClose: () => any;
    handleDelete: () => any;
    isOpen: boolean;
    routineName: string;
}

export interface ShareDialogProps extends DialogProps {
    open: boolean;
    onClose: () => any;
}

export interface ReportDialogProps extends DialogProps {
    open: boolean;
    onClose: () => any;
    title?: string;
    reportFor: ReportFor;
    forId: string;
}

export interface AddResourceDialogProps extends DialogProps {
    mutate?: boolean; // Determines if add resource should be called by this dialog, or is handled later
    open: boolean;
    onClose: () => any;
    onCreated: (resource: Resource) => any;
    title?: string;
    listId: string;
}

export interface OrganizationDialogProps {
    canEdit?: boolean; // Can only edit if you own the object
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Organization>;
    session: Session;
};

export interface ProjectDialogProps {
    canEdit?: boolean; // Can only edit if you own the object
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Project>;
    session: Session;
};

export interface RoutineDialogProps {
    canEdit?: boolean; // Can only edit if you own the object
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Routine>;
    session: Session;
};

export interface StandardDialogProps {
    canEdit?: boolean; // Can only edit if you own the object
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Standard>;
    session: Session;
};

export interface UserDialogProps {
    canEdit?: boolean; // Can only edit if you own the object
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<User>;
    session: Session;
};

/**
 * All available actions an object can possibly have
 */
export enum BaseObjectAction {
    Copy = 'Copy',
    Delete = "Delete",
    Donate = "Donate",
    Downvote = "Downvote",
    Edit = "Edit",
    Fork = "Fork",
    Report = "Report",
    Share = "Share",
    Star = "Star",
    Unstar = "Unstar",
    Upvote = "Upvote",
}

export interface BaseObjectActionDialogProps {
    objectId: string;
    objectType: string;
    title: string;
    anchorEl: HTMLElement | null;
    availableOptions: BaseObjectAction[];
    onClose: () => any;
}

export interface LinkDialogProps {
    handleClose: (newLink?: NodeLink) => void;
    handleDelete: (link: NodeLink) => void;
    isAdd: boolean;
    isOpen: boolean;
    language: string; // Language to display/edit
    link?: NodeLink; // Link to display on open, if editing
    routine: Routine;
}

export interface BuildInfoDialogProps {
    handleUpdate: (routine: Routine) => any;
    handleDelete: () => any;
    isEditing: boolean;
    language: string; // Language to display/edit
    routine: Routine | null;
    session: Session;
    sxs?: { icon: any };
}

export interface SubroutineInfoDialogProps {
    open: boolean;
    language: string; // Language to display/edit
    subroutine: Routine | null;
    onClose: () => any;
}

export interface UnlinkedNodesDialogProps {
    open: boolean;
    nodes: Node[];
    handleNodeDelete: (nodeId: string) => any;
    handleToggleOpen: () => any; // Expand/shrink dialog
}

export interface CreateNewDialogProps {
    handleClose: () => any;
    isOpen: boolean;
}

export interface RunStepsDialogProps {
    handleLoadSubroutine: (id: string) => any;
    handleStepParamsUpdate: (step: number[]) => any;
    history: Array<number>[];
    percentComplete: number; // Out of 100
    routineId: string | null | undefined;
    stepList: RoutineStep | null;
    sxs?: { icon: any };
}

export interface AddSubroutineDialogProps {
    handleAdd: (nodeId: string, subroutine: Routine) => any;
    handleClose: () => any;
    isOpen: boolean;
    language: string;
    nodeId: string;
    routineId: string;
    session: Session;
}