/**
 * Navigate to various objects and object search pages
 */

import { NavigableObject, getObjectSearchParams, getObjectSlug, getObjectUrl, getObjectUrlBase, handleRegex, isOfType, urlRegex, urlRegexDev, walletAddressRegex } from "@local/shared";
import { SetLocation } from "route";
import { ResourceType } from "utils/consts";
import { setCookiePartialData } from "utils/localStorage";

export type ObjectType = "Api" |
    "Bookmark" |
    "Chat" |
    "ChatInvite" |
    "Code" |
    "Comment" |
    "FocusMode" |
    "Meeting" |
    "MemberInvite" |
    "Note" |
    "Project" |
    "Question" |
    "Reaction" |
    "Reminder" |
    "Routine" |
    "RunProject" |
    "RunRoutine" |
    "Schedule" |
    "Standard" |
    "Tag" |
    "Team" |
    "User";

/**
 * Opens any object with an id and type
 * @param object Object to open
 * @param setLocation Function to set location in history
 */
export const openObject = (object: NavigableObject, setLocation: SetLocation) => {
    // Actions don't open to anything
    if (isOfType(object, "Action")) return;
    // Store object in local storage, so we can display it while the full data loads
    setCookiePartialData(object, "list");
    // Navigate to object
    setLocation(getObjectUrl(object));
};

/**
 * Finds edit page URL for any object with an id and type
 * @param object Object being navigated to
 */
export const getObjectEditUrl = (object: NavigableObject) => `${getObjectUrlBase(object)}/edit/${getObjectSlug(object, true)}${getObjectSearchParams(object)}`;

/**
 * Opens the edit page for an object with an id and type
 * @param object Object to open
 * @param setLocation Function to set location in history
 */
export const openObjectEdit = (object: NavigableObject, setLocation: SetLocation) => {
    // Actions don't open to anything
    if (isOfType(object, "Action")) return;
    // Store object in local storage, so we can display it while the full data loads
    setCookiePartialData(object, "list");
    // Navigate to object
    setLocation(getObjectEditUrl(object));
};

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
    if (urlRegex.test(link) || urlRegexDev.test(link)) return ResourceType.Url;
    if (walletAddressRegex.test(link)) return ResourceType.Wallet;
    if (handleRegex.test(link)) return ResourceType.Handle;
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
