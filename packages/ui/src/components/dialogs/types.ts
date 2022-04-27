import { MouseEvent } from 'react';
import { DialogProps } from '@mui/material';
import { HelpButtonProps } from "components/buttons/types";
import { SvgIconComponent } from '@mui/icons-material';
import { ReportFor } from '@local/shared';
import { Node, NodeLink, Organization, Project, Resource, Routine, RoutineStep, Session, Standard, User } from 'types';

export interface AlertDialogProps extends DialogProps { };

export interface BaseObjectDialogProps extends DialogProps {
    children: JSX.Element | JSX.Element[];
    hasNext?: boolean;
    hasPrevious?: boolean;
    /**
     * Callback when option button or close button is pressed
     */
    onAction: (state: ObjectDialogAction) => any;
    open: boolean;
    title: string;
};

export interface DeleteRoutineDialogProps {
    handleClose: () => any;
    handleDelete: () => any;
    isOpen: boolean;
    routineName: string;
}

export interface FormDialogProps {
    children: JSX.Element;
    maxWidth?: string | number;
    onClose: () => void;
    title: string;
}

export interface ListMenuItemData<T> {
    /**
     * Displays help button with data
     */
    helpData?: HelpButtonProps;
    /**
     * Icon to display
     */
    Icon?: SvgIconComponent;
    /**
     * Color of Icon, if different than text
     */
    iconColor?: string;
    /**
     * Text to display
     */
    label: string;
    /**
     * Determines if the item is a preview (i.e. not selectable, coming soon)
     */
    preview?: boolean; // Determines if the item is a preview (i.e. not selectable, coming soon)
    /**
     * Value to pass back when selected
     */
    value: T;
}
export interface ListMenuProps<T> {
    anchorEl: HTMLElement | null;
    data?: ListMenuItemData<T>[];
    id: string;
    onSelect: (value: T) => void;
    onClose: () => void;
    title?: string;
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

export interface OrganizationDialogProps {
    /**
     * Can only edit if you own the object
     */
    canEdit?: boolean;
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Organization>;
    session: Session;
};

export interface ProjectDialogProps {
    /**
     * Can only edit if you own the object
     */
    canEdit?: boolean;
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Project>;
    session: Session;
};

export interface ReportDialogProps extends DialogProps {
    forId: string;
    onClose: () => any;
    open: boolean;
    reportFor: ReportFor;
    session: Session;
    title?: string;
}

export interface ResourceDialogProps extends DialogProps {
    /**
     * Index in resource list. -1 if new
     */
    index: number;
    listId: string;
    /**
     * Determines if add resource should be called by this dialog, or is handled later
     */
    mutate: boolean;
    onClose: () => any;
    onCreated: (resource: Resource) => any;
    open: boolean;
    onUpdated: (index: number, resource: Resource) => any;
    partialData?: Partial<Resource>;
    session: Session;
}

export interface RoutineDialogProps {
    /**
     * Can only edit if you own the object
     */
    canEdit?: boolean;
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Routine>;
    session: Session;
};

export interface ShareDialogProps extends DialogProps {
    open: boolean;
    onClose: () => any;
}

export interface StandardDialogProps {
    /**
     * Can only edit if you own the object
     */
    canEdit?: boolean;
    hasPrevious?: boolean;
    hasNext?: boolean;
    partialData?: Partial<Standard>;
    session: Session;
};

export interface UserDialogProps {
    /**
     * Can only edit if you own the object
     */
    canEdit?: boolean;
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
    Stats = "Stats",
    Unstar = "Unstar",
    Update = "Update", // Not a synonym for edit. Used when COMPLETING an edit
    UpdateCancel = "UpdateCancel",
    Upvote = "Upvote",
}

export interface BaseObjectActionDialogProps {
    anchorEl: HTMLElement | null;
    availableOptions: BaseObjectAction[];
    handleActionComplete: (action: BaseObjectAction, data: any) => any;
    handleDelete: () => any;
    handleEdit: () => any;
    objectId: string;
    objectType: string;
    onClose: () => any;
    session: Session;
    title: string;
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
    handleAction: (action: BaseObjectAction) => any;
    handleUpdate: (routine: Routine) => any;
    isEditing: boolean;
    language: string;
    routine: Routine | null;
    session: Session;
    sxs?: { icon: any };
}

export interface SubroutineInfoDialogProps {
    handleUpdate: (updatedSubroutine: Routine) => any;
    handleViewFull: () => any;
    isEditing: boolean;
    language: string;
    open: boolean;
    subroutine: Routine | null;
    onClose: () => any;
}

export interface UnlinkedNodesDialogProps {
    handleNodeDelete: (nodeId: string) => any;
    /**
     * Expand/shrink dialog
     */
    handleToggleOpen: () => any;
    language: string;
    nodes: Node[];
    open: boolean;
}

export interface CreateNewDialogProps {
    handleClose: () => any;
    isOpen: boolean;
}

export interface RunStepsDialogProps {
    handleLoadSubroutine: (id: string) => any;
    handleStepParamsUpdate: (step: number[]) => any;
    history: Array<number>[];
    /**
     * Out of 100
     */
    percentComplete: number;
    routineId: string | null | undefined;
    stepList: RoutineStep | null;
    sxs?: { icon: any };
}

export interface AddStandardDialogProps {
    handleAdd: (standard: Standard) => any;
    handleClose: () => any;
    isOpen: boolean;
    session: Session;
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

export interface SelectLanguageDialogProps {
    /**
     * Languages to restrict selection to
     */
    availableLanguages?: string[];
    canDelete?: boolean;
    canDropdownOpen?: boolean;
    color?: string;
    handleDelete?: () => any;
    /**
     * Callback when language is selected
     */
    handleSelect: (language: string) => any;
    /**
     * Selected language
     */
    language: string;
    onClick?: (event: MouseEvent<HTMLDivElement>) => any;
    /**
     * Contains user's languages. These are displayed at the top of the language selection list
     */
    session: Session;
    sxs?: { root: any };
}

export interface AdvancedSearchDialogProps {
    handleClose: () => any;
    handleSearch: (searchQuery: { [x: string]: any }) => any;
    isOpen: boolean;
    session: Session;
}