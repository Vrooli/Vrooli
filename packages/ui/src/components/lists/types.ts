import { ApiVersionShape, BookmarkList, Chat, ChatInvite, ChatParticipant, CodeVersionShape, FocusMode, ListObject, Meeting, MeetingInvite, Member, MemberInvite, ModelType, NavigableObject, NoteVersionShape, Notification, OrArray, Project, ProjectVersion, ProjectVersionDirectory, ProjectVersionShape, QuestionForType, Reminder, ReminderList, Report, ReportResponse, Role, Routine, RoutineVersion, RoutineVersionShape, RunProject, RunRoutine, SearchType, StandardVersionShape, Tag, Team, TeamShape, TimeFrame, TranslationKeyCommon, User } from "@local/shared";
import { ReactNode } from "react";
import { UsePressEvent } from "../../hooks/gestures.js";
import { type UseObjectActionsReturn } from "../../hooks/objectActions.js";
import { type UseFindManyResult } from "../../hooks/useFindMany.js";
import { IconInfo } from "../../icons/Icons.js";
import { SxType, ViewDisplayType } from "../../types.js";
import { ObjectAction } from "../../utils/actions/objectActions.js";
import { RelationshipButtonType } from "../../utils/consts.js";
import { ObjectType } from "../../utils/navigation/openObject.js";
import { LineGraphProps } from "../graphs/types.js";
import { ObjectListProps } from "./ObjectList/ObjectList.js";

export interface ObjectActionsRowProps<T extends ListObject> {
    actionData: UseObjectActionsReturn;
    exclude?: readonly ObjectAction[];
    object: T | null | undefined;
}


export type ActionFunctions<T> = {
    [K in keyof T]: T[K] extends (...args: infer U) => any ? U : never;
};

export type ActionsType<T extends ListObject> = { onAction: <K extends keyof ObjectListActions<T>>(action: K, ...args: ActionFunctions<ObjectListActions<T>>[K]) => unknown }

type ObjectListItemBaseProps<T extends ListObject> = {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    canNavigate?: (item: NavigableObject) => boolean | void;
    belowSubtitle?: React.ReactNode;
    belowTags?: React.ReactNode;
    handleContextMenu: (target: EventTarget, object: ListObject | null) => unknown;
    handleToggleSelect: (item: ListObject, event?: UsePressEvent) => unknown;
    /** If update button should be hidden */
    hideUpdateButton?: boolean;
    isMobile: boolean;
    /** Disables actions and navigation */
    isSelecting: boolean;
    /** Used when isSelecting is true */
    isSelected: boolean;
    /** If data is still being fetched */
    loading: boolean;
    data: T | null;
    objectType: T["__typename"];
    onClick?: (data: T) => unknown;
    subtitleOverride?: string;
    titleOverride?: string;
    toTheRight?: React.ReactNode;
}
export type ObjectListItemProps<T extends ListObject = ListObject> = {
    /** Optional content to display below subtitle */
    belowSubtitle?: ReactNode;
    /** Optional content to display below tags */
    belowTags?: ReactNode;
    /** If function, will validate if we can navigate to the object */
    canNavigate?: (item: T) => boolean;
    /** For ObjectListItemBase, the underlying data to display */
    data: T | null;
    /** Data index for keyboard navigation */
    dataIndex?: number;
    /** Function to handle opening the context menu upon right click */
    handleContextMenu: (target: ReactNode, object: ListObject | null) => void;
    /** Function to toggle item selection when in selection mode */
    handleToggleSelect: (item: T, event?: UsePressEvent) => void;
    /** Hide object's update button */
    hideUpdateButton?: boolean;
    /** Whether to display the mobile or the desktop version */
    isMobile: boolean;
    /** True if ObjectList is in selection mode. Hides actions and allows toggling select */
    isSelecting: boolean;
    /** True if the object is selected */
    isSelected: boolean;
    /** True if the item is a dummy, i.e. we're just displaying a loading state */
    loading: boolean;
    /** Generic callback fired when actions are taken from the context menu */
    onAction?: (action: keyof ObjectListActions<T>, ...data: any[]) => unknown;
    /** Callback fired when the object is clicked */
    onClick?: (item: T) => unknown;
    /** The type of object which determines which component to use */
    objectType: `${ModelType}` | "CalendarEvent";
    /** Optional subtitle override */
    subtitleOverride?: string;
    /** Tab index for keyboard navigation */
    tabIndex?: number;
    /** Optional title override */
    titleOverride?: string;
    /** Optional content to display to the right of the item */
    toTheRight?: ReactNode;
};

export type ObjectListActions<T> = {
    Deleted: (id: string) => unknown;
    Updated: (data: T) => unknown;
};

export type BookmarkListListItemProps = ObjectListItemProps<BookmarkList>
export type ChatListItemProps = ObjectListItemProps<Chat>
export type ChatInviteListItemProps = ObjectListItemProps<ChatInvite>
export type ChatParticipantListItemProps = ObjectListItemProps<ChatParticipant>
export type MeetingInviteListItemProps = ObjectListItemProps<MeetingInvite>
export type MemberInviteListItemProps = ObjectListItemProps<MemberInvite>
export type MemberListItemProps = ObjectListItemProps<Member>
export type NotificationListItemProps = ObjectListItemProps<Notification>
export type ReminderListItemProps = ObjectListItemProps<Reminder>
export type ReportListItemProps = ObjectListItemProps<Report>
export type ReportResponseListItemProps = ObjectListItemProps<ReportResponse>
export type RunProjectListItemProps = ObjectListItemProps<RunProject>
export type RunRoutineListItemProps = ObjectListItemProps<RunRoutine>

