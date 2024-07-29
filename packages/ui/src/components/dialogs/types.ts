import { ApiVersion, Bookmark, BookmarkFor, CodeVersion, CommonKey, FocusMode, ListObject, Meeting, Node, NodeLinkShape, NodeRoutineList, NodeRoutineListItem, NodeRoutineListItemShape, NodeShape, NoteVersion, ProjectVersion, Question, RootStep, RoutineVersion, StandardVersion, Team, User } from "@local/shared";
import { DialogProps, PopoverProps } from "@mui/material";
import { HelpButtonProps } from "components/buttons/types";
import { TitleProps } from "components/text/types";
import { UseObjectActionsReturn } from "hooks/useObjectActions";
import { ReactNode } from "react";
import { SvgComponent, SxType, ViewDisplayType } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { CookiePreferences } from "utils/cookies";
import { FormProps } from "../../types";
import { FindObjectTabOption } from "./FindObjectDialog/FindObjectDialog";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface SideMenuProps { }

export interface BulkDeleteDialogProps {
    handleClose: (selectedForDelete: ListObject[]) => unknown;
    isOpen: boolean;
    selectedData: ListObject[];
}

export interface CookieSettingsDialogProps {
    handleClose: (preferences?: CookiePreferences) => unknown;
    isOpen: boolean;
}

export interface DeleteAccountDialogProps {
    handleClose: (wasDeleted: boolean) => unknown;
    isOpen: boolean;
}

export interface DeleteDialogProps {
    handleClose: (wasDeleted: boolean) => unknown;
    handleDelete: () => unknown;
    isOpen: boolean;
    objectName: string;
}

export interface DialogTitleProps extends Omit<TitleProps, "sxs"> {
    below?: JSX.Element | boolean | undefined;
    id: string;
    onClose?: () => unknown;
    startComponent?: JSX.Element;
    sxs?: TitleProps["sxs"] & { root?: SxType; };
}

export type SelectOrCreateObjectType = "ApiVersion" |
    "CodeVersion" |
    "FocusMode" |
    "Meeting" |
    "NoteVersion" |
    "ProjectVersion" |
    "Question" |
    "RoutineVersion" |
    "RunProject" |
    "RunRoutine" |
    "StandardVersion" |
    "Team" |
    "User";
export type SelectOrCreateObject = ApiVersion |
    CodeVersion |
    FocusMode |
    Meeting |
    NoteVersion |
    ProjectVersion |
    Question |
    RoutineVersion |
    StandardVersion |
    Team |
    User;
/**
 * Determines what type of data is returned when an object is selected. 
 * "Full" uses an additional request to get the full object. 
 * "List" returns the object as it appears in a findMany query.
 * "Url" returns the url of the object.
 */
export type FindObjectDialogType = "Full" | "List" | "Url";
export interface FindObjectDialogProps<Find extends FindObjectDialogType, ObjectType extends SelectOrCreateObject> {
    /** The type of data returned when an object is selected */
    find: Find;
    handleCancel: () => unknown;
    handleComplete: (data: Find extends "Url" ? string : ObjectType) => unknown;
    isOpen: boolean;
    limitTo?: readonly FindObjectTabOption[];
    /** Forces selection to be a version, and removes unversioned items from limitTo */
    onlyVersioned?: boolean;
    where?: Record<string, any>;
}

export interface FindSubroutineDialogProps {
    handleCancel: () => unknown;
    handleComplete: (nodeId: string, subroutine: RoutineVersion) => unknown;
    isOpen: boolean;
    nodeId: string;
    routineVersionId: string | null | undefined;
}

export interface ListMenuItemData<T> {
    /** Displays help button with data */
    helpData?: HelpButtonProps;
    /** Icon to display */
    Icon?: SvgComponent;
    /** Color of Icon, if different than text */
    iconColor?: string;
    /** Text to display */
    label?: string;
    /** Translation key for label, if label not provided */
    labelKey?: CommonKey;
    /** Value to pass back when selected */
    value: T;
}
export interface ListMenuProps<T> {
    anchorEl: HTMLElement | null;
    data?: ListMenuItemData<T>[];
    id: string;
    onSelect: (value: T) => unknown;
    onClose: () => unknown;
    title?: string;
}

export interface MenuTitleProps extends Omit<TitleProps, "sxs"> {
    ariaLabel?: string;
    onClose: () => unknown;
    sxs?: TitleProps["sxs"] & { root?: SxType; };
}

export enum ObjectDialogAction {
    Add = "Add",
    Cancel = "Cancel",
    Close = "Close",
    Delete = "Delete",
    Edit = "Edit",
    Next = "Next",
    Previous = "Previous",
    Save = "Save",
}

export interface ReorderInputDialogProps {
    handleClose: (toIndex?: number) => unknown;
    isInput: boolean;
    listLength: number;
    startIndex: number;
}

export interface ShareObjectDialogProps extends DialogProps {
    object: ListObject | null | undefined;
    open: boolean;
    onClose: () => unknown;
}

