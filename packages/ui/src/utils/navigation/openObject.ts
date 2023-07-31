/**
 * Navigate to various objects and object search pages
 */

import { adaHandleRegex, Bookmark, GqlModelType, isOfType, LINKS, Reaction, RunProject, RunRoutine, urlRegex, View, walletAddressRegex } from "@local/shared";
import { SetLocation, stringifySearchParams } from "route";
import { CalendarEvent, NavigableObject } from "types";
import { ResourceType } from "utils/consts";
import { uuidToBase36 } from "./urlTools";

export type ObjectType = "Api" |
    "Bookmark" |
    "Chat" |
    "Comment" |
    "FocusMode" |
    "Meeting" |
    "Note" |
    "Organization" |
    "Project" |
    "Question" |
    "Reaction" |
    "Reminder" |
    "Routine" |
    "RunProject" |
    "RunRoutine" |
    "Schedule" |
    "SmartContract" |
    "Standard" |
    "Tag" |
    "User";

/**
 * Gets URL base for object type
 * @param object Object to get base for
 * @returns Search URL base for object type
 */
export const getObjectUrlBase = (object: Omit<NavigableObject, "id">): string => {
    // If object is a user, use 'Profile'
    if (isOfType(object, "User")) return LINKS.Profile;
    // If object is a star/vote/some other type that links to a main object, use the "to" property
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getObjectUrlBase((object as Bookmark | Reaction | View).to as NavigableObject);
    // If the object is a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getObjectUrlBase((object as RunRoutine).routineVersion as NavigableObject);
    // If the object is a run project, use the project version
    if (isOfType(object, "RunProject")) return getObjectUrlBase((object as RunProject).projectVersion as NavigableObject);
    // Otherwise, use __typename (or root if versioned object)
    return LINKS[object.__typename.replace("Version", "")];
};

type SluggableObject = {
    __typename: `${GqlModelType}` | "Action" | "Shortcut" | "CalendarEvent",
    id: string,
    handle?: string | null,
    root?: { handle?: string | null, id: string } | null,
}

/**
 * Determines string used to reference object in URL slug
 * @param object Object being navigated to
 * @returns String used to reference object in URL slug
 */
export const getObjectSlug = (object: SluggableObject): string => {
    // If object is an action/shortcut/event, return blank
    if (isOfType(object, "Action", "Shortcut", "CalendarEvent")) return "";
    // If object is a star/vote/some other __typename that links to a main object, use that object's slug
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getObjectSlug((object as Bookmark | Reaction | View).to as SluggableObject);
    // If the object is a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getObjectSlug((object as RunRoutine).routineVersion as SluggableObject);
    // If the object is a run project, use the project version
    if (isOfType(object, "RunProject")) return getObjectSlug((object as RunProject).projectVersion as SluggableObject);
    // If object has root, use its handle or id
    if (object.root) {
        const root = object.root.handle ?? uuidToBase36(object.root.id);
        const verion = uuidToBase36(object.id);
        return `${root}/${verion}`;
    }
    // Otherwise, use object's handle or id
    return object.handle ?? uuidToBase36(object.id);
};

type SearchParamableObject = {
    __typename: `${GqlModelType}` | "Action" | "Shortcut" | "CalendarEvent",
    id: string,
}

/**
 * Determines string for object's search params
 * @param object Object being navigated to
 * @returns Stringified search params for object
 */
export const getObjectSearchParams = (object: SearchParamableObject): string | null => {
    // If object is an action/shortcut, return blank
    if (isOfType(object, "Action", "Shortcut")) return "";
    // If object is an event, add start time to search params
    if (isOfType(object, "CalendarEvent")) return stringifySearchParams({ start: (object as CalendarEvent).start });
    // If object is a run
    if (isOfType(object, "RunProject", "RunRoutine")) return stringifySearchParams({ run: uuidToBase36(object.id) });
    return "";
};
// Omit<NavigableObject, '__typename'> & { __typename: `${GqlModelType}`}

/**
 * Finds view page URL for any object with an id and type
 * @param object Object being navigated to
 */
export const getObjectUrl = (object: NavigableObject): string =>
    isOfType(object, "Action") ? "" :
        isOfType(object, "Shortcut", "CalendarEvent") ? object.id :
            `${getObjectUrlBase(object)}/${getObjectSlug(object)}${getObjectSearchParams(object)}`;

/**
 * Opens any object with an id and type
 * @param object Object to open
 * @param setLocation Function to set location in history
 */
export const openObject = (object: NavigableObject, setLocation: SetLocation) => !isOfType(object, "Action") && setLocation(getObjectUrl(object));

/**
 * Finds edit page URL for any object with an id and type
 * @param object Object being navigated to
 */
export const getObjectEditUrl = (object: NavigableObject) => `${getObjectUrlBase(object)}/edit/${getObjectSlug(object)}${getObjectSearchParams(object)}`;

/**
 * Opens the edit page for an object with an id and type
 * @param object Object to open
 * @param setLocation Function to set location in history
 */
export const openObjectEdit = (object: NavigableObject, setLocation: SetLocation) => setLocation(getObjectEditUrl(object));

/**
 * Finds report page URL for any object with an id and type
 * @param object Object being navigated to
 */
export const getObjectReportUrl = (object: NavigableObject) => `${getObjectUrlBase(object)}/reports/${getObjectSlug(object)}`;

/**
 * Opens the report page for an object with an id and type.
 * 
 * NOTE: For VIEWING reports, not creating them
 * @param object Object to open
 */
export const openObjectReport = (object: NavigableObject, setLocation: SetLocation) => setLocation(getObjectReportUrl(object));

/**
 * Determines if a resource is a URL, wallet payment address, or an ADA handle
 * @param link String to check
 * @returns ResourceType if type found, or null if not
 */
export const getResourceType = (link: string): ResourceType | null => {
    if (urlRegex.test(link)) return ResourceType.Url;
    if (walletAddressRegex.test(link)) return ResourceType.Wallet;
    if (adaHandleRegex.test(link)) return ResourceType.Handle;
    return null;
};

/**
 * Finds the URL for a resource
 * @param link Resource string. May be a URL, handle, or wallet address
 * @returns link as a URL (i.e. wallet opens in cardanoscan, handle opens in handle.me)
 */
export const getResourceUrl = (link: string): string | undefined => {
    const resourceType = getResourceType(link);
    if (resourceType === ResourceType.Url) return link;
    if (resourceType === ResourceType.Handle) return `https://handle.me/${link}`;
    if (resourceType === ResourceType.Wallet) return `https://cardanoscan.io/address/${link}`;
    return undefined;
};
