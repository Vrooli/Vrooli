import { BookmarkFor, CommonKey, DotNotation, exists, GqlModelType, isOfType } from "@local/shared";
import { ObjectListItem } from "components/lists/ObjectListItem/ObjectListItem";
import { ActionsType, ListActions, SearchListGenerator } from "components/lists/types";
import { AutocompleteOption, NavigableObject } from "types";
import { SearchType } from "utils/search/objectToSearch";
import { valueFromDot } from "utils/shape/general";
import { displayDate, firstString } from "./stringTools";
import { getTranslation, getUserLanguages } from "./translationTools";

// NOTE: Ideally this would be a union of all possible types, but there's actually so 
// many types that it causes a heap out of memory error :(
export type ListObjectType = {
    __typename: `${GqlModelType}` | "CalendarEvent";
    completedAt?: number | null;
    startedAt?: number | null;
    name?: string | null;
    projectVersion?: ListObjectType | null;
    root?: ListObjectType | null;
    routineVersion?: ListObjectType | null;
    translations?: {
        id: string;
        language: string;
        name?: string | null;
    }[] | null;
    user?: ListObjectType | null;
    versions?: ListObjectType[] | null;
    you?: Partial<YouInflated> | null;
} & Omit<NavigableObject, "__typename">;

/**
 * All possible permissions/user-statuses any object can have
 */
export type YouInflated = {
    canComment: boolean;
    canCopy: boolean;
    canDelete: boolean;
    canRead: boolean;
    canReport: boolean;
    canShare: boolean;
    canBookmark: boolean;
    canUpdate: boolean;
    canReact: boolean;
    isBookmarked: boolean;
    isViewed: boolean;
    reaction: string | null; // Your reaction to object, or thumbs up/down if vote instead of reaction
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
    bookmarks: number;
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
    if (isOfType(object, "Bookmark", "View", "Vote")) return getYouDot(object.to as ListObjectType, property);
    // If the object is a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getYouDot(object.routineVersion as ListObjectType, property);
    // If the object is a run project, use the project version
    if (isOfType(object, "RunProject")) return getYouDot(object.projectVersion as ListObjectType, property);
    // Check object.you
    if (exists((object as any).you?.[property])) return `you.${property}`;
    // Check object.root.you
    if (exists((object as any).root?.you?.[property])) return `root.you.${property}`;
    // If not found, return null
    return null;
};

export const defaultYou: YouInflated = {
    canComment: false,
    canCopy: false,
    canDelete: false,
    canRead: false,
    canReport: false,
    canShare: false,
    canBookmark: false,
    canUpdate: false,
    canReact: false,
    isBookmarked: false,
    isViewed: false,
    reaction: null,
};

/**
 * Gets user permissions and statuses for an object. These are inflated to match YouInflated, so any fields not present are false
 * @param object An object
 */
export const getYou = (
    object: ListObjectType | null | undefined,
): YouInflated => {
    // Initialize fields to false (except reaction, since that's an emoji or null instead of a boolean)
    const defaultPermissions = { ...defaultYou };
    if (!object) return defaultPermissions;
    // If a star, view, or vote, use the "to" object
    if (isOfType(object, "Bookmark", "View", "Vote")) return getYou(object.to as ListObjectType);
    // If a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getYou(object.routineVersion as ListObjectType);
    // If a run project, use the project version
    if (isOfType(object, "RunProject")) return getYou(object.projectVersion as ListObjectType);
    // Otherwise, get the permissions from the object
    // Loop through all permission fields
    for (const key in defaultPermissions) {
        // Check if the field is in the object
        const field = valueFromDot(object, `you.${key}`);
        if (field === true || field === false || typeof field === "string") defaultPermissions[key] = field;
        // If not, check if the field is in the root.you object
        else {
            const field = valueFromDot(object, `root.you.${key}`);
            if (field === true || field === false || typeof field === "string") defaultPermissions[key] = field;
        }
    }
    return defaultPermissions;
};

/**
 * Gets counts for an object. These are inflated to match CountsInflated, so any fields not present are 0
 * @param object An object
 */
export const getCounts = (
    object: ListObjectType | null | undefined,
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
        bookmarks: 0,
        transfers: 0,
        translations: 0,
        versions: 0,
        views: 0,
    };
    if (!object) return defaultCounts;
    // If a star, view, or vote, use the "to" object
    if (isOfType(object, "Bookmark", "View", "Vote")) return getCounts(object.to as ListObjectType);
    // If a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getCounts(object.routineVersion as ListObjectType);
    // If a run project, use the project version
    if (isOfType(object, "RunProject")) return getCounts(object.projectVersion as ListObjectType);
    // If a NodeRoutineListItem, use the routine version
    if (isOfType(object, "NodeRoutineListItem")) return getCounts(object.routineVersion as ListObjectType);
    // Otherwise, get the counts from the object
    // Loop through all count fields
    for (const key in defaultCounts) {
        // For every field except score and views, property name is field + "Count"
        const objectProp = ["score", "views"].includes(key) ? key : `${key}Count`;
        // Check if the field is in the object
        const field = valueFromDot(object, objectProp);
        if (field !== undefined) defaultCounts[key] = field;
        // If not, check if the field is in the root.counts object
        else {
            const field = valueFromDot(object, `root.${objectProp}`);
            if (field !== undefined) defaultCounts[key] = field;
        }
    }
    return defaultCounts;
};

