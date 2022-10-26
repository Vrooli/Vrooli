import { AutocompleteOption, ListStar, ListView, NavigableObject, Session } from "types";
import { getTranslation, getUserLanguages } from "./translationTools";
import { ObjectListItemType } from "components/lists/types";
import { displayDate, firstString } from "./stringTools";
import { getCurrentUser } from "utils/authentication";
import { ObjectListItem } from "components";
import { StarFor } from "@shared/consts";

export type ListObjectType = ObjectListItemType | ListStar | ListView;

/**
 * Gets the title of a list object
 * @param object A list object
 * @param languages User languages
 * @returns The title of the object
 */
export const getListItemTitle = (
    object: ListObjectType | null | undefined,
    languages?: readonly string[]
): string => {
    if (!object) return "";
    const langs: readonly string[] = languages ?? getUserLanguages(undefined);
    switch (object.__typename) {
        case 'Organization':
            return firstString(getTranslation(object, langs, true).name, object.handle);
        case 'Project':
            return firstString(getTranslation(object, langs, true).name, object.handle);
        case 'Routine':
            return firstString(getTranslation(object, langs, true).title);
        case 'Run':
            const title = firstString(object.title, getTranslation(object.routine, langs, true).title);
            const date = object.timeStarted ? (new Date(object.timeStarted)) : null;
            if (date) return `${title} (${date.toLocaleDateString()} ${date.toLocaleTimeString()})`;
            return title;
        case 'Standard':
            return firstString(object.name);
        case 'Star':
            return getListItemTitle(object.to as any, langs);
        case 'User':
            return firstString(object.name, object.handle);
        case 'View':
            return firstString(object.title).length > 0 ? object.title : getListItemTitle(object.to as any, langs);
        default:
            return '';
    }
};

/**
 * Gets the subtitle of a list object
 * @param object A list object
 * @param languages User languages
 */
export const getListItemSubtitle = (
    object: ListObjectType | null | undefined,
    languages?: readonly string[]
): string => {
    if (!object) return "";
    const langs: readonly string[] = languages ?? getUserLanguages(undefined);
    switch (object.__typename) {
        case 'Organization':
            return firstString(getTranslation(object, langs, true).bio);
        case 'Project':
            return firstString(getTranslation(object, langs, true).description);
        case 'Routine':
            return firstString(getTranslation(object, langs, true).description);
        case 'Run':
            // Subtitle for a run is the time started/completed, or nothing (depending on status)
            const startedAt: string | null = object?.timeStarted ? displayDate(object.timeStarted) : null;
            const completedAt: string | null = object?.timeCompleted ? displayDate(object.timeCompleted) : null;
            if (completedAt) return `Completed: ${completedAt}`;
            if (startedAt) return `Started: ${startedAt}`;
            return '';
        case 'Standard':
            return firstString(getTranslation(object, langs, true).description);
        case 'Star':
            return getListItemSubtitle(object.to as any, langs);
        case 'User':
            return firstString(getTranslation(object, langs, true).bio);
        case 'View':
            return getListItemSubtitle(object.to as any, langs);
        default:
            return '';
    }
};

/**
 * All possible permissions any object can have
 */
export type PermissionsInflated = {
    canComment: boolean;
    canDelete: boolean;
    canEdit: boolean;
    canFork: boolean;
    canReport: boolean;
    canShare: boolean;
    canStar: boolean;
    canView: boolean;
    canVote: boolean;
}

/**
 * Gets the permissions of a list object
 * @param object A list object
 * @param session The current session
 */
