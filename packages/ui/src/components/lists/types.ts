import { ApiVersion, GqlModelType, NoteVersion, Organization, ProjectVersion, Role, RoutineVersion, Session, SmartContractVersion, StandardVersion, Tag, User } from '@shared/consts';
import { CommonKey } from '@shared/translations';
import { AwardDisplay, NavigableObject } from 'types';
import { ObjectAction } from 'utils/actions/objectActions';
import { ListObjectType } from 'utils/display/listTools';
import { UseObjectActionsReturn } from 'utils/hooks/useObjectActions';
import { SearchType } from 'utils/search/objectToSearch';

export type ObjectActionsRowObject = ApiVersion | NoteVersion | Organization | ProjectVersion | RoutineVersion | SmartContractVersion | StandardVersion | User;
export interface ObjectActionsRowProps<T extends ObjectActionsRowObject> {
    actionData: UseObjectActionsReturn;
    exclude?: ObjectAction[];
    object: T | null | undefined;
    session: Session | undefined;
    zIndex: number;
}

export interface ObjectListItemProps<T extends ListObjectType> {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    beforeNavigation?: (item: NavigableObject) => boolean | void,
    /**
     * True if update button should be hidden
     */
    hideUpdateButton?: boolean;
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
    session: Session | undefined;
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
    minDate?: Date;
    maxDate?: Date;
    range?: { after: Date | undefined, before: Date | undefined };
    /**
     * If set, the date range will ensure that the difference between the two dates exact
     * matches.
     */
    strictIntervalRange?: number;
}

export interface RoleListProps {
    maxCharacters?: number;
    roles: Role[];
    sx?: { [x: string]: any };
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
     * True if update button should be hidden
     */
    hideUpdateButton?: boolean;
    id: string;
    searchPlaceholder?: CommonKey;
    take?: number; // Number of items to fetch per page
    searchType: SearchType | `${SearchType}`;
    onScrolledFar?: () => void; // Called when scrolled far enough to prompt the user to create a new object
    where?: any; // Additional where clause to pass to the query
    session: Session | undefined;
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
    checked: boolean;
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
    title: string;
}

export interface TagListProps {
    /**
     * Maximum characters to display before tags are truncated
     */
    maxCharacters?: number;
    session: Session | undefined;
    parentId: string;
    sx?: { [x: string]: any };
    tags: Partial<Tag>[];
}

export interface AwardListProps {
    awards: AwardDisplay[];
}