/**
 * Navigate to various objects and object search pages
 */

import { APP_LINKS } from "@shared/consts";
import { adaHandleRegex, urlRegex, walletAddressRegex } from "@shared/validation";
import { NavigableObject, SetLocation } from "types";
import { ResourceType } from "utils/consts";
import { stringifySearchParams, uuidToBase36 } from "./urlTools";

export type ObjectType = 'Comment' |
    'Organization' |
    'Project' |
    'Routine' |
    'Run' |
    'Standard' |
    'Star' |
    'Tag' |
    'User' |
    'View';

/**
 * Gets URL base for object type
 * @param object Object to get base for
 * @returns Search URL base for object type
 */
export const getObjectUrlBase = (object: Omit<NavigableObject, 'id'>): string => {
    switch (object.type) {
        case 'Organization':
            return APP_LINKS.Organization;
        case 'Project':
            return APP_LINKS.Project;
        case 'Routine':
            return APP_LINKS.Routine;
        case 'Standard':
            return APP_LINKS.Standard;
        case 'User':
            return APP_LINKS.Profile;
        case 'Star':
        case 'View':
            return getObjectUrlBase(object.to as any);
        case 'Run':
            return getObjectUrlBase({
                type: 'Routine',
                id: object.routine?.id,
            } as any);
        default:
            return '';
    }
}

/**
 * Determines string used to reference object in URL slug
 * @param object Object being navigated to
 * @returns String used to reference object in URL slug
 */
export const getObjectSlug = (object: NavigableObject): string => {
    // If object is a star/vote/some other type that links to a main object, use that object's slug
    if (object.to) return getObjectSlug(object.to);
    // If object is a run, navigate to the routine
    if (object.routine) return getObjectSlug(object.routine);
    // If object has a handle, use that (Note: objects with handles don't have versioning, so we don't need to worry about that)
    if (object.handle) return object.handle;
    // If object has a versionGroupId, and an id, use versionGroupId/id
    if (object.versionGroupId && object.id) return `${uuidToBase36(object.versionGroupId)}/${uuidToBase36(object.id)}`;
    // If object only has a versionGroupId, use that
    if (object.versionGroupId) return uuidToBase36(object.versionGroupId);
    // Otherwise, use the id
    return uuidToBase36(object.id);
}

/**
 * Determines string for object's search params
 * @param object Object being navigated to
 * @returns Stringified search params for object
 */
export const getObjectSearchParams = (object: any) => {
    // If object is a run
    if (object.type === 'RunRoutine') return stringifySearchParams({ run: uuidToBase36(object.id) });
    return '';
}

/**
 * Finds view page URL for any object with an id and type
 * @param object Object being navigated to
 */
export const getObjectUrl = (object: NavigableObject) => `${getObjectUrlBase(object)}/${getObjectSlug(object)}${getObjectSearchParams(object)}`;

/**
 * Opens any object with an id and type
 * @param object Object to open
 * @param setLocation Function to set location in history
 */
export const openObject = (object: NavigableObject, setLocation: SetLocation) => setLocation(getObjectUrl(object));

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
 * Opens the report page for an object with an id and type
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
}

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
}