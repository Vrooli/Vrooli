import { type BookmarkList, type Chat, type ChatInvite, type ChatParticipant, type ListObject, type Meeting, type MeetingInvite, type Member, type MemberInvite, type ModelType, type NavigableObject, type Notification, type OrArray, type Reminder, type Report, type ReportResponse, type Resource, type ResourceVersion, type SearchType, type Tag, type Team, type TimeFrame, type TranslationKeyCommon, type User } from "@vrooli/shared";
import { type ReactNode } from "react";
import { type PermissionsFilter } from "../buttons/PermissionsButton.js";
import { type UsePressEvent } from "../../hooks/gestures.js";
import { type UseObjectActionsReturn } from "../../hooks/objectActions.js";
import { type UseFindManyResult } from "../../hooks/useFindMany.js";
import { type IconInfo } from "../../icons/Icons.js";
import { type SxType, type ViewDisplayType } from "../../types.js";
import { type ObjectAction } from "../../utils/actions/objectActions.js";
import { type RelationshipButtonType } from "../../utils/consts.js";
import { type ObjectType } from "../../utils/navigation/openObject.js";
import { type LineGraphProps } from "../graphs/types.js";
import { type ObjectListProps } from "./ObjectList/ObjectList.js";

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
    handleContextMenu: (target: EventTarget, object: ListObject | null) => void;
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
export type NotificationListItemProps = ObjectListItemProps<Notification> & {
    sx?: SxType;
    style?: React.CSSProperties;
};
export type ReminderListItemProps = ObjectListItemProps<Reminder>
export type ReportListItemProps = ObjectListItemProps<Report>
export type ReportResponseListItemProps = ObjectListItemProps<ReportResponse>
export type RunListItemProps = ObjectListItemProps<RunRoutine>

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
export type RelationshipItemUser = Pick<User, "handle" | "id" | "name"> & {
    __typename: "User";
}
export type RelationshipItemProjectVersion = Pick<ResourceVersion, "id"> &
{
    root: Pick<Resource, "__typename" | "id" | "owner">;
    translations?: Pick<ResourceVersion["translations"][0], "name" | "id" | "language">[];
    __typename: "ResourceVersion";
};
export type RelationshipItemRoutineVersion = Pick<ResourceVersion, "id"> &
{
    root: Pick<Resource, "__typename" | "id" | "owner">;
    translations?: Pick<ResourceVersion["translations"][0], "name" | "id" | "language">[];
    __typename: "ResourceVersion";
};

export interface RelationshipListProps {
    limitTo?: RelationshipButtonType[];
    isEditing: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
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
        display: ViewDisplayType | `${ViewDisplayType}`;
        /**
         * How many dummy lists to display while loading. Smaller is better for lists displayed 
         * in dialogs, since a large dummy list with a small number of results will give 
         * an annoying grow/shrink effect.
         */
        dummyLength?: number;
        /** If update button on list items should be hidden */
        hideUpdateButton?: boolean;
        /**
         * Optional permissions filter value
         */
        permissionsFilter?: PermissionsFilter;
        scrollContainerId: string;
        searchBarVariant?: "basic" | "paper";
        searchPlaceholder?: string;
        searchType: SearchType | `${SearchType}`;
        /**
         * Optional setter for permissions filter
         */
        setPermissionsFilter?: (filter: PermissionsFilter) => void;
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
