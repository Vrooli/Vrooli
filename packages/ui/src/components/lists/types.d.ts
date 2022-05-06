import { StarFor, VoteFor } from '@local/shared';
import { ListOrganization, ListProject, ListRoutine, ListRun, ListStandard, ListUser, Resource, ResourceList, Session, Tag } from 'types';
import { LabelledSortOption } from 'utils';

export interface ObjectListItemProps {
    session: Session;
    index: number; // Index in list
    onClick?: (e: any, data: any) => void; // Full data passed back, to display while more details load
}

export interface OrganizationListItemProps extends ObjectListItemProps {
    data: ListOrganization;
}

export interface ProjectListItemProps extends ObjectListItemProps {
    data: ListProject;
}

export interface RoutineListItemProps extends ObjectListItemProps {
    data: ListRoutine;
}

export interface RunListItemProps extends ObjectListItemProps {
    data: ListRun;
}

export interface StandardListItemProps extends ObjectListItemProps {
    data: ListStandard;
}

export interface SortMenuProps {
    sortOptions: any[];
    anchorEl: HTMLElement | null;
    onClose: (label?: string, value?: string) => void;
}

export interface TimeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: (label?: string, after?: Date | null, before?: Date | null) => void;
}

export interface DateRangeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    onSubmit: (after?: Date | null, before?: Date | null) => void;
}

/**
 * Return type for a SearchList generator function
 */
export interface SearchListGenerator {
    itemKeyPrefix: string;
    placeholder: string;
    noResultsText: string;
    sortOptions: LabelledSortOption<any>[];
    defaultSortOption: LabelledSortOption<any>;
    searchQuery: any;
    where: any;
    onSearchSelect: (objectData: any) => void;
}

export interface SearchListProps<SortBy> {
    handleAdd?: () => void; // Not shown if not passed
    itemKeyPrefix: string;
    searchPlaceholder?: string;
    sortOptions: SearchSortBy<SortBy>[];
    defaultSortOption: SearchSortBy<SortBy>;
    query: DocumentNode;
    take?: number; // Number of items to fetch per page
    searchString: string;
    sortBy: string | undefined;
    timeFrame: string | undefined;
    setSearchString: (searchString: string) => void;
    setSortBy: (sortBy: string | undefined) => void;
    setTimeFrame: (timeFrame: string | undefined) => void;
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

export interface UserListItemProps extends ObjectListItemProps {
    data: ListUser;
}