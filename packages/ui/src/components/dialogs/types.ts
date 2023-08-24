import { ApiVersion, Bookmark, BookmarkFor, Comment, CommonKey, DeleteType, FocusMode, Meeting, Node, NodeRoutineList, NodeRoutineListItem, NoteVersion, Organization, ProjectVersion, Question, RoutineVersion, RunProject, RunRoutine, SmartContractVersion, StandardVersion, User } from "@local/shared";
import { DialogProps, PopoverProps } from "@mui/material";
import { HelpButtonProps } from "components/buttons/types";
import { TitleProps } from "components/text/types";
import { BaseObjectFormProps } from "forms/types";
import { UseObjectActionsReturn } from "hooks/useObjectActions";
import { ReactNode } from "react";
import { DirectoryStep, NavigableObject, RoutineListStep, SvgComponent, SxType } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { CookiePreferences } from "utils/cookies";
import { ListObject } from "utils/display/listTools";
import { CommentShape } from "utils/shape/models/comment";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { ViewDisplayType } from "views/types";
import { FindObjectTabOption } from "./FindObjectDialog/FindObjectDialog";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface SideMenuProps { }

export interface CommentDialogProps extends Omit<BaseObjectFormProps<CommentShape>, "display"> {
    parent: Comment | null;
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
    isOpen: boolean;
    objectId: string;
    objectName: string;
    objectType: DeleteType;
}

export interface DialogTitleProps extends Omit<TitleProps, "sxs"> {
    below?: JSX.Element | boolean | undefined;
    id: string;
    onClose?: () => unknown;
    sxs?: TitleProps["sxs"] & { root?: SxType; };
}

export type SelectOrCreateObjectType = "ApiVersion" |
    "FocusMode" |
    "Meeting" |
    "NoteVersion" |
    "Organization" |
    "ProjectVersion" |
    "Question" |
    "RoutineVersion" |
    "RunProject" |
    "RunRoutine" |
    "SmartContractVersion" |
    "StandardVersion" |
    "User";
export type SelectOrCreateObject = ApiVersion |
    FocusMode |
    Meeting |
    NoteVersion |
    Organization |
    ProjectVersion |
    Question |
    RoutineVersion |
    SmartContractVersion |
    StandardVersion |
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
    limitTo?: FindObjectTabOption[];
    /** Forces selection to be a version, and removes unversioned items from limitTo */
    onlyVersioned?: boolean;
    where?: { [key: string]: object };
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
    handleClose: (toIndex?: number) => void;
    isInput: boolean;
    listLength: number;
    startIndex: number;
}

export interface ShareObjectDialogProps extends DialogProps {
    object: NavigableObject | null | undefined;
    open: boolean;
    onClose: () => unknown;
}

export interface TranscriptDialogProps {
    handleClose: () => void;
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
    handleClose: (newLink?: NodeLinkShape) => void;
    handleDelete: (link: NodeLinkShape) => void;
    isAdd: boolean;
    isOpen: boolean;
    language: string; // Language to display/edit
    link?: NodeLinkShape; // Link to display on open, if editing
    nodeFrom?: NodeShape | null; // Initial "from" node
    nodeTo?: NodeShape | null; // Initial "to" node
    routineVersion: Pick<RoutineVersion, "id" | "nodes" | "nodeLinks">;
}

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
    rootStep: RoutineListStep | DirectoryStep | null;
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

export interface RunPickerMenuProps {
    anchorEl: HTMLElement | null;
    handleClose: () => unknown;
    onAdd: (run: RunProject | RunRoutine) => unknown;
    onDelete: (run: RunProject | RunRoutine) => unknown;
    onSelect: (run: RunProject | RunRoutine | null) => unknown;
    runnableObject?: Partial<RoutineVersion | ProjectVersion> | null;
}

export interface LargeDialogProps {
    children: ReactNode;
    id: string;
    isOpen: boolean;
    onClose: () => unknown;
    titleId?: string;
    sxs?: { paper?: SxType; }
}

export interface MaybeLargeDialogProps extends Omit<LargeDialogProps, "onClose"> {
    display: ViewDisplayType;
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
