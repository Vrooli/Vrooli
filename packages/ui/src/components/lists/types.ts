import { BookmarkList, Chat, ChatInvite, ChatParticipant, CommonKey, FocusMode, Meeting, MeetingInvite, Member, MemberInvite, Notification, OrArray, Organization, Project, ProjectVersion, QuestionForType, Reminder, ReminderList, Role, Routine, RoutineVersion, RunProject, RunRoutine, Tag, TimeFrame, User } from "@local/shared";
import { LineGraphProps } from "components/graphs/types";
import { UseFindManyResult } from "hooks/useFindMany";
import { UseObjectActionsReturn } from "hooks/useObjectActions";
import { ReactNode } from "react";
import { NavigableObject, SvgComponent, SxType } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { ListObject } from "utils/display/listTools";
import { ObjectType } from "utils/navigation/openObject";
import { SearchType } from "utils/search/objectToSearch";
import { ViewDisplayType } from "views/types";
import { ObjectListProps } from "./ObjectList/ObjectList";

export interface ObjectActionsRowProps<T extends ListObject> {
    actionData: UseObjectActionsReturn;
    exclude?: ObjectAction[];
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
    handleToggleSelect: (object: ListObject) => unknown;
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
export type ObjectListItemProps<T extends ListObject> = ObjectListItemBaseProps<T> & ActionsType<T>;

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
export type RunProjectListItemProps = ObjectListItemProps<RunProject>
export type RunRoutineListItemProps = ObjectListItemProps<RunRoutine>

export interface DateRangeMenuProps {
    anchorEl: HTMLElement | null;
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
export type RelationshipItemOrganization = Pick<Organization, "handle" | "id"> &
{
    translations?: Pick<Organization["translations"][0], "name" | "id" | "language">[];
    __typename: "Organization";
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

export type RelationshipButtonType =
    "Owner" |
    "Project" |
    "Parent" |
    "IsPrivate" |
    "IsComplete" |
    "FocusMode" |
    "Meeting" |
    "RunProject" |
    "RunRoutine" |
    "QuestionFor" |
    "Members" |
    "Participants" |
    "Organization" |
    "User";

export interface RelationshipListProps {
    limitTo?: RelationshipButtonType[];
    isEditing: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
    sx?: SxType;
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
    placeholder: CommonKey;
    where: any;
}

export type SearchListProps<T extends OrArray<ListObject>> =
    Pick<ObjectListProps<T>, "handleToggleSelect" | "isSelecting" | "selectedItems"> &
    Omit<UseFindManyResult<T>, "setAllData"> &
    {
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
        handleAdd?: (event?: any) => unknown; // Not shown if not passed
        /** If update button on list items should be hidden */
        hideUpdateButton?: boolean;
        id: string;
        searchPlaceholder?: CommonKey;
        searchType: SearchType | `${SearchType}`;
        sxs?: {
            search?: SxType;
            buttons?: SxType;
            listContainer?: SxType;
        }
        onItemClick?: (item: any) => unknown;
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
    parentId: string;
    sx?: SxType;
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

export interface TIDCardProps {
    buttonText: string;
    description: string;
    key: string | number;
    Icon: SvgComponent | null | undefined;
    id?: string;
    onClick: (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => unknown;
    title: string;
}