/**
 * Attempts to find the most relevant title for an object. Does not check root object or versions
 * @param obj An object with (hopefully) a title
 * @param langs The user's preferred languages
 * @returns The title, or null if none found
 */
const tryTitle = (obj: Record<string, any>, langs: readonly string[]) => {
    const translations: Record<string, any> = getTranslation(obj, langs, true);
    // The order of these is important to display the most relevant title
    return firstString(
        obj.title,
        obj.name,
        obj.label,
        translations.title,
        translations.name,
        translations.label,
        obj.handle ? `$${obj.handle}` : null,
    );
};

/**
 * Attempts to find the most relevant subtitle for an object. Does not check root object or versions
 * @param obj An object with (hopefully) a subtitle
 * @param langs The user's preferred languages
 * @returns The subtitle, or null if none found
 */
const trySubtitle = (obj: Record<string, any>, langs: readonly string[]) => {
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
};

/**
 * For an object which does not have a direct title (i.e. it's likely in the root object or a version), 
 * tries to find the most relevant title and subtitle
 * @param obj An object
 * @param langs The user's preferred languages
 * @returns The title and subtitle, or blank strings if none found
 */
const tryVersioned = (obj: Record<string, any>, langs: readonly string[]) => {
    // Initialize the title and subtitle
    let title: string | null = null;
    let subtitle: string | null = null;
    // Create a list of objects to check. Order is important
    const objectsToCheck = [
        obj, // The object itself
        obj.root, // The root object (only found if obj is a version)
        obj.versions?.find(v => v.isLatest), // The latest version (only found if obj is a root object)
        ...([...(obj.versions ?? [])].sort((a, b) => b.versionIndex - a.versionIndex)), // All versions, sorted by versionIndex (i.e. newest first)
    ];
    // Loop through the objects
    for (const curr of objectsToCheck) {
        // If the object is null or undefined, skip it
        if (!exists(curr)) continue;
        // Check for a title if we haven't found one yet
        if (!title) {
            title = tryTitle(curr, langs);
        }
        // Check for a subtitle if we haven't found one yet
        if (!subtitle) {
            subtitle = trySubtitle(curr, langs);
        }
        // If both are found, break
        if (title && subtitle) break;
    }
    return { title: title ?? "", subtitle: subtitle ?? "" };
};

/**
 * Gets the name and subtitle of a list object
 * @param object A list object
 * @param languages User languages
 * @returns The name and subtitle of the object
 */
