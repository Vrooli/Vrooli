/**
 * Navigate to various objects and object search pages
 */

import { APP_LINKS } from "@shared/consts";
import { SnackSeverity } from "components";
import { SetLocation } from "types";
import { PubSub } from "utils/pubsub";
import { stringifySearchParams, uuidToBase36 } from "./urlTools";

export enum ObjectType {
    Comment = 'Comment',
    Organization = 'Organization',
    Project = 'Project',
    Routine = 'Routine',
    Run = 'Run',
    Standard = 'Standard',
    Star = 'Star',
    Tag = 'Tag',
    User = 'User',
    View = 'View',
}

/**
 * Gets URL base for object type
 * @param object Object to get base for
 * @returns Search URL base for object type
 */
export const getObjectUrlBase = (object: Omit<OpenObjectProps['object'], 'id'>): string => {
    switch (object.__typename) {
        case ObjectType.Organization:
            return APP_LINKS.Organization;
        case ObjectType.Project:
            return APP_LINKS.Project;
        case ObjectType.Routine:
            return APP_LINKS.Routine;
        case ObjectType.Standard:
            return APP_LINKS.Standard;
        case ObjectType.User:
            return APP_LINKS.Profile;
        case ObjectType.Star:
        case ObjectType.View:
            return getObjectUrlBase(object.to as any);
        case ObjectType.Run:
            return getObjectUrlBase({
                __typename: ObjectType.Routine,
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
export const getObjectSlug = (object: any) => {
    // If object is a star/vote/some other type that links to a main object, use that object's slug
    if (object.to) return getObjectSlug(object.to);
    // If object is a run, navigate to the routine
    if (object.routine) return uuidToBase36(object.routine.id);
    // Otherwise, use either the object's (ADA) handle or its ID
    return object.handle ? object.handle : uuidToBase36(object.id);
}

/**
 * Determines string for object's search params
 * @param object Object being navigated to
 * @returns Stringified search params for object
 */
export const getObjectSearchParams = (object: any) => {
    // If object is a run
    if (object.__typename === ObjectType.Run) return stringifySearchParams({ run: uuidToBase36(object.id) });
    return '';
}

export type OpenObjectProps = {
    object: {
        __typename: string
        handle?: string | null,
        id: string,
        routine?: {
            id: string
        } | null,
        to?: {
            __typename: string,
            handle?: string | null,
            id?: string,
        }
    };
    setLocation: SetLocation;
}
/**
 * Opens any object with an id and __typename
 * @param object Object to open
 * @param setLocation Function to set location in history
 */
export const openObject = (object: OpenObjectProps['object'], setLocation: OpenObjectProps['setLocation']) => {
    // Check if __typename is in objectLinkMap
    if (!ObjectType.hasOwnProperty(object.__typename)) {
        PubSub.get().publishSnack({ message: 'Could not parse object type.', severity: SnackSeverity.Error });
        return;
    }
    // Navigate to object page
    setLocation(`${getObjectUrlBase(object)}/${getObjectSlug(object)}${getObjectSearchParams(object)}`);
}