export const getListItemPermissions = (
    object: ListObjectType | null | undefined,
    session: Session,
): PermissionsInflated => {
    const defaultPermissions = { canComment: false, canDelete: false, canEdit: false, canFork: false, canReport: false, canShare: false, canStar: false, canView: false, canVote: false };
    if (!object) return defaultPermissions;
    // Helper function to convert every field in an object to boolean
    const toBoolean = <T extends { [key: string]: any }>(obj: T | null | undefined): { [K in keyof T]?: boolean } => {
        if (!obj) return {};
        const newObj: { [K in keyof T]?: boolean } = {};
        for (const key in obj) {
            newObj[key] = obj[key] === true;
        }
        return newObj as { [K in keyof T]: boolean };
    };
    switch (object.__typename) {
        case 'Organization':
            return { ...defaultPermissions, ...toBoolean(object.permissionsOrganization), canShare: object.isPrivate !== true };
        case 'Project':
            return { ...defaultPermissions, ...toBoolean(object.permissionsProject), canShare: object.isPrivate !== true };
        case 'Routine':
            return { ...defaultPermissions, ...toBoolean(object.permissionsRoutine), canShare: object.isPrivate !== true };
        case 'Run':
            return { ...defaultPermissions, ...toBoolean(object.routine?.permissionsRoutine), canShare: object.routine?.isPrivate !== true };
        case 'Standard':
            return { ...defaultPermissions, ...toBoolean(object.permissionsStandard), canShare: object.isPrivate !== true };
        case 'Star':
            return getListItemPermissions(object.to as any, session);
        case 'User':
            const isOwn = object.id === getCurrentUser(session).id;
            return { canComment: false, canDelete: isOwn, canEdit: isOwn, canFork: false, canReport: !isOwn, canShare: true, canStar: !isOwn, canView: true, canVote: false };
        case 'View':
            return getListItemPermissions(object.to as any, session);
        default:
            return defaultPermissions;
    }
}

/**
 * Gets stars for a single object
 * @param object A searchable object
 * @returns stars
 */
export const getListItemStars = (
    object: ListObjectType | null | undefined,
): number => {
    if (!object) return 0;
    switch (object.__typename) {
        case 'Organization':
        case 'Project':
        case 'Routine':
        case 'Standard':
        case 'User':
            return object.stars;
        case 'Run':
            return object.routine?.stars ?? 0;
        case 'Star':
        case 'View':
            return getListItemStars(object.to as any);
        default:
            return 0;
    }
}

export const getListItemStarFor = (
    object: ListObjectType | null | undefined,
): StarFor | null => {
    if (!object) return null;
    switch (object.__typename) {
        case 'Organization':
        case 'Project':
        case 'Routine':
        case 'Standard':
        case 'User':
            return object.__typename as StarFor;
        case 'Run':
            return StarFor.Routine;
        case 'Star':
        case 'View':
            return getListItemStarFor(object.to as any);
        default:
            return null;
    }
}

/**
 * Gets isStarred for a single object
 * @param object A searchable object
 * @returns isStarred
 */
export const getListItemIsStarred = (
    object: ListObjectType | null | undefined,
): boolean => {
    if (!object) return false;
    switch (object.__typename) {
        case 'Organization':
        case 'Project':
        case 'Routine':
        case 'Standard':
        case 'User':
            return object.isStarred;
        case 'Run':
            return object.routine?.isStarred ?? false;
        case 'Star':
        case 'View':
            return getListItemIsStarred(object.to as any);
        default:
            return false;
    }
}

/**
 * Gets isUpvoted for a single object
 * @param object A searchable object
 * @returns isUpvoted
 */
export const getListItemIsUpvoted = (
    object: ListObjectType | null | undefined,
): boolean | null => {
    if (!object) return false;
    switch (object.__typename) {
        case 'Project':
        case 'Routine':
        case 'Standard':
            return object.isUpvoted;
        case 'Run':
            return object.routine?.isUpvoted ?? null;
        case 'Star':
        case 'View':    
            return getListItemIsUpvoted(object.to as any);
        default:
            return null;
    }
}

/**
 * Gets reportsCount for a single object
 * @param object A searchable object
 * @returns number of reports
 */
export const getListItemReportsCount = (
    object: ListObjectType | null | undefined,
): number => {
    if (!object) return 0;
    switch (object.__typename) {
        case 'Organization':
        case 'Project':
        case 'Routine':
        case 'Standard':
        case 'User':
            return object.reportsCount;
        case 'Star':
        case 'View':
            return getListItemStars(object.to as any);
        default:
            return 0;
    }
}

