import { ApiVersion, Bookmark, BookmarkFor, Comment, DeleteType, FocusMode, Node, NodeRoutineList, NodeRoutineListItem, NoteVersion, Organization, ProjectVersion, ReportFor, Resource, RoutineVersion, RunProject, RunRoutine, Schedule, SmartContractVersion, StandardVersion, SvgComponent, User } from "@local/shared";
import { DialogProps, PopoverProps } from "@mui/material";
import { HelpButtonProps } from "components/buttons/types";
import { StatsCompactPropsObject } from "components/text/types";
import { BaseObjectFormProps } from "forms/types";
import { NavigableObject, RoutineStep } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { CookiePreferences } from "utils/cookies";
import { ListObjectType } from "utils/display/listTools";
import { UseObjectActionsReturn } from "utils/hooks/useObjectActions";
import { CalendarPageTabOption, SearchType } from "utils/search/objectToSearch";
import { CommentShape } from "utils/shape/models/comment";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { ScheduleShape } from "utils/shape/models/schedule";

export interface AccountMenuProps {
    anchorEl: HTMLElement | null;
    onClose: (event: React.MouseEvent<HTMLElement>) => void;
}

export interface BaseObjectDialogProps extends DialogProps {
    children: JSX.Element | JSX.Element[];
    /**
     * Callback when option button or close button is pressed
     */
    onAction: (state: ObjectDialogAction) => any;
    open: boolean;
    title?: string;
    zIndex: number;
}

export interface CommentDialogProps extends Omit<BaseObjectFormProps<CommentShape>, "display"> {
    parent: Comment | null;
}

export interface CookieSettingsDialogProps {
    handleClose: (preferences?: CookiePreferences) => void;
    isOpen: boolean;
}

export interface DeleteAccountDialogProps {
    handleClose: (wasDeleted: boolean) => void;
    isOpen: boolean;
    zIndex: number;
}

export interface DeleteDialogProps {
    handleClose: (wasDeleted: boolean) => void;
    isOpen: boolean;
    objectId: string;
    objectName: string;
    objectType: DeleteType;
    zIndex: number;
}

export interface DialogTitleProps {
    below?: JSX.Element | boolean | undefined;
    helpText?: string;
    id: string;
    onClose: () => void;
    title?: string;
}

export type SelectOrCreateObjectType = "ApiVersion" | "NoteVersion" | "Organization" | "ProjectVersion" | "RoutineVersion" | "SmartContractVersion" | "StandardVersion" | "User";
export type SelectOrCreateObject = ApiVersion |
    NoteVersion |
    Organization |
    ProjectVersion |
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
    /**
     * Determines the type of data returned when an object is selected
     */
    find: Find;
    handleCancel: () => any;
    handleComplete: (data: Find extends "Url" ? string : ObjectType) => any;
    isOpen: boolean;
    limitTo?: SelectOrCreateObjectType[];
    /**
     * If not set, uses "Popular" endpoint to get data
     */
    searchData?: {
        searchType: SearchType | `${SearchType}`;
        where: { [key: string]: any };
    }
    zIndex: number;
}

export interface FindSubroutineDialogProps {
    handleCancel: () => any;
    handleComplete: (nodeId: string, subroutine: RoutineVersion) => any;
    isOpen: boolean;
    nodeId: string;
    routineVersionId: string | null | undefined;
    zIndex: number;
}

export interface FocusModeDialogProps extends Omit<DialogProps, "open"> {
    isCreate: boolean;
    isOpen: boolean;
    onClose: () => any;
    onCreated: (focusMode: FocusMode) => any;
    onUpdated: (focusMode: FocusMode) => any;
    partialData?: Partial<FocusMode>;
    zIndex: number;
}

export interface ListMenuItemData<T> {
    /**
     * Displays help button with data
     */
    helpData?: HelpButtonProps;
    /**
     * Icon to display
     */
    Icon?: SvgComponent;
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
    zIndex: number;
}

export interface MenuTitleProps {
    ariaLabel?: string;
    helpText?: string;
    onClose: () => void;
    title?: string;
}

export enum ObjectDialogAction {
    Add = "Add",
    Cancel = "Cancel",
    Close = "Close",
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
    zIndex: number;
}

export interface ReportDialogProps extends DialogProps {
    forId: string;
    onClose: () => any;
    open: boolean;
    reportFor: ReportFor;
    title?: string;
    zIndex: number;
}

export interface ResourceDialogProps extends Omit<DialogProps, "open"> {
    /**
     * Index in resource list. -1 if new
     */
    index: number;
    isOpen: boolean;
    listId: string;
    /**
     * Determines if add resource should be called by this dialog, or is handled later
     */
    mutate: boolean;
    onClose: () => any;
    onCreated: (resource: Resource) => any;
    onUpdated: (index: number, resource: Resource) => any;
    partialData?: Partial<Resource>;
    zIndex: number;
}