export interface TranscriptDialogProps {
    handleClose: () => unknown;
    isListening: boolean;
    showHint: boolean;
    transcript: string;
}

export type ObjectActionDialogsProps = UseObjectActionsReturn & {
    object: ListObject | null | undefined;
}

export interface ObjectActionMenuProps {
    actionData: UseObjectActionsReturn;
    anchorEl: HTMLElement | null;
    exclude?: ObjectAction[];
    object: ListObject | null | undefined;
    onClose: () => unknown;
}

export interface LinkDialogProps {
    handleClose: (newLink?: NodeLinkShape) => unknown;
    handleDelete: (link: NodeLinkShape) => unknown;
    isAdd: boolean;
    isOpen: boolean;
    language: string; // Language to display/edit
    link?: NodeLinkShape; // Link to display on open, if editing
    nodeFrom?: NodeShape | null; // Initial "from" node
    nodeTo?: NodeShape | null; // Initial "to" node
    routineVersion: Pick<RoutineVersion, "id" | "nodes" | "nodeLinks">;
}

export interface SubroutineCreateDialogProps {
    isOpen: boolean;
    onClose: () => unknown;
}

export type SubroutineFormProps = Omit<FormProps<Node, NodeRoutineListItemShape>, "disabled" | "display" | "existing" | "isLoading" | "isOpen" | "onCancel" | "isReadLoading" | "onClose" | "onCompleted"> & Required<Pick<SubroutineInfoDialogProps, "handleUpdate" | "handleReorder" | "handleViewFull" | "onClose">> & {
    /**
     * True if the routine version itself can be updated. Otherwise, 
     * only node-level properties can be updated (e.g. index)
     */
    canUpdateRoutineVersion: boolean;
    isEditing: boolean;
    isOpen: boolean;
    /** Number of subroutines in parent routine list */
    numSubroutines: number;
    versions: string[];
};

export interface SubroutineInfoDialogProps {
    data: { node: Node & { routineList: NodeRoutineList }, routineItemId: string } | null;
    defaultLanguage: string;
    handleUpdate: (updatedSubroutine: NodeRoutineListItem) => unknown;
    handleReorder: (nodeId: string, oldIndex: number, newIndex: number) => unknown;
    handleViewFull: () => unknown;
    isEditing: boolean;
    open: boolean;
    onClose: () => unknown;
}

export interface UnlinkedNodesDialogProps {
    handleNodeDelete: (nodeId: string) => unknown;
    /** Expand/shrink dialog */
    handleToggleOpen: () => unknown;
    language: string;
    nodes: Node[];
    open: boolean;
}

export interface RunStepsDialogProps {
    currStep: number[] | null;
    handleLoadSubroutine: (id: string) => unknown;
    handleCurrStepLocationUpdate: (step: number[]) => unknown;
    history: number[][];
    /** Out of 100 */
    percentComplete: number;
    rootStep: RootStep | null;
}

export interface DeleteBookmarkListDialogProps {
    bookmarkDeleteOptions: Bookmark[];
    onClose: () => unknown;
    onDelete: (bookmarks: Bookmark[]) => unknown;
}

export interface SelectBookmarkListDialogProps {
    objectId: string | null;
    objectType: BookmarkFor | `${BookmarkFor}`;
    onClose: (inList: boolean) => unknown;
    isCreate: boolean;
    isOpen: boolean;
}

export interface SelectLanguageMenuProps {
    /**
     * While there may be multiple selected languages, 
     * there is only ever one current language
     */
    currentLanguage: string;
    handleDelete?: (language: string) => unknown;
    /** Callback when new current language is selected */
    handleCurrent: (language: string) => unknown;
    isEditing?: boolean;
    /** Languages that currently have a translation */
    languages: string[];
    sxs?: { root: SxType; };
}

export interface LargeDialogProps {
    children: ReactNode;
    id: string;
    isOpen: boolean;
    onClose: (_event: unknown, reason: "backdropClick" | "escapeKeyDown") => unknown;
    titleId?: string;
    sxs?: {
        paper?: SxType;
        root?: SxType;
    };
    zIndexOffset?: number;
}

export interface MaybeLargeDialogProps extends Omit<LargeDialogProps, "isOpen" | "onClose"> {
    display: ViewDisplayType;
    isOpen?: boolean;
    onClose?: () => unknown;
}

export interface WalletInstallDialogProps {
    onClose: () => unknown;
    open: boolean;
}

export interface WalletSelectDialogProps {
    handleOpenInstall: () => unknown;
    onClose: (selectedKey: string | null) => unknown;
    open: boolean;
}

export interface PopoverWithArrowProps extends Omit<PopoverProps, "open" | "sx"> {
    anchorEl: HTMLElement | null;
    children: ReactNode;
    disableScrollLock?: boolean;
    handleClose?: () => unknown;
    placement?: "top" | "right" | "bottom" | "left";
    sxs?: {
        root?: Record<string, unknown>;
        content?: SxType;
        paper?: SxType;
    }
}


export interface TutorialDialogProps {
    isOpen: boolean;
    onClose: () => unknown;
}
