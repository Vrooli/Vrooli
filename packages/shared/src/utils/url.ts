import { ApiVersion, Bookmark, ChatParticipant, GqlModelType, Member, NoteVersion, Notification, ProjectVersion, Reaction, RoutineVersion, RunProject, RunRoutine, Schedule, Session, SmartContractVersion, StandardVersion, View } from "../api/generated/graphqlTypes";
import { LINKS } from "../consts/ui";
import { isOfType } from "./objects";

type Primitive = string | number | boolean | object;
export type ParseSearchParamsResult = { [x: string]: Primitive | Primitive[] | ParseSearchParamsResult | null | undefined };

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
};

/** Any object that can be displayed using ObjectListItemBase. 
 * Typically objects from the API (e.g. users, routines), but can also be events (which 
 * are calculated from schedules).
 * 
 * NOTE: This type supports partially-loaded objects, as would be the case when waiting 
 * for full data to load. This means the only known property is the __typename.
 * 
 * NOTE 2: Ideally this would be a Partial union, but a union on so many types causes a 
 * heap out of memory error  */
export type ListObject = {
    __typename: `${GqlModelType}` | "CalendarEvent";
    id?: string;
    completedAt?: number | null;
    startedAt?: number | null;
    name?: string | null;
    projectVersion?: ListObject | null;
    root?: ListObject | null;
    routineVersion?: ListObject | null;
    to?: ListObject | null;
    translations?: {
        id: string;
        language: string;
        name?: string | null;
    }[] | null;
    user?: ListObject | null;
    versions?: ListObject[] | null;
    you?: Partial<YouInflated> | null;
};

/** An object connected to routing */
export type NavigableObject = Omit<ListObject, "__typename"> & {
    __typename: `${GqlModelType}` | "Shortcut" | "Action" | "CalendarEvent",
    handle?: string | null,
    id?: string,
};

export interface ObjectOption {
    __typename: `${GqlModelType}`;
    handle?: string | null;
    id: string;
    root?: ListObject | null;
    versions?: ListObject[] | null;
    isFromHistory?: boolean;
    isBookmarked?: boolean;
    label: string;
    bookmarks?: number;
    [key: string]: any;
    runnableObject?: ListObject | null;
    to?: ListObject;
}

export interface ShortcutOption {
    __typename: "Shortcut";
    isFromHistory?: boolean;
    label: string;
    id: string; // Actually URL, but id makes it easier to use
}

export interface ActionOption {
    __typename: "Action";
    canPerform: (session: Session) => boolean;
    id: string;
    isFromHistory?: boolean;
    label: string;
}

export interface CalendarEventOption {
    __typename: "CalendarEvent";
    id: string; // Shape is <scheduleId>|<startDate>|<endDate>
    title: string;
}

export type AutocompleteOption = ObjectOption | ShortcutOption | ActionOption;

export type CalendarEvent = {
    __typename: "CalendarEvent",
    id: string;
    title: string;
    start: Date;
    end: Date;
    allDay: boolean;
    schedule: Schedule;
};

/**
 * Recursively encodes values in preparation for JSON serialization and URL encoding.
 * This function specifically targets the handling of percent signs (%) in strings
 * to ensure they are preserved accurately in the URL parameters by replacing them with '%25'.
 * This avoids issues with URL encoding where percent-encoded values might be misinterpreted.
 * 
 * - For strings, all '%' characters are replaced with '%25'.
 * - For arrays, the function is applied recursively to each element.
 * - For objects, the function is applied recursively to each value.
 * 
 * @param value - The value to be encoded. Can be of any type.
 * @returns The encoded value, with all '%' characters in strings replaced by '%25'.
 *                      Arrays and objects are handled recursively, and other types are returned as is.
 */
export const encodeValue = (value: unknown): unknown => {
    if (typeof value === "string") {
        // encodeURIComponent will skip what looks like percent-encoded values. 
        // For this reason, we must manually replace all '%' characters with '%25'
        return value.replace(/%/g, "%25");
    } else if (typeof value === "object" && value !== null) {
        if (Array.isArray(value)) {
            return value.map(encodeValue);
        }
        return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, encodeValue(v)]));
    }
    return value;
};

/**
 * Recursively processes values after JSON parsing, intended to be the inverse of encodeValue.
 * 
 * @param value - The value to be decoded, typically after JSON parsing.
 * @returns The value without any special encoding from encodeValue.
 */
export const decodeValue = (value: unknown): unknown => {
    if (typeof value === "string") {
        return value.replace(/%25/g, "%");
    } else if (typeof value === "object" && value !== null) {
        if (Array.isArray(value)) {
            return value.map(decodeValue);
        }
        return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, decodeValue(v)]));
    }
    return value;
};

/**
 * Converts object to url search params. 
 * Ignores top-level keys with null values.
 * 
 * NOTE: Values which are NOT supported are:
 * - undefined
 * - functions
 * - symbols
 * - BigInts, Infinity, -Infinity, NaN
 * - Dates
 * - Maps, Sets
 * @param params Object with key/value pairs, representing search params
 * @returns string of search params, matching the format of window.location.search
 */
export const stringifySearchParams = (params: ParseSearchParamsResult) => {
    const keys = Object.keys(params).filter(key => params[key] != null && params[key] !== undefined);
    const encodedParams = keys.map(key => {
        try {
            return `${encodeURIComponent(key)}=${encodeURIComponent(JSON.stringify(encodeValue(params[key])))}`;
        } catch (e: unknown) {
            console.error(`Error encoding value for key "${key}": ${typeof e === "string" ? e : (e as Error).message}`);
            return null;
        }
    }).filter(param => param !== null).join("&");
    return encodedParams ? `?${encodedParams}` : "";
};

/**
 * Converts url search params to object
 * @returns Object with key/value pairs, or empty object if no params
 */
export const parseSearchParams = (): ParseSearchParamsResult => {
    const params = new URLSearchParams(window.location.search);
    const obj = {};
    for (const [key, value] of params) {
        try {
            obj[decodeURIComponent(key)] = JSON.parse(decodeURIComponent(value));
        } catch (e: any) {
            console.error(`Error decoding parameter "${key}": ${e.message}`);
        }
    }
    return obj;
};

/**
 * Converts a string to a BigInt
 * @param value String to convert
 * @param radix Radix (base) to use
 * @returns 
 */
function toBigInt(value: string, radix: number) {
    return [...value.toString()]
        .reduce((r, v) => r * BigInt(radix) + BigInt(parseInt(v, radix)), 0n);
}

/**
 * Converts a UUID into a shorter, base 36 string without dashes. 
 * Useful for displaying UUIDs in a more compact format, such as in a URL.
 * @param uuid v4 UUID to convert
 * @returns base 36 string without dashes
 */
export const uuidToBase36 = (uuid: string): string => {
    try {
        const base36 = toBigInt(uuid.replace(/-/g, ""), 16).toString(36);
        return base36 === "0" ? "" : base36;
    } catch (error) {
        return "";
    }
};

/**
 * Converts a base 36 string without dashes into a UUID.
 * @param base36 base 36 string without dashes
 * @param showError Whether to show an error snack if the conversion fails
 * @returns v4 UUID
 */
export const base36ToUuid = (base36: string): string => {
    try {
        // Convert to base 16. If the ID is less than 32 characters, pad start with 0s. 
        // Then, insert dashes
        const uuid = toBigInt(base36, 36).toString(16).padStart(32, "0").replace(/(.{8})(.{4})(.{4})(.{4})(.{12})/, "$1-$2-$3-$4-$5");
        return uuid === "0" ? "" : uuid;
    } catch (error) {
        return "";
    }
};

/**
 * Gets URL base for object type
 * @param object Object to get base for
 * @returns Search URL base for object type
 */
export const getObjectUrlBase = (object: Omit<NavigableObject, "id">): string => {
    // If object is a user, use 'Profile'
    if (isOfType(object, "User", "SessionUser")) return LINKS.Profile;
    // If object is a star/vote/some other type that links to a main object, use the "to" property
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getObjectUrlBase((object as Bookmark | Reaction | View).to as NavigableObject);
    // If the object is a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getObjectUrlBase((object as RunRoutine).routineVersion as NavigableObject);
    // If the object is a run project, use the project version
    if (isOfType(object, "RunProject")) return getObjectUrlBase((object as RunProject).projectVersion as NavigableObject);
    // If the object is a member or participant, use the user
    if (isOfType(object, "Member", "ChatParticipant")) return getObjectUrlBase((object as Member | ChatParticipant).user as NavigableObject);
    // If the object is a notification, use the "link" property
    if (isOfType(object, "Notification")) return (object as Notification).link ?? "";
    // Otherwise, use __typename (or root if versioned object)
    return LINKS[object.__typename.replace("Version", "")];
};

/**
 * Determines string used to reference object in URL slug
 * @param object Object being navigated to
 * @param prefersId Whether to prefer the ID over the handle
 * @returns String used to reference object in URL slug
 */
export const getObjectSlug = (object: NavigableObject | null | undefined, prefersId = false): string => {
    // If object is an action/shortcut/event, return blank
    if (isOfType(object, "Action", "Shortcut", "CalendarEvent")) return "";
    // If object is a star/vote/some other __typename that links to a main object, use that object's slug
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getObjectSlug((object as Partial<Bookmark | Reaction | View>).to);
    // If the object is a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getObjectSlug((object as Partial<RunRoutine>).routineVersion);
    // If the object is a run project, use the project version
    if (isOfType(object, "RunProject")) return getObjectSlug((object as Partial<RunProject>).projectVersion);
    // If object has root, use the root and version
    if ((object as Partial<ApiVersion | NoteVersion | ProjectVersion | RoutineVersion | StandardVersion | SmartContractVersion>).root) {
        const root = getObjectSlug((object as Partial<ApiVersion | NoteVersion | ProjectVersion | RoutineVersion | StandardVersion | SmartContractVersion>).root);
        const versionId = uuidToBase36((object as { id?: string }).id ?? "");
        const version = prefersId ? versionId : (object as { handle?: string }).handle ?? versionId;
        return `${root}/${version}`;
    }
    // If the object is a member or chat participant, use the user's slug
    if (isOfType(object, "Member", "ChatParticipant")) return getObjectSlug((object as Partial<ChatParticipant | Member>).user);
    // If the object is a notification, return an empty string
    if (isOfType(object, "Notification")) return "";
    // Otherwise, use object's handle or id
    const id = uuidToBase36((object as { id?: string }).id ?? "");
    return prefersId ? id : (object as { handle?: string }).handle ?? id;
};

/**
 * Determines string for object's search params
 * @param object Object being navigated to
 * @returns Stringified search params for object
 */
export const getObjectSearchParams = (object: NavigableObject): string | null => {
    // If object is an action/shortcut, return blank
    if (isOfType(object, "Action", "Shortcut")) return "";
    // If object is an event, add start time to search params
    if (isOfType(object, "CalendarEvent")) return stringifySearchParams({ start: (object as CalendarEvent).start });
    // If object is a run
    if (isOfType(object, "RunProject", "RunRoutine")) return stringifySearchParams({ run: uuidToBase36((object as Partial<RunProject | RunRoutine>).id ?? "") });
    return "";
};

/**
 * Finds view page URL for any object with an id and type
 * @param object Object being navigated to
 */
export const getObjectUrl = (object: NavigableObject): string =>
    isOfType(object, "Action") ? "" :
        isOfType(object, "Shortcut", "CalendarEvent") ? ((object as ShortcutOption | CalendarEventOption).id ?? "") :
            `${getObjectUrlBase(object)}/${getObjectSlug(object)}${getObjectSearchParams(object)}`;