export interface ShareObjectDialogProps extends DialogProps {
    object: NavigableObject | null | undefined;
    open: boolean;
    onClose: () => any;
    zIndex: number;
}

export interface ShareSiteDialogProps extends DialogProps {
    open: boolean;
    onClose: () => any;
    zIndex: number;
}

export interface TranscriptDialogProps {
    handleClose: () => void;
    isListening: boolean;
    transcript: string;
}

export type ObjectActionDialogsProps = UseObjectActionsReturn & {
    object: ListObjectType | null | undefined;
    zIndex: number;
}

export interface ObjectActionMenuProps {
    actionData: UseObjectActionsReturn;
    anchorEl: HTMLElement | null;
    exclude?: ObjectAction[];
    object: ListObjectType | null | undefined;
    onClose: () => any;
    zIndex: number;
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
    zIndex: number;
}

export interface ScheduleDialogProps extends Omit<DialogProps, "open"> {
    defaultTab: CalendarPageTabOption;
    existing?: ScheduleShape;
    isMutate: boolean;
    isOpen: boolean;
    onClose: () => any;
    onCreated: (schedule: Schedule) => any;
    onUpdated: (schedule: Schedule) => any;
    zIndex: number;
}

export interface SubroutineInfoDialogProps {
    data: { node: Node & { routineList: NodeRoutineList }, routineItemId: string } | null;
    defaultLanguage: string;
    handleUpdate: (updatedSubroutine: NodeRoutineListItem) => any;
    handleReorder: (nodeId: string, oldIndex: number, newIndex: number) => any;
    handleViewFull: () => any;
    isEditing: boolean;
    open: boolean;
    onClose: () => any;
    zIndex: number;
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
    zIndex: number;
}

export interface RunStepsDialogProps {
    currStep: number[] | null;
    handleLoadSubroutine: (id: string) => any;
    handleCurrStepLocationUpdate: (step: number[]) => any;
    history: number[][];
    /**
     * Out of 100
     */
    percentComplete: number;
    stepList: RoutineStep | null;
    zIndex: number;
}

export interface DeleteBookmarkListDialogProps {
    bookmarkDeleteOptions: Bookmark[];
    onClose: () => any;
    onDelete: (bookmarks: Bookmark[]) => any;
    zIndex: number;
}

export interface SelectBookmarkListDialogProps {
    objectId: string | null;
    objectType: BookmarkFor | `${BookmarkFor}`;
    onClose: (inList: boolean) => any;
    isCreate: boolean;
    isOpen: boolean;
    zIndex: number;
}

export interface SelectLanguageMenuProps {
    /**
     * While there may be multiple selected languages, 
     * there is only ever one current language
     */
    currentLanguage: string;
    handleDelete?: (language: string) => any;
    /**
     * Callback when new current language is selected
     */
    handleCurrent: (language: string) => any;
    isEditing?: boolean;
    /**
     * Languages that currently have a translation
     */
    languages: string[];
    sxs?: { root: any };
    zIndex: number;
}

export interface StatsDialogProps<T extends StatsCompactPropsObject> {
    handleObjectUpdate: (object: T) => void;
    isOpen: boolean;
    object: T | null | undefined;
    onClose: () => void;
    zIndex: number;
}

export interface AdvancedSearchDialogProps {
    handleClose: () => any;
    handleSearch: (searchQuery: { [x: string]: any }) => any;
    isOpen: boolean;
    searchType: SearchType | `${SearchType}`;
    zIndex: number;
}

export interface RunPickerMenuProps {
    anchorEl: HTMLElement | null;
    handleClose: () => any;
    onAdd: (run: RunProject | RunRoutine) => any;
    onDelete: (run: RunProject | RunRoutine) => any;
    onSelect: (run: RunProject | RunRoutine | null) => any;
    runnableObject?: RoutineVersion | ProjectVersion | null;
}

export interface LargeDialogProps {
    children: JSX.Element | null | undefined | (JSX.Element | null | undefined)[];
    id: string;
    isOpen: boolean;
    onClose: () => any;
    titleId?: string;
    zIndex: number;
    sxs?: {
        paper?: { [x: string]: any };
    }
}

export interface WalletInstallDialogProps {
    onClose: () => any;
    open: boolean;
    zIndex: number;
}

export interface WalletSelectDialogProps {
    handleOpenInstall: () => any;
    onClose: (selectedKey: string | null) => any;
    open: boolean;
    zIndex: number;
}

export interface PopoverWithArrowProps extends Omit<PopoverProps, "open" | "sx"> {
    anchorEl: HTMLElement | null;
    children: React.ReactNode;
    handleClose: () => any;
    sxs?: {
        root?: { [x: string]: any };
        content?: { [x: string]: any };
    }
}

export interface WelcomeDialogProps {
    isOpen: boolean;
    onClose: () => any;
}
