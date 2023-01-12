import { AutocompleteOption, NavigableObject } from "types";
import { getTranslation, getUserLanguages } from "./translationTools";
import { ObjectListItemType } from "components/lists/types";
import { displayDate, firstString } from "./stringTools";
import { ObjectListItem } from "components";
import { GqlModelType, RunProject, RunRoutine, Session, Star, StarFor, View, Vote } from "@shared/consts";
import { valueFromDot } from "utils/shape";

export type ListObjectType = ObjectListItemType | Star | Vote | View;

/**
 * All possible permissions/user-statuses any object can have
 */
export type YouInflated = {
    canComment: boolean;
    canCopy: boolean;
    canDelete: boolean;
    canEdit: boolean;
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
 * Most possible counts (including score) any object can have
 */
export type CountsInflated = {
    comments: number;
    forks: number;
    issues: number;
    labels: number;
    pullRequests: number;
    questions: number;
    reports: number;
    score: number;
    stars: number;
    transfers: number;
    translations: number;
    versions: number;
    views: number;
}

/**
 * Gets user permissions and statuses for an object. These are inflated to match YouInflated, so any fields not present are false
 * @param object An object
 */
export const getYou = (
    object: ListObjectType | null | undefined
): YouInflated => {
    // Initialize fields to false (except isUpvoted, where false means downvoted)
    const defaultPermissions = {
        canComment: false,
        canCopy: false,
        canDelete: false,
        canEdit: false,
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
    // If the object is a star, view, or vote, use the "to" object
    if (['Star', 'View', 'Vote'].includes(object.type)) return getYou((object as Star | View | Vote).to as any);
    // If the object is a run routine, use the routine version
    if (object.type === 'RunRoutine') return getYou((object as RunRoutine).routineVersion as any);
    // If the object is a run project, use the project version
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
 * Gets counts for an object. These are inflated to match CountsInflated, so any fields not present are 0
 * @param object An object
 */
export const getCounts = (
    object: ListObjectType | null | undefined
): CountsInflated => {
    // Initialize fields to 0
    const defaultCounts = {
        comments: 0,
        forks: 0,
        issues: 0,
        labels: 0,
        pullRequests: 0,
        questions: 0,
        reports: 0,
        score: 0,
        stars: 0,
        transfers: 0,
        translations: 0,
        versions: 0,
        views: 0,
    };
    if (!object) return defaultCounts;
    // If the object is a star, view, or vote, use the "to" object
    if (['Star', 'View', 'Vote'].includes(object.type)) return getCounts((object as Star | View | Vote).to as any);
    // If the object is a run routine, use the routine version
    if (object.type === 'RunRoutine') return getCounts((object as RunRoutine).routineVersion as any);
    // If the object is a run project, use the project version
    if (object.type === 'RunProject') return getCounts((object as RunProject).projectVersion as any);
    // Otherwise, get the counts from the object
    // Loop through all count fields
    for (const key in defaultCounts) {
        // Check if the field is in the object
        const field = valueFromDot(object, `${key}Count`);
        if (field !== undefined) defaultCounts[key] = field;
        // If not, check if the field is in the root.counts object
        else {
            const field = valueFromDot(object, `root.${key}Count`);
            if (field !== undefined) defaultCounts[key] = field;
        }
    }
    return defaultCounts;
}

/**
 * Gets the name and subtitle of a list object
 * @param object A list object
 * @param languages User languages
 * @returns The name and subtitle of the object
 */
export const getDisplay = (
    object: ListObjectType | null | undefined,
    languages?: readonly string[]
): { title: string, subtitle: string } => {
    if (!object) return { title: '', subtitle: '' };
    // If the object is a star, view, or vote, use the "to" object
    if (['Star', 'View', 'Vote'].includes(object.type)) return getDisplay((object as Star | View | Vote).to as any);
    const langs: readonly string[] = languages ?? getUserLanguages(undefined);
    // If the object is a run routine, use the routine version's display and the startedAt/completedAt date
    if (object.type === 'RunRoutine') {
        const { completedAt, name, routineVersion, startedAt } = object as RunRoutine;
        const title = firstString(name, getTranslation(routineVersion!, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        return {
            title: started ? `${title} (started)` : title,
            subtitle: started ? 'Started: ' + started : completed ? 'Completed: ' + completed : ''
        }
    }
    // If the object is a run project, use the project version's display and the startedAt/completedAt date
    if (object.type === 'RunProject') {
        const { completedAt, name, projectVersion, startedAt } = object as RunProject;
        const title = firstString(name, getTranslation(projectVersion!, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        return {
            title: started ? `${title} (started)` : title,
            subtitle: started ? 'Started: ' + started : completed ? 'Completed: ' + completed : ''
        }
    }
    // For all other objects, fields may differ. 
    // Priority for title is: title, name, translations[number].title, translations[number].name, handle
    // Priority for subtitle is: bio, description, summary, details, translations[number].bio, translations[number].description, translations[number].summary, translations[number].details
    // If all else fails, attempt to find in "root" object
    const tryTitle = (obj: Record<string, any>) => {
        const translations: Record<string, any> = getTranslation(obj, langs, true);
        return firstString(
            obj.title,
            obj.name,
            translations.title,
            translations.name,
            obj.handle
        );
    }
    const trySubtitle = (obj: Record<string, any>) => {
        const translations: Record<string, any> = getTranslation(obj, langs, true);
        return firstString(
            obj.bio,
            obj.description,
            obj.summary,
            obj.details,
            translations.bio,
            translations.description,
            translations.summary,
            translations.details
        );
    }
    const title = tryTitle(object) ?? tryTitle((object as any).root) ?? '';
    const subtitle = trySubtitle(object) ?? trySubtitle((object as any).root) ?? '';
    return { title, subtitle };
};

/**
 * Finds the information required to star an object
 * @param object 
 * @returns StarFor type and ID of the object. For versions, for example, 
 * the ID is of the root object instead of the version passed in.
 */
export const getStarFor = (
    object: ListObjectType | null | undefined,
): { starFor: StarFor, starForId: string } | { starFor: null, starForId: null } => {
    if (!object) return { starFor: null, starForId: null };
    // If the object is a star, view, or vote, use the "to" object
    if (['Star', 'View', 'Vote'].includes(object.type)) return getStarFor((object as Star | View | Vote).to as any);
    // If the object is a run routine, use the routine version
    if (object.type === 'RunRoutine') return getStarFor((object as RunRoutine).routineVersion as any);
    // If the object is a run project, use the project version
    if (object.type === 'RunProject') return getStarFor((object as RunProject).projectVersion as any);
    // If the object contains a root object, use that
    if ((object as any).root) return getStarFor((object as any).root);
    // Use current object
    return { starFor: object.type as unknown as StarFor, starForId: object.id };
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
        label: getDisplay(o, languages).title,
        runnableObject: o.type === GqlModelType.RunProject ?
            (o as RunProject).projectVersion :
            o.type === GqlModelType.RunRoutine ?
                (o as RunRoutine).routineVersion :
                undefined,
        stars: getCounts(o).stars,
        to: ['Star', 'View', 'Vote'].includes(o.type) ? (o as Star | View | Vote).to : undefined,
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
        // If "Star", "View", or "Vote", use the "to" object
        if (['Star', 'View', 'Vote'].includes(curr.type)) {
            curr = (curr as Star | View | Vote).to as ObjectListItemType;
        }
        listItems.push(<ObjectListItem
            key={`${keyPrefix}-${curr.id}`}
            beforeNavigation={beforeNavigation}
            data={curr as ObjectListItemType}
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