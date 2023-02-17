import { ApiVersion, GqlModelType, NoteVersion, Organization, ProjectVersion, RoutineVersion, Session, SmartContractVersion, StandardVersion, Tag, User, VoteFor } from '@shared/consts';
import { CommonKey, NavigableObject } from 'types';
import { ListObjectType, ObjectAction, SearchType, UseObjectActionsReturn } from 'utils';

export type ObjectActionsRowObject = ApiVersion | NoteVersion | Organization | ProjectVersion | RoutineVersion | SmartContractVersion | StandardVersion | User;
export interface ObjectActionsRowProps<T extends ObjectActionsRowObject> {
    actionData: UseObjectActionsReturn;
    exclude?: ObjectAction[];
    object: T | null | undefined;
    session: Session;
    zIndex: number;
}

export interface ObjectListItemProps<T extends ListObjectType> {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    beforeNavigation?: (item: NavigableObject) => boolean | void,
    /**
     * True if role (admin, owner, etc.) should be hidden
     */
    hideRole?: boolean;
    /**
     * Index in list
     */
    index: number;
    /**
     * True if data is still being fetched
     */
    loading: boolean;
    data: T | null;
    objectType: GqlModelType | `${GqlModelType}`;
    session: Session;
    zIndex: number;
}

export interface SortMenuProps {
    sortOptions: any[];
    anchorEl: HTMLElement | null;
    lng: string;
    onClose: (label?: string, value?: string) => void;
}

export interface TimeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: (label?: string, timeFrame?: { after?: Date, before?: Date }) => void;
    session: Session;
}

export interface DateRangeMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    onSubmit: (after?: Date | undefined, before?: Date | undefined) => void;
    minDate?: Date;
    maxDate?: Date;
    resetDateRange?: boolean;
    session: Session;
    /**
     * If set, the date range will ensure that the difference between the two dates exact
     * matches.
     */
    strictIntervalRange?: number;
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
    beforeNavigation?: (item: any) => boolean,
    canSearch?: boolean;
    handleAdd?: (event?: any) => void; // Not shown if not passed
    /**
     * True if roles (admin, owner, etc.) should be hidden in list items
     */
    hideRoles?: boolean;
    id: string;
    searchPlaceholder?: CommonKey;
    take?: number; // Number of items to fetch per page
    searchType: SearchType | `${SearchType}`;
    onScrolledFar?: () => void; // Called when scrolled far enough to prompt the user to create a new object
    where?: any; // Additional where clause to pass to the query
    session: Session;
    zIndex: number;
}

export interface SearchQueryVariablesInput<SortBy> {
    ids?: string[] | null;
    sortBy?: SortBy | null;
    searchString?: string | null;
    after?: string | null;
    take?: number | null;
}

export interface StatsListProps {
    data: Array<any>;
}

export interface TagListProps {
    /**
     * Maximum characters to display before tags are truncated
     */
    maxCharacters?: number;
    session: Session;
    parentId: string;
    sx?: { [x: string]: any };
    tags: Partial<Tag>[];
}

export interface UpvoteDownvoteProps {
    direction?: 'row' | 'column';
    disabled?: boolean;
    session: Session;
    score?: number; // Net score - can be negative
    isUpvoted?: boolean | null; // If not passed, then there is neither an upvote nor a downvote
    objectId: string;
    voteFor: VoteFor;
    onChange: (isUpvote: boolean | null, newScore: number) => void;
}