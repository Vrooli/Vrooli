import { VoteFor } from '@shared/consts';
import { SearchType } from 'components/dialogs';
import { ListOrganization, ListProject, ListRoutine, ListRun, ListStandard, ListUser, Project, Routine, Session, Standard, Tag } from 'types';
import { ObjectAction } from 'utils';

export type ObjectActionsRowPropsObject = Project | Routine | Standard;
export interface ObjectActionsRowProps {
    exclude?: ObjectAction[];
    /**
     * Completed actions, which may require updating state or navigating to a new page
     */
    onActionComplete: (action: ObjectActionComplete, data: any) => any;
    /**
     * Actions which cannot be performed by the menu
     */
    onActionStart: (action: ObjectAction.Edit | ObjectAction.Stats) => any;
    object: T | null;
    session: Session;
    zIndex: number;
}

export type ObjectListItemType = ListOrganization | ListProject | ListRoutine | ListRun | ListStandard | ListUser;

export interface ObjectListItemProps<T extends ObjectListItemType> {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     * 
     * NOTE: Passes back full data so we can display 
     * known properties of the object, while waiting for the full data to be fetched.
     */
    beforeNavigation?: (item: NavigableObject) => boolean | void,
    data: T | null;
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
    session: Session;
    zIndex: number;
}

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
    searchType: SearchType;
    placeholder: string;
    noResultsText: string;
    where: any;
}

export interface SearchListProps {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    beforeNavigation?: (item: any) => boolean | void,
    canSearch?: boolean;
    handleAdd?: (event?: any) => void; // Not shown if not passed
    /**
     * True if roles (admin, owner, etc.) should be hidden in list items
     */
    hideRoles?: boolean;
    id: string;
    itemKeyPrefix: string;
    searchPlaceholder?: string;
    take?: number; // Number of items to fetch per page
    searchType: SearchType;
    onScrolledFar?: () => void; // Called when scrolled far enough to prompt the user to create a new object
    where?: any; // Additional where clause to pass to the query
    noResultsText?: string; // Text to display when no results are found
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