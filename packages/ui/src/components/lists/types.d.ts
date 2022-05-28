import { StarFor, VoteFor } from '@local/shared';
import { ListOrganization, ListProject, ListRoutine, ListRun, ListStandard, ListUser, Session, Tag } from 'types';
import { ObjectType } from 'utils';

export interface ObjectListItemProps<DataType> {
    data: DataType | null;
    /**
     * Index in list
     */
    index: number;
    /**
     * True if data is still being fetched
     */
    loading: boolean;
    session: Session;
    /**
     * onClick handler. Passes back full data so we can display 
     * known properties of the object, while waiting for the full data to be fetched.
     */
    onClick?: (e: React.MouseEvent<any>, data: DataType) => any;
    /**
     * Tooltip text to display when hovering over the item
     */
    tooltip?: string | null | undefined;
}

export type OrganizationListItemProps = ObjectListItemProps<ListOrganization>;
export type ProjectListItemProps = ObjectListItemProps<ListProject>;
export type RoutineListItemProps = ObjectListItemProps<ListRoutine>;
export type RunListItemProps = ObjectListItemProps<ListRun>;
export type StandardListItemProps = ObjectListItemProps<ListStandard>;
export type UserListItemProps = ObjectListItemProps<ListUser>;

export interface SortMenuProps {
    sortOptions: any[];
    anchorEl: HTMLElement | null;
    onClose: (label?: string, value?: string) => void;
}

export interface TimeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: (label?: string, timeFrame?: { after?: Date, before?: Date }) => void;
}

export interface DateRangeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    onSubmit: (after?: Date | undefined, before?: Date | undefined) => void;
}

/**
 * Return type for a SearchList generator function
 */
export interface SearchListGenerator {
    itemKeyPrefix: string;
    objectType: ObjectType;
    placeholder: string;
    noResultsText: string;
    searchQuery: any;
    where: any;
    onSearchSelect: (objectData: any) => void;
}

export interface SearchListProps {
    canSearch?: boolean;
    handleAdd?: () => void; // Not shown if not passed
    itemKeyPrefix: string;
    searchPlaceholder?: string;
    query: DocumentNode;
    take?: number; // Number of items to fetch per page
    objectType: ObjectType;
    onObjectSelect: (objectData: any) => void; // Passes all object data to the parent, so the known information can be displayed while more details are queried
    onScrolledFar?: () => void; // Called when scrolled far enough to prompt the user to create a new object
    where?: any; // Additional where clause to pass to the query
    noResultsText?: string; // Text to display when no results are found
    session: Session;
}

export interface SearchQueryVariablesInput<SortBy> {
    ids?: string[] | null;
    sortBy?: SortBy | null;
    searchString?: string | null;
    after?: string | null;
    take?: number | null;
}

export interface StarButtonProps {
    session: Session;
    isStar?: boolean | null; // Defaults to false
    showStars?: boolean; // Defaults to true. If false, the number of stars is not shown
    stars?: number | null; // Defaults to 0
    objectId: string;
    starFor: StarFor;
    onChange: (isStar: boolean) => void;
    tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
}

export interface StatsListProps {
    data: Array<any>;
}

export interface TagListProps {
    session: Session;
    parentId: string;
    tags: Partial<Tag>[];
}

export interface UpvoteDownvoteProps {
    session: Session;
    score?: number; // Net score - can be negative
    isUpvoted?: boolean | null; // If not passed, then there is neither an upvote nor a downvote
    objectId: string;
    voteFor: VoteFor;
    onChange: (isUpvote: boolean | null) => void;
}