export const getDisplay = (
    object: ListObjectType | null | undefined,
    languages?: readonly string[],
): { title: string, subtitle: string } => {
    if (!object) return { title: "", subtitle: "" };
    // If a star, view, or vote, use the "to" object
    if (isOfType(object, "Bookmark", "View", "Vote")) return getDisplay(object.to as ListObjectType);
    const langs: readonly string[] = languages ?? getUserLanguages(undefined);
    // If a run routine, use the routine version's display and the startedAt/completedAt date
    if (isOfType(object, "RunRoutine")) {
        const { completedAt, name, routineVersion, startedAt } = object;
        const title = firstString(name, getTranslation(routineVersion!, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        const { subtitle: routineVersionSubtitle } = getDisplay(routineVersion!, langs);
        return {
            title,
            subtitle: (started ? "Started: " + started : completed ? "Completed: " + completed : "") + (routineVersionSubtitle ? " | " + routineVersionSubtitle : ""),
        };
    }
    // If a run project, use the project version's display and the startedAt/completedAt date
    if (isOfType(object, "RunProject")) {
        const { completedAt, name, projectVersion, startedAt } = object;
        const title = firstString(name, getTranslation(projectVersion!, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        const { subtitle: projectVersionSubtitle } = getDisplay(projectVersion!, langs);
        return {
            title,
            subtitle: (started ? "Started: " + started : completed ? "Completed: " + completed : "") + (projectVersionSubtitle ? " | " + projectVersionSubtitle : ""),
        };
    }
    // If a member or chat participant, use the user's display
    if (isOfType(object, "Member", "ChatParticipant")) return getDisplay({ __typename: "User", ...object.user } as ListObjectType);
    // For all other objects, fields may differ. 
    const { title, subtitle } = tryVersioned(object, langs);
    // If a NodeRoutineListItem, use the routine version's display if title or subtitle is empty
    if (isOfType(object, "NodeRoutineListItem") && title.length === 0 && subtitle.length === 0) {
        const routineVersionDisplay = getDisplay({ __typename: "RoutineVersion", ...object.routineVersion } as ListObjectType, langs);
        return {
            title: firstString(title, routineVersionDisplay.title),
            subtitle: firstString(subtitle, routineVersionDisplay.subtitle),
        };
    }
    return { title, subtitle };
};

/**
 * Finds the information required to bookmark an object
 * @param object 
 * @returns BookmarkFor type and ID of the object. For versions, for example, 
 * the ID is of the root object instead of the version passed in.
 */
export const getBookmarkFor = (
    object: ListObjectType | null | undefined,
): { bookmarkFor: BookmarkFor, starForId: string } | { bookmarkFor: null, starForId: null } => {
    if (!object) return { bookmarkFor: null, starForId: null };
    // If object does not support bookmarking, return null
    if (isOfType(object, "BookmarkList", "Member")) return { bookmarkFor: null, starForId: null }; //TODO add more types
    // If a star, view, or vote, use the "to" object
    if (isOfType(object, "Bookmark", "View", "Vote")) return getBookmarkFor(object.to as ListObjectType);
    // If a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getBookmarkFor(object.routineVersion as ListObjectType);
    // If a run project, use the project version
    if (isOfType(object, "RunProject")) return getBookmarkFor(object.projectVersion as ListObjectType);
    // If a NodeRoutineListItem, use the routine version
    if (isOfType(object, "NodeRoutineListItem")) return getBookmarkFor(object.routineVersion as ListObjectType);
    // If the object contains a root object, use that
    if ((object as any).root) return getBookmarkFor((object as any).root);
    // Use current object
    return { bookmarkFor: object.__typename as unknown as BookmarkFor, starForId: object.id };
};

/**
 * Converts a list of GraphQL objects to a list of autocomplete information.
 * @param objects The list of search results
 * @param languages User languages
 * @returns The list of autocomplete information
 */
export function listToAutocomplete(
    objects: readonly ListObjectType[],
    languages: readonly string[],
): AutocompleteOption[] {
    return objects.map(o => ({
        __typename: o.__typename as any,
        id: o.id,
        isBookmarked: getYou(o).isBookmarked,
        label: getDisplay(o, languages).title,
        runnableObject: o.__typename === "RunProject" ?
            o.projectVersion :
            o.__typename === "RunRoutine" ?
                o.routineVersion :
                undefined,
        bookmarks: getCounts(o).bookmarks,
        to: isOfType(o, "Bookmark", "View", "Vote") ? o.to : undefined,
        user: isOfType(o, "Member") ? o.user : undefined,
        versions: isOfType(o, "Api", "Note", "Project", "Routine", "SmartContract", "Standard") ? o.versions : undefined,
        root: isOfType(o, "ApiVersion", "NoteVersion", "ProjectVersion", "RoutineVersion", "SmartContractVersion", "StandardVersion") ? o.root : undefined,
    }));
}


export type ListToListItemProps<T extends keyof ListActions | undefined> = {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    canNavigate?: (item: NavigableObject) => boolean | void,
    /**
     * List of dummy items types to display while loading
     */
    dummyItems?: (GqlModelType | `${GqlModelType}`)[];
    /**
     * True if update button should be hidden
     */
    hideUpdateButton?: boolean,
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
    onClick?: (item: NavigableObject) => void,
    zIndex: number,
} & (T extends keyof ListActions ? ActionsType<ListActions[T & keyof ListActions]> : object);

/**
 * Converts a list of objects to a list of ListItems
 * @returns A list of ListItems
 */
export function listToListItems<T extends keyof ListActions | undefined>({
    canNavigate,
    dummyItems,
    keyPrefix,
    hideUpdateButton,
    items,
    loading,
    onClick,
    zIndex,
}: ListToListItemProps<T>): JSX.Element[] {
    const listItems: JSX.Element[] = [];
    // If loading, display dummy items
    if (loading) {
        if (!dummyItems) return listItems;
        for (let i = 0; i < dummyItems.length; i++) {
            listItems.push(<ObjectListItem
                key={`${keyPrefix}-${i}`}
                data={null}
                hideUpdateButton={hideUpdateButton}
                loading={true}
                objectType={dummyItems[i]}
                zIndex={zIndex}
            />);
        }
    }
    if (!items) return listItems;
    for (let i = 0; i < items.length; i++) {
        let curr = items[i];
        // If "Star", "View", or "Vote", use the "to" object
        if (isOfType(curr, "Bookmark", "View", "Vote")) curr = curr.to as ListObjectType;
        listItems.push(<ObjectListItem
            key={`${keyPrefix}-${curr.id}`}
            canNavigate={canNavigate}
            data={curr as ListObjectType}
            hideUpdateButton={hideUpdateButton}
            loading={false}
            objectType={curr.__typename as any}
            onClick={onClick}
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
];

/**
 * Finds a random color for a placeholder icon
 * @returns [background color code, silhouette color code]
 */
export const placeholderColor = (): [string, string] => {
    return placeholderColors[Math.floor(Math.random() * placeholderColors.length)];
};

/**
 * Creates object containing information required to display a search list 
 * for an object type.
 */
export const toSearchListData = (
    searchType: SearchType | `${SearchType}`,
    placeholder: CommonKey,
    where: Record<string, any>,
): SearchListGenerator => ({
    searchType,
    placeholder,
    where,
});
