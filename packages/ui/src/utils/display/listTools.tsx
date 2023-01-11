import { AutocompleteOption, NavigableObject } from "types";
import { getTranslation, getUserLanguages } from "./translationTools";
import { ObjectListItemType } from "components/lists/types";
import { displayDate, firstString } from "./stringTools";
import { getCurrentUser } from "utils/authentication";
import { ObjectListItem } from "components";
import { RunProject, RunRoutine, Session, Star, StarFor, User, View, Vote } from "@shared/consts";
import { valueFromDot } from "utils/shape";

export type ListObjectType = ObjectListItemType | Star | Vote | View;

/**
 * All possible permissions/user-statuses any object can have
 */
export type YouInflated = {
    canComment: boolean;
    canDelete: boolean;
    canEdit: boolean;
    canFork: boolean;
    canReport: boolean;
    canShare: boolean;
    canStar: boolean;
    canView: boolean;
    canVote: boolean;
    isStarred: boolean;
    isUpvoted: boolean | null;
    isViewed: boolean;
}

/**
 * Gets the name (title) of a list object
 * @param object A list object
 * @param languages User languages
 * @returns The name of the object
 */
export const getListItemTitle = (
    object: ListObjectType | null | undefined,
    languages?: readonly string[]
): string => {
    if (!object) return "";
    const langs: readonly string[] = languages ?? getUserLanguages(undefined);
    switch (object.type) { //TODO add new types
        case 'Organization':
            return firstString(getTranslation(object, langs, true).name, object.handle);
        case 'Project':
            return firstString(getTranslation(object, langs, true).name, object.handle);
        case 'Routine':
            return firstString(getTranslation(object, langs, true).name);
        case 'RunRoutine':
            const name = firstString(object.name, getTranslation(object.routine, langs, true).name);
            const date = object.startedAt ? (new Date(object.startedAt)) : null;
            if (date) return `${name} (${date.toLocaleDateString()} ${date.toLocaleTimeString()})`;
            return name;
        case 'Standard':
            return firstString(getTranslation(object, langs, true).name, object.name);
        case 'Star':
            return getListItemTitle(object.to as any, langs);
        case 'User':
            return firstString(object.name, object.handle);
        case 'View':
            return firstString(object.name).length > 0 ? object.name : getListItemTitle(object.to as any, langs);
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
    switch (object.type) {
        case 'Organization':
            return firstString(getTranslation(object, langs, true).bio);
        case 'Project':
            return firstString(getTranslation(object, langs, true).description);
        case 'Routine':
            return firstString(getTranslation(object, langs, true).description);
        case 'RunRoutine':
            // Subtitle for a run is the time started/completed, or nothing (depending on status)
            const startedAt: string | null = object?.startedAt ? displayDate(object.startedAt) : null;
            const completedAt: string | null = object?.completedAt ? displayDate(object.completedAt) : null;
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
 * Gets user permissions and statuses for an object. These are inflated to match YouInflated, so any fields not present are false
 * @param object An object
 * @param session The current session
 */
export const getYou = (
    object: ListObjectType | null | undefined
): YouInflated => {
    // Initialize fields to false (except isUpvoted, where false means downvoted)
    const defaultPermissions = {
        canComment: false,
        canDelete: false,
        canEdit: false,
        canFork: false,
        canReport: false,
        canShare: false,
        canStar: false,
        canView: false,
        canVote: false,
        isStarred: false,
        isUpvoted: null,
        isViewed: false,
    };
    if (!object) return defaultPermissions;
    // If the object is a star, view, or vote, get the permissions for the object it is a star of
    if (['Star', 'View', 'Vote'].includes(object.type)) return getYou((object as Star | View | Vote).to as any);
    // If the object is a run routine, get the permissions for the routine
    if (object.type === 'RunRoutine') return getYou((object as RunRoutine).routineVersion as any);
    // If the object is a run project, get the permissions for the project
    if (object.type === 'RunProject') return getYou((object as RunProject).projectVersion as any);
    // Otherwise, get the permissions from the object
    // Loop through all permission fields
    for (const key in defaultPermissions) {
        // Check if the field is in the object
        const field = valueFromDot(object, `you.${key}`);
        if (field === true || field === false) defaultPermissions[key] = field;
        // If not, check if the field is in the root.you object
        else {
            const field = valueFromDot(object, `root.you.${key}`);
            if (field === true || field === false) defaultPermissions[key] = field;
        }
    }
    return defaultPermissions;
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
    switch (object.type) {
        case 'Organization':
        case 'Project':
        case 'Routine':
        case 'Standard':
        case 'User':
            return object.stars;
        case 'RunRoutine':
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
    switch (object.type) {
        case 'Organization':
        case 'Project':
        case 'Routine':
        case 'Standard':
        case 'User':
            return object.type as StarFor;
        case 'RunRoutine':
            return StarFor.Routine;
        case 'Star':
        case 'View':
            return getListItemStarFor(object.to as any);
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
    switch (object.type) {
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
        type: o.type,
        id: o.id,
        isStarred: getYou(o).isStarred,
        label: getListItemSubtitle(o, languages),
        routine: o.type === 'RunRoutine' ? o.routineVersion : undefined,
        stars: getListItemStars(o),
        to: o.type === 'View' || o.type === 'Star' ? o.to : undefined,
        versionGroupId: undefined// TODO o.type === 'Routine' || o.type === 'Standard' ? o.versionGroupId : undefined,
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
        if (curr.type === 'View' || curr.type === 'Star') {
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