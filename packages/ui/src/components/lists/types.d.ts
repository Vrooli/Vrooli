import { StarFor, VoteFor } from '@local/shared';
import { Organization, Project, Resource, ResourceList, Routine, Session, Standard, Tag, User } from 'types';
import { LabelledSortOption } from 'utils';

export interface ObjectListItemProps {
    session: Session;
    index: number; // Index in list
    onClick?: (e: any, data: any) => void; // Full data passed back, to display while more details load
}

export interface ActorListItemProps extends ObjectListItemProps {
    data: Partial<User>;
}

export interface OrganizationListItemProps extends ObjectListItemProps {
    data: Partial<Organization>;
}

export interface ProjectListItemProps extends ObjectListItemProps {
    data: Partial<Project>;
}

export interface RoutineListItemProps extends ObjectListItemProps {
    data: Partial<Routine>;
}

export interface StandardListItemProps extends ObjectListItemProps {
    data: Partial<Standard>;
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

export interface StatsListProps {
    data: Array<any>;
}

export interface UpvoteDownvoteProps {
    session: Session;
    score?: number; // Net score - can be negative
    isUpvoted?: boolean | null; // If not passed, then there is neither an upvote nor a downvote
    objectId: string;
    voteFor: VoteFor;
    onChange: (isUpvote: boolean | null) => void;
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

export interface SearchQueryVariablesInput<SortBy> {
    ids?: string[] | null;
    sortBy?: SortBy | null;
    searchString?: string | null;
    after?: string | null;
    take?: number | null;
}

/**
 * Return type for a SearchList generator function
 */
export interface SearchListGenerator {
    placeholder: string;
    noResultsText: string;
    sortOptions: LabelledSortOption<any>[];
    defaultSortOption: LabelledSortOption<any>;
    sortOptionLabel: (sortOption: any, languages: readonly string[]) => string;
    searchQuery: any;
    where: any;
    onSearchSelect: (objectData: any) => void;
    searchItemFactory: (node: any, index: number) => JSX.Element | null;
}

export interface SearchListProps<DataType, SortBy> {
    handleAdd?: () => void; // Not shown if not passed
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
    listItemFactory: (node: DataType, index: number) => JSX.Element | null;
    getOptionLabel: (option: any, languages: readonly string[]) => string;
    onObjectSelect: (objectData: any) => void; // Passes all object data to the parent, so the known information can be displayed while more details are queried
    onScrolledFar?: () => void; // Called when scrolled far enough to prompt the user to create a new object
    where?: any; // Additional where clause to pass to the query
    noResultsText?: string; // Text to display when no results are found
    session: Session;
}

export interface TagListProps {
    session: Seession;
    parentId: string;
    tags: Partial<Tag>[];
}