/**
 * Converts a list of GraphQL objects to a list of autocomplete information.
 * @param objects The list of search results
 * @param languages User languages
 * @returns The list of autocomplete information. Each object has the following shape: 
 * {
 *  id: The ID of the object.
 *  label: The label of the object.
 *  stars: The number of stars the object has.
 * }
 */
export function listToAutocomplete(
    objects: readonly ListObjectType[],
    languages: readonly string[]
): AutocompleteOption[] {
    return objects.map(o => ({
        __typename: o.__typename,
        id: o.id,
        isStarred: getListItemIsStarred(o),
        label: getListItemSubtitle(o, languages),
        routine: o.__typename === 'Run' ? o.routine : undefined,
        stars: getListItemStars(o),
        to: o.__typename === 'View' || o.__typename === 'Star' ? o.to : undefined,
        versionGroupId: o.__typename === 'Routine' || o.__typename === 'Standard' ? o.versionGroupId : undefined,
    }));
}

export interface ListToListItemProps {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    beforeNavigation?: (item: NavigableObject) => boolean | void,
    /**
     * List of dummy items types to display while loading
     */
    dummyItems?: string[];
    /**
     * If role (admin, owner, etc.) should be hiden in list itmes
     */
    hideRoles?: boolean,
    /**
     * The list of item data. Objects like view and star are converted to their respective objects.
     */
    items?: readonly ListObjectType[],
    /**
     * Each list item's key will be `${keyPrefix}-${id}`
     */
    keyPrefix: string,
    /**
     * Whether the list is loading
     */
    loading: boolean,
    /**
     * Current session
     */
    session: Session,
    zIndex: number,
}

/**
 * Converts a list of objects to a list of ListItems
 * @returns A list of ListItems
 */
export function listToListItems({
    beforeNavigation,
    dummyItems,
    keyPrefix,
    hideRoles,
    items,
    loading,
    session,
    zIndex,
}: ListToListItemProps): JSX.Element[] {
    let listItems: JSX.Element[] = [];
    // If loading, display dummy items
    if (loading) {
        if (!dummyItems) return listItems;
        for (let i = 0; i < dummyItems.length; i++) {
            listItems.push(<ObjectListItem
                key={`${keyPrefix}-${i}`}
                data={null}
                hideRole={hideRoles}
                index={i}
                loading={true}
                session={session}
                zIndex={zIndex}
            />);
        }
    }
    if (!items) return listItems;
    for (let i = 0; i < items.length; i++) {
        let curr = items[i];
        // If "View" or "Star" item, display the object it points to
        if (curr.__typename === 'View' || curr.__typename === 'Star') {
            curr = (curr as ListStar | ListView).to as ObjectListItemType;
        }
        listItems.push(<ObjectListItem
            key={`${keyPrefix}-${curr.id}`}
            beforeNavigation={beforeNavigation}
            data={curr}
            hideRole={hideRoles}
            index={i}
            loading={false}
            session={session}
            zIndex={zIndex}
        />);
    }
    return listItems;
}

/**
 * Color options for placeholder icon
 * [background color, silhouette color]
 */
const placeholderColors: [string, string][] = [
    ["#197e2c", "#b5ffc4"],
    ["#b578b6", "#fecfea"],
    ["#4044d6", "#e1c7f3"],
    ["#d64053", "#fbb8c5"],
    ["#d69440", "#e5d295"],
    ["#40a4d6", "#79e0ef"],
    ["#6248e4", "#aac3c9"],
    ["#8ec22c", "#cfe7b4"],
]

/**
 * Finds a random color for a placeholder icon
 * @returns [background color code, silhouette color code]
 */
export const placeholderColor = (): [string, string] => {
    return placeholderColors[Math.floor(Math.random() * placeholderColors.length)];
}