export interface DateRangeMenuProps {
    anchorEl: Element | null;
    onClose: () => unknown;
    onSubmit: (after?: Date | undefined, before?: Date | undefined) => unknown;
    minDate?: Date;
    maxDate?: Date;
    range?: { after: Date | undefined, before: Date | undefined };
    /**
     * If set, the date range will ensure that the difference between the two dates exact
     * matches.
     */
    strictIntervalRange?: number;
}

export type RelationshipItemFocusMode = Pick<FocusMode, "id" | "name"> &
{
    __typename: "FocusMode";
    reminderList?: Partial<ReminderList>;
};
export type RelationshipItemMeeting = Pick<Meeting, "id"> &
{
    translations?: Pick<Meeting["translations"][0], "name" | "id" | "language">[];
    __typename: "Meeting";
}
export type RelationshipItemTeam = Pick<Team, "handle" | "id"> &
{
    translations?: Pick<Team["translations"][0], "name" | "id" | "language">[];
    __typename: "Team";
};
export type RelationshipItemQuestionForObject = { __typename: QuestionForType | `${QuestionForType}`, id: string }
export type RelationshipItemUser = Pick<User, "handle" | "id" | "name"> & {
    __typename: "User";
}
export type RelationshipItemProjectVersion = Pick<ProjectVersion, "id"> &
{
    root: Pick<Project, "__typename" | "id" | "handle" | "owner">;
    translations?: Pick<ProjectVersion["translations"][0], "name" | "id" | "language">[];
    __typename: "ProjectVersion";
};
export type RelationshipItemRoutineVersion = Pick<RoutineVersion, "id"> &
{
    root: Pick<Routine, "__typename" | "id" | "owner">;
    translations?: Pick<RoutineVersion["translations"][0], "name" | "id" | "language">[];
    __typename: "RoutineVersion";
};
export type RelationshipItemRunProject = Pick<RunProject, "id" | "name"> &
{
    __typename: "RunProject";
    projectVersion: RelationshipItemProjectVersion;
}
export type RelationshipItemRunRoutine = Pick<RunRoutine, "id" | "name"> &
{
    __typename: "RunRoutine";
    routineVersion: RelationshipItemRoutineVersion;
}

export interface RelationshipListProps {
    limitTo?: RelationshipButtonType[];
    isEditing: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
}

export interface RoleListProps {
    maxCharacters?: number;
    roles: Role[];
    sx?: SxType;
}

/**
 * Return type for a SearchList generator function
 */
export interface SearchListGenerator {
    searchType: SearchType | `${SearchType}`;
    placeholder: TranslationKeyCommon;
    where: any;
}

export type SearchListProps<T extends OrArray<ListObject>> =
    Pick<ObjectListProps<T>, "handleToggleSelect" | "isSelecting" | "selectedItems"> &
    Omit<UseFindManyResult<T>, "setAllData"> &
    {
        /**
         * The border radius of the search list container
         */
        borderRadius?: number;
        /**
         * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
         * If the callback returns false, the list item will not be selected.
         */
        canNavigate?: (item: any) => boolean,
        display: ViewDisplayType | "partial";
        /**
         * How many dummy lists to display while loading. Smaller is better for lists displayed 
         * in dialogs, since a large dummy list with a small number of results will give 
         * an annoying grow/shrink effect.
         */
        dummyLength?: number;
        /** If update button on list items should be hidden */
        hideUpdateButton?: boolean;
        scrollContainerId: string;
        searchBarVariant?: "basic" | "paper";
        searchPlaceholder?: string;
        searchType: SearchType | `${SearchType}`;
        sxs?: {
            searchBarAndButtonsBox?: SxType;
            buttons?: SxType;
            listContainer?: SxType;
        }
        onItemClick?: (item: any) => unknown;
        /**
         * Callback when search input is focused
         */
        onSearchFocus?: () => void;
        /**
         * Callback when search input loses focus
         */
        onSearchBlur?: () => void;
        /**
         * Changes the display of the search list.
         * - "normal" displays the list with a search bar and buttons
         * - "minimal" displays the list without a search bar or buttons
         */
        variant?: "normal" | "minimal";
    }

export interface SearchQueryVariablesInput<SortBy> {
    ids?: string[] | null;
    sortBy?: SortBy | null;
    searchString?: string | null;
    after?: string | null;
    take?: number | null;
    createdTimeFrame?: TimeFrame | null;
}

export interface SettingsToggleListItemProps {
    description?: string;
    disabled?: boolean;
    name: string;
    title: string;
}

export interface TagListProps {
    /**
     * Maximum characters to display before tags are truncated
     */
    maxCharacters?: number;
    tags: Partial<Tag>[];
}

export interface CardGridProps {
    children: ReactNode;
    disableMargin?: boolean;
    minWidth: number;
    sx?: SxType;
}

export interface LineGraphCardProps extends Omit<LineGraphProps, "dims"> {
    title?: string;
    index: number;
}

export type TIDCardSize = "default" | "small";
export interface TIDCardProps {
    below?: ReactNode;
    buttonText?: string;
    description: string;
    iconInfo: IconInfo | null | undefined;
    key: string | number;
    id?: string;
    onClick?: (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => unknown;
    size?: TIDCardSize;
    title: string;
    warning?: string;
}

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
