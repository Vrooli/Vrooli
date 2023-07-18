import { ApiVersion, Chat, CommonKey, FocusMode, GqlModelType, Meeting, Member, NoteVersion, Notification, Organization, Project, ProjectVersion, Question, QuestionForType, Reminder, Role, Routine, RoutineVersion, RunProject, RunRoutine, SmartContractVersion, StandardVersion, Tag, User } from "@local/shared";
import { LineGraphProps } from "components/graphs/types";
import { ReactNode } from "react";
import { AwardDisplay, NavigableObject, SvgComponent, SxType } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { ListObjectType } from "utils/display/listTools";
import { UseObjectActionsReturn } from "utils/hooks/useObjectActions";
import { ObjectType } from "utils/navigation/openObject";
import { SearchType } from "utils/search/objectToSearch";

export type ObjectActionsRowObject = ApiVersion | NoteVersion | Organization | ProjectVersion | Question | RoutineVersion | SmartContractVersion | StandardVersion | User;
export interface ObjectActionsRowProps<T extends ObjectActionsRowObject> {
    actionData: UseObjectActionsReturn;
    exclude?: ObjectAction[];
    object: T | null | undefined;
    zIndex: number;
}


type ActionFunctions<T> = {
    [K in keyof T]: T[K] extends (...args: infer U) => any ? U : never;
};

export type ActionsType<A = undefined> = A extends undefined
    ? { onAction?: undefined }
    : { onAction: <K extends keyof A>(action: K, ...args: ActionFunctions<A>[K]) => void }

type ObjectListItemBaseProps<T extends ListObjectType> = {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    canNavigate?: (item: NavigableObject) => boolean | void;
    belowSubtitle?: React.ReactNode;
    belowTags?: React.ReactNode;
    /**
     * True if update button should be hidden
     */
    hideUpdateButton?: boolean;
    /**
     * True if data is still being fetched
     */
    loading: boolean;
    data: T | null;
    objectType: GqlModelType | `${GqlModelType}`;
    onClick?: (data: T) => void;
    subtitleOverride?: string;
    titleOverride?: string;
    toTheRight?: React.ReactNode;
    zIndex: number;
}
export type ObjectListItemProps<T extends ListObjectType, A = undefined> = ObjectListItemBaseProps<T> & ActionsType<A>

export type ChatListItemActions = {
    MarkAsRead: (id: string) => void;
    Delete: (id: string) => void;
};
export type NotificationListItemActions = {
    MarkAsRead: (id: string) => void;
    Delete: (id: string) => void;
};
export type ReminderListItemActions = {
    Delete: (id: string) => void;
    Update: (data: Reminder) => void;
};

/**
 * Maps object types to their list item's custom actions.
 * Not all object types have custom actions.
 */
export interface ListActions {
    Chat: ChatListItemActions;
    Notification: NotificationListItemActions;
    Reminder: ReminderListItemActions;
}


export type ChatListItemProps = ObjectListItemProps<Chat, ChatListItemActions>

export type MemberListItemProps = ObjectListItemProps<Member>

export type NotificationListItemProps = ObjectListItemProps<Notification, NotificationListItemActions>

export type ReminderListItemProps = ObjectListItemProps<Reminder, ReminderListItemActions>

export type RunProjectListItemProps = ObjectListItemProps<RunProject>

export type RunRoutineListItemProps = ObjectListItemProps<RunRoutine>


export interface SortMenuProps {
    sortOptions: any[];
    anchorEl: HTMLElement | null;
    onClose: (label?: string, value?: string) => void;
}

export interface TimeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: (labelKey?: CommonKey, timeFrame?: { after?: Date, before?: Date }) => void;
    zIndex: number;
}

export interface DateRangeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    onSubmit: (after?: Date | undefined, before?: Date | undefined) => void;
    minDate?: Date;
    maxDate?: Date;
    range?: { after: Date | undefined, before: Date | undefined };
    /**
     * If set, the date range will ensure that the difference between the two dates exact
     * matches.
     */
    strictIntervalRange?: number;
    zIndex: number;
}

export type RelationshipItemFocusMode = Pick<FocusMode, "id" | "name"> &
{
    __typename: "FocusMode";
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

export interface RelationshipListProps {
    isEditing: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
    zIndex: number;
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

export interface SearchListProps {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    canNavigate?: (item: any) => boolean,
    canSearch?: (where: any) => boolean;
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
    take?: number; // Number of items to fetch per page
    resolve?: (data: any) => unknown;
    searchType: SearchType | `${SearchType}`;
    sxs?: {
        search?: SxType;
    }
    onItemClick?: (item: any) => unknown;
    onScrolledFar?: () => unknown; // Called when scrolled far enough to prompt the user to create a new object
    /** Additional where clause to pass to the query */
    where?: { [key: string]: object };
    zIndex: number;
}

export interface SearchQueryVariablesInput<SortBy> {
    ids?: string[] | null;
    sortBy?: SortBy | null;
    searchString?: string | null;
    after?: string | null;
    take?: number | null;
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

export interface AwardCardProps {
    award: AwardDisplay;
    isEarned: boolean;
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
    onClick: (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => void;
    title: string;
}
