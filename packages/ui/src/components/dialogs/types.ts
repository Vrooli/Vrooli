import { DialogProps, PopoverProps } from '@mui/material';
import { HelpButtonProps } from "components/buttons/types";
import { Api, Comment, DeleteType, Node, NodeRoutineList, NodeRoutineListItem, Organization, Project, ProjectVersion, Quiz, ReportFor, Resource, Routine, RoutineVersion, RunProject, RunRoutine, Session, SmartContract, Standard, User } from '@shared/consts';
import { NavigableObject, RoutineStep } from 'types';
import { ListObjectType, NodeLinkShape, NodeShape, SearchType, UseObjectActionsReturn } from 'utils';
import { SvgComponent } from '@shared/icons';
import { ObjectAction } from 'utils/actions/objectActions';
import { CookiePreferences } from 'utils/cookies';
import { StatsCompactPropsObject } from 'components/text/types';

export interface AccountMenuProps {
    anchorEl: HTMLElement | null;
    onClose: (event: React.MouseEvent<HTMLElement>) => void;
    session: Session;
}

export interface AlertDialogProps {
    session: Session | undefined;
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
};

export interface CommentDialogProps {
    errorText: string;
    handleSubmit: () => void;
    handleClose: () => void;
    isAdding: boolean;
    isOpen: boolean;
    language: string;
    onTranslationChange: (e: { target: { name: string; value: string; }; }) => void;
    parent: Comment | null;
    session: Session;
    text: string;
    zIndex: number;
}

export interface CookieSettingsDialogProps {
    handleClose: (preferences?: CookiePreferences) => void;
    isOpen: boolean;
};

export interface DeleteAccountDialogProps {
    handleClose: (wasDeleted: boolean) => void;
    isOpen: boolean;
    session: Session;
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
    ariaLabel: string;
    helpText?: string;
    onClose: () => void;
    title?: string;
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
    Add = 'Add',
    Cancel = 'Cancel',
    Close = 'Close',
    Edit = 'Edit',
    Next = 'Next',
    Previous = 'Previous',
    Save = 'Save',
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
    session: Session;
    title?: string;
    zIndex: number;
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
    lng: string;
    transcript: string;
}

export type ObjectActionDialogsProps = UseObjectActionsReturn & {
    object: ListObjectType | null | undefined;
    session: Session;
    zIndex: number;
}

export interface ObjectActionMenuProps {
    actionData: UseObjectActionsReturn;
    anchorEl: HTMLElement | null;
    exclude?: ObjectAction[];
    object: ListObjectType | null | undefined;
    onClose: () => any;
    session: Session;
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
    routineVersion: Pick<RoutineVersion, 'id' | 'nodes' | 'nodeLinks'>;
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
    session: Session;
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
     * Contains user's languages. These are displayed at the top of the language selection list
     */
    session: Session;
    /**
     * Available translations
     */
    translations: { language: string }[];
    sxs?: { root: any };
    zIndex: number;
}

export interface StatsDialogProps<T extends StatsCompactPropsObject> {
    handleObjectUpdate: (object: T) => void;
    isOpen: boolean;
    object: T | null | undefined;
    onClose: () => void;
    session: Session;
    zIndex: number;
}

export interface AdvancedSearchDialogProps {
    handleClose: () => any;
    handleSearch: (searchQuery: { [x: string]: any }) => any;
    isOpen: boolean;
    searchType: SearchType | `${SearchType}`;
    session: Session;
    zIndex: number;
}

export interface RunPickerMenuProps {
    anchorEl: HTMLElement | null;
    handleClose: () => any;
    onAdd: (run: RunProject | RunRoutine) => any;
    onDelete: (run: RunProject | RunRoutine) => any;
    onSelect: (run: RunProject | RunRoutine | null) => any;
    runnableObject?: RoutineVersion | ProjectVersion | null;
    session: Session;
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

export interface PopoverWithArrowProps extends Omit<PopoverProps, 'open' | 'sx'> {
    anchorEl: HTMLElement | null;
    children: React.ReactNode;
    handleClose: () => any;
    sxs?: {
        root?: { [x: string]: any };
        content?: { [x: string]: any };
    }
}