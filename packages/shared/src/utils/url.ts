import { ApiVersion, Bookmark, ChatParticipant, Code, CodeType, CodeVersion, GqlModelType, Member, NoteVersion, Notification, ProjectVersion, Reaction, Routine, RoutineType, RoutineVersion, RunProject, RunRoutine, Schedule, Session, Standard, StandardType, StandardVersion, View } from "../api/generated/graphqlTypes";
import { LINKS } from "../consts/ui";
import { isOfType } from "./objects";

export type UrlPrimitive = string | number | boolean | object;
export type ParseSearchParamsResult = { [x: string]: UrlPrimitive | UrlPrimitive[] | ParseSearchParamsResult | null | undefined };

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

export type AutocompleteOption = (ObjectOption | ShortcutOption | ActionOption) & {
    key: string;
};

export type CalendarEvent = {
    __typename: "CalendarEvent",
    id: string;
    title: string;
    start: Date;
    end: Date;
    allDay: boolean;
    schedule: Schedule;
};


export enum RunTypeInPath {
    Project = "p",
    Routine = "r",
}

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
export function encodeValue(value: unknown): unknown {
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
}

/**
 * Recursively processes values after JSON parsing, intended to be the inverse of encodeValue.
 * 
 * @param value - The value to be decoded, typically after JSON parsing.
 * @returns The value without any special encoding from encodeValue.
 */
export function decodeValue(value: unknown): unknown {
    if (typeof value === "string") {
        return value.replace(/%25/g, "%");
    } else if (typeof value === "object" && value !== null) {
        if (Array.isArray(value)) {
            return value.map(decodeValue);
        }
        return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, decodeValue(v)]));
    }
    return value;
}

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
export function stringifySearchParams(params: ParseSearchParamsResult) {
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
}

/**
 * Converts url search params to object
 * @returns Object with key/value pairs, or empty object if no params
 */
export function parseSearchParams(): ParseSearchParamsResult {
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
}

/**
 * Converts a string to a BigInt
 * @param value String to convert
 * @param radix Radix (base) to use
 * @returns 
 */
function toBigInt(value: string, radix: number) {
    return [...value.toString()]
        // eslint-disable-next-line no-magic-numbers
        .reduce((r, v) => r * BigInt(radix) + BigInt(parseInt(v, radix)), 0n);
}

/**
 * Converts a UUID into a shorter, base 36 string without dashes. 
 * Useful for displaying UUIDs in a more compact format, such as in a URL.
 * @param uuid v4 UUID to convert
 * @returns base 36 string without dashes
 */
export function uuidToBase36(uuid: string): string {
    try {
        // eslint-disable-next-line no-magic-numbers
        const base36 = toBigInt(uuid.replace(/-/g, ""), 16).toString(36);
        return base36 === "0" ? "" : base36;
    } catch (error) {
        return "";
    }
}

/**
 * Converts a base 36 string without dashes into a UUID.
 * @param base36 base 36 string without dashes
 * @param showError Whether to show an error snack if the conversion fails
 * @returns v4 UUID
 */
export function base36ToUuid(base36: string): string {
    try {
        // Convert to base 16. If the ID is less than 32 characters, pad start with 0s. 
        // Then, insert dashes
        // eslint-disable-next-line no-magic-numbers
        const uuid = toBigInt(base36, 36).toString(16).padStart(32, "0").replace(/(.{8})(.{4})(.{4})(.{4})(.{12})/, "$1-$2-$3-$4-$5");
        return uuid === "0" ? "" : uuid;
    } catch (error) {
        return "";
    }
}

/**
 * Gets URL base for object type
 * @param object Object to get base for
 * @returns Search URL base for object type
 */
export function getObjectUrlBase(object: Omit<NavigableObject, "id">): string {
    if (typeof object !== "object" || object === null || !Object.prototype.hasOwnProperty.call(object, "__typename")) return "";
    // If object is a user, use 'Profile'
    if (isOfType(object, "User", "SessionUser")) return LINKS.Profile;
    // If object is a code, base URL on its type
    if (isOfType(object, "CodeVersion")) return (object as CodeVersion).codeType === CodeType.DataConvert ? LINKS.DataConverter : LINKS.SmartContract;
    if (isOfType(object, "Code")) return (object as Code).versions?.find(v => v.isLatest === true)?.codeType === CodeType.DataConvert ? LINKS.DataConverter : LINKS.SmartContract;
    // If object is a routine, base URL on its type
    if (isOfType(object, "RoutineVersion")) return (object as RoutineVersion).routineType === RoutineType.MultiStep ? LINKS.RoutineMultiStep : LINKS.RoutineSingleStep;
    if (isOfType(object, "Routine")) return (object as Routine).versions?.find(v => v.isLatest === true)?.routineType === RoutineType.MultiStep ? LINKS.RoutineMultiStep : LINKS.RoutineSingleStep;
    // If object is a standard, base URL on its type
    if (isOfType(object, "StandardVersion")) return (object as StandardVersion).variant === StandardType.Prompt ? LINKS.Prompt : LINKS.DataStructure;
    if (isOfType(object, "Standard")) return (object as Standard).versions?.find(v => v.isLatest === true)?.variant === StandardType.Prompt ? LINKS.Prompt : LINKS.DataStructure;
    // If object is a star/vote/some other type that links to a main object, use the "to" property
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getObjectUrlBase((object as Bookmark | Reaction | View).to as NavigableObject);
    // If the object is a member or participant, use the user
    if (isOfType(object, "Member", "ChatParticipant")) return getObjectUrlBase((object as Member | ChatParticipant).user as NavigableObject);
    // If the object is a notification, use the "link" property
    if (isOfType(object, "Notification")) return (object as Notification).link ?? "";
    // If the object is a run
    if (isOfType(object, "RunProject", "RunRoutine")) {
        // Use run page for multi-step routines
        if (isOfType(object, "RunProject") || (object as RunRoutine).routineVersion?.routineType === RoutineType.MultiStep) {
            return LINKS.Run;
        }
        // Otherwise, use routine view page
        return LINKS.RoutineSingleStep;
    }
    // Otherwise, use __typename (or root if versioned object)
    return LINKS[object.__typename.replace("Version", "")];
}

/**
 * Determines string used to reference object in URL slug
 * @param object Object being navigated to
 * @param prefersId Whether to prefer the ID over the handle
 * @returns String used to reference object in URL slug
 */
export function getObjectSlug(object: NavigableObject | null | undefined, prefersId = false): string {
    if (typeof object !== "object" || object === null) return "";
    // If object is an action/shortcut/event, return blank
    if (isOfType(object, "Action", "Shortcut", "CalendarEvent")) return "";
    // If object is a star/vote/some other __typename that links to a main object, use that object's slug
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getObjectSlug((object as Partial<Bookmark | Reaction | View>).to);
    // If object has root, use the root and version
    if ((object as Partial<ApiVersion | CodeVersion | NoteVersion | ProjectVersion | RoutineVersion | StandardVersion>).root) {
        const root = getObjectSlug((object as Partial<ApiVersion | CodeVersion | NoteVersion | ProjectVersion | RoutineVersion | StandardVersion>).root);
        const versionId = uuidToBase36((object as { id?: string }).id ?? "");
        const version = prefersId ? versionId : (object as { handle?: string }).handle ?? versionId;
        return `${root}/${version}`;
    }
    // If the object is a member or chat participant, use the user's slug
    if (isOfType(object, "Member", "ChatParticipant")) return getObjectSlug((object as Partial<ChatParticipant | Member>).user);
    // If the object is a notification, return an empty string
    if (isOfType(object, "Notification")) return "";
    // Handle run slugs
    if (isOfType(object, "RunProject")) {
        return `${RunTypeInPath.Project}/${uuidToBase36((object as RunProject).id)}`;
    }
    if (isOfType(object, "RunRoutine")) {
        if ((object as RunRoutine).routineVersion?.routineType === RoutineType.MultiStep) {
            return `${RunTypeInPath.Routine}/${uuidToBase36((object as RunRoutine).id)}`;
        }
        // Use routine id for single-step routines (run id will be in search params)
        return getObjectSlug((object as RunRoutine).routineVersion);
    }
    // Otherwise, use object's handle or id
    const id = uuidToBase36((object as { id?: string }).id ?? "");
    return prefersId ? id : (object as { handle?: string }).handle ?? id;
}

/**
 * Determines string for object's search params
 * @param object Object being navigated to
 * @returns Stringified search params for object
 */
export function getObjectSearchParams(object: NavigableObject): string | null {
    // If object is an event, add start time to search params
    if (isOfType(object, "CalendarEvent")) {
        return UrlTools.stringifySearchParams(LINKS.Calendar, { start: (object as CalendarEvent).start });
    }
    // If object is a run
    if (isOfType(object, "RunProject", "RunRoutine")) {
        // Use run page for multi-step routines
        if (isOfType(object, "RunProject") || (object as RunRoutine).routineVersion?.routineType === RoutineType.MultiStep) {
            return UrlTools.stringifySearchParams(LINKS.Run, { step: (object as RunProject | RunRoutine).lastStep ?? undefined });
        }
        // Otherwise, use routine view page
        return UrlTools.stringifySearchParams(LINKS.RoutineSingleStep, { runId: uuidToBase36((object as RunRoutine).id) });
    }
    return "";
}

/**
 * Finds view page URL for any object with an id and type
 * @param object Object being navigated to
 */
export function getObjectUrl(object: NavigableObject): string {
    if (isOfType(object, "Action")) return "";
    if (isOfType(object, "Shortcut", "CalendarEvent")) return (object as ShortcutOption | CalendarEventOption).id ?? "";
    const base = getObjectUrlBase(object);
    let slug = getObjectSlug(object);
    if (slug.length > 0 && !slug.startsWith("/")) slug = `/${slug}`;
    const search = getObjectSearchParams(object);
    return `${base}${slug}${search}`;
}


export type CalendarViewSearchParams = {
    start?: Date | undefined;
    end?: Date | undefined;
}

export type LoginViewSearchParams = {
    redirect?: string | undefined;
}

export type ResetPasswordViewSearchParams = {
    code?: string | undefined;
}

export type RunViewSearchParams = {
    step?: number[] | undefined;
}

export type RoutineSingleStepViewSearchParams = {
    runId?: string | undefined;
}

export type RoutineMultiStepViewSearchParams = {
    runId?: string | undefined;
}

export type SignUpViewSearchParams = {
    redirect?: string | undefined;
}

export type TutorialViewSearchParams = {
    tutorial_section?: number | undefined;
    tutorial_step?: number | undefined;
}

export interface SearchParamsPayloads {
    [LINKS.Calendar]: CalendarViewSearchParams;
    [LINKS.Login]: LoginViewSearchParams;
    [LINKS.ResetPassword]: ResetPasswordViewSearchParams;
    [LINKS.Run]: RunViewSearchParams;
    [LINKS.RoutineSingleStep]: RoutineSingleStepViewSearchParams;
    [LINKS.RoutineMultiStep]: RoutineMultiStepViewSearchParams;
    [LINKS.Signup]: SignUpViewSearchParams;
    Tutorial: TutorialViewSearchParams;
}

const defaultSearchParams: Partial<{ [K in keyof SearchParamsPayloads]: SearchParamsPayloads[K] }> = {
    // Add default search params here
};

export type SearchParamsType = keyof SearchParamsPayloads;

export class UrlTools {
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Enforces that the provided search params match the type
     */
    static stringifySearchParams<T extends SearchParamsType>(
        link: T,
        searchParams: SearchParamsPayloads[T] = defaultSearchParams[link] as SearchParamsPayloads[T],
    ): string {
        return stringifySearchParams(searchParams);
    }

    static parseSearchParams<T extends SearchParamsType>(link: T): SearchParamsPayloads[T] {
        return parseSearchParams() as SearchParamsPayloads[T];
    }

    /**
     * Creates a URL with search params
     */
    static linkWithSearchParams<T extends SearchParamsType>(
        link: T,
        searchParams: SearchParamsPayloads[T] = defaultSearchParams[link] as SearchParamsPayloads[T],
    ): string {
        return `${link}${UrlTools.stringifySearchParams(link, searchParams)}`;
    }
}
