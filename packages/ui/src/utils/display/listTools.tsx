import { AutocompleteOption, NavigableObject } from "types";
import { getTranslation, getUserLanguages } from "./translationTools";
import { displayDate, firstString } from "./stringTools";
import { ObjectListItem } from "components";
import { Api, ApiVersion, Comment, DotNotation, Meeting, Member, Node, NodeRoutineListItem, Note, NoteVersion, Organization, Project, ProjectVersion, Question, Quiz, Reminder, Resource, Role, Routine, RoutineVersion, RunProject, RunProjectSchedule, RunRoutine, RunRoutineSchedule, Session, SmartContract, SmartContractVersion, Standard, StandardVersion, Star, StarFor, User, UserSchedule, View, Vote } from "@shared/consts";
import { valueFromDot } from "utils/shape";
import { exists, isOfType } from "@shared/utils";

export type ListObjectType = Api |
    ApiVersion |
    Comment |
    Meeting |
    Member |
    Node | 
    NodeRoutineListItem |
    Note |
    NoteVersion |
    Organization |
    Project |
    ProjectVersion |
    Question |
    Quiz |
    Reminder |
    Resource |
    Role |
    Routine |
    RoutineVersion |
    RunProject |
    RunProjectSchedule |
    RunRoutine |
    RunRoutineSchedule |
    SmartContract |
    SmartContractVersion |
    Standard |
    StandardVersion |
    Star |
    User |
    UserSchedule |
    View |
    Vote;

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
 * Finds dot notation for the location of the "you" property in an object which contains the specified property
 * @param object An object
 * @param property A property to find in the "you" property of the object
 */
export const getYouDot = (
    object: ListObjectType | null | undefined,
    property: keyof YouInflated,
): DotNotation<typeof object> | null => {
    // If no object, return null
    if (!object) return null;
    // If the object is a star, view, or vote, use the "to" object
    if (isOfType(object, 'Star', 'View', 'Vote')) return getYouDot(object.to as ListObjectType, property);
    // If the object is a run routine, use the routine version
    if (isOfType(object, 'RunRoutine')) return getYouDot(object.routineVersion as ListObjectType, property);
    // If the object is a run project, use the project version
    if (isOfType(object, 'RunProject')) return getYouDot(object.projectVersion as ListObjectType, property);
    // Check object.you
    if (exists((object as any).you?.[property])) return 'you';
    // Check object.root.you
    if (exists((object as any).root?.you?.[property])) return 'root.you';
    // If not found, return null
    return null;
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
    // If a star, view, or vote, use the "to" object
    if (isOfType(object, 'Star', 'View', 'Vote')) return getYou(object.to as ListObjectType);
    // If a run routine, use the routine version
    if (isOfType(object, 'RunRoutine')) return getYou(object.routineVersion as ListObjectType);
    // If a run project, use the project version
    if (isOfType(object, 'RunProject')) return getYou(object.projectVersion as ListObjectType);
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
    // If a star, view, or vote, use the "to" object
    if (isOfType(object, 'Star', 'View', 'Vote')) return getCounts(object.to as ListObjectType);
    // If a run routine, use the routine version
    if (isOfType(object, 'RunRoutine')) return getCounts(object.routineVersion as ListObjectType);
    // If a run project, use the project version
    if (isOfType(object, 'RunProject')) return getCounts(object.projectVersion as ListObjectType);
    // If a NodeRoutineListItem, use the routine version
    if (isOfType(object, 'NodeRoutineListItem')) return getCounts(object.routineVersion as ListObjectType);
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
    // If a star, view, or vote, use the "to" object
    if (isOfType(object, 'Star', 'View', 'Vote')) return getDisplay(object.to as ListObjectType);
    const langs: readonly string[] = languages ?? getUserLanguages(undefined);
    // If a run routine, use the routine version's display and the startedAt/completedAt date
    if (isOfType(object, 'RunRoutine')) {
        const { completedAt, name, routineVersion, startedAt } = object;
        const title = firstString(name, getTranslation(routineVersion!, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        return {
            title: started ? `${title} (started)` : title,
            subtitle: started ? 'Started: ' + started : completed ? 'Completed: ' + completed : ''
        }
    }
    // If a run project, use the project version's display and the startedAt/completedAt date
    if (isOfType(object, 'RunProject')) {
        const { completedAt, name, projectVersion, startedAt } = object;
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
    // Priority for subtitle is: bio, description, summary, details, text, translations[number].bio, translations[number].description, translations[number].summary, translations[number].details, translations[number].text
    // If all else fails, attempt to find in "root" object
    const tryTitle = (obj: Record<string, any>) => {
        const translations: Record<string, any> = getTranslation(obj, langs, true);
        return firstString(
            obj.title,
            obj.name,
            translations.title,
            translations.name,
            obj.handle ? `$${obj.handle}` : null,
        );
    }
    const trySubtitle = (obj: Record<string, any>) => {
        const translations: Record<string, any> = getTranslation(obj, langs, true);
        return firstString(
            obj.bio,
            obj.description,
            obj.summary,
            obj.details,
            obj.text,
            translations.bio,
            translations.description,
            translations.summary,
            translations.details,
            translations.text,
        );
    }
    const title = tryTitle(object) ?? tryTitle((object as any).root) ?? '';
    const subtitle = trySubtitle(object) ?? trySubtitle((object as any).root) ?? '';
    // If a NodeRoutineListItem, use the routine version's display if title or subtitle is empty
    if (isOfType(object, 'NodeRoutineListItem') && title.length === 0 && subtitle.length === 0) {
        const routineVersionDisplay = getDisplay(object.routineVersion as ListObjectType, languages);
        return {
            title: title.length === 0 ? routineVersionDisplay.title : title,
            subtitle: subtitle.length === 0 ? routineVersionDisplay.subtitle : subtitle,
        }
    }
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
    // If a star, view, or vote, use the "to" object
    if (isOfType(object, 'Star', 'View', 'Vote')) return getStarFor(object.to as ListObjectType);
    // If a run routine, use the routine version
    if (isOfType(object, 'RunRoutine')) return getStarFor(object.routineVersion as ListObjectType);
    // If a run project, use the project version
    if (isOfType(object, 'RunProject')) return getStarFor(object.projectVersion as ListObjectType);
    // If a NodeRoutineListItem, use the routine version
    if (isOfType(object, 'NodeRoutineListItem')) return getStarFor(object.routineVersion as ListObjectType);
    // If the object contains a root object, use that
    if ((object as any).root) return getStarFor((object as any).root);
    // Use current object
    return { starFor: object.__typename as unknown as StarFor, starForId: object.id };
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
        isStarred: getYou(o).isStarred,
        label: getDisplay(o, languages).title,
        runnableObject: o.__typename === 'RunProject' ?
            o.projectVersion :
            o.__typename === 'RunRoutine' ?
                o.routineVersion :
                undefined,
        stars: getCounts(o).stars,
        to: isOfType(o, 'Star', 'View', 'Vote') ? o.to : undefined,
        versions: isOfType(o, 'Api', 'Note', 'Project', 'Routine', 'SmartContract', 'Standard') ? o.versions : undefined,
        root: isOfType(o, 'ApiVersion', 'NoteVersion', 'ProjectVersion', 'RoutineVersion', 'SmartContractVersion', 'StandardVersion') ? o.root : undefined,
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
        if (isOfType(curr, 'Star', 'View', 'Vote')) curr = curr.to as ListObjectType;
        listItems.push(<ObjectListItem
            key={`${keyPrefix}-${curr.id}`}
            beforeNavigation={beforeNavigation}
            data={curr as ListObjectType}
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