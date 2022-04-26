/**
 * Navigate to various objects and object search pages
 */

import { APP_LINKS } from "@local/shared";
import { SetLocation } from "types";
import { Pubs } from "utils/consts";

export enum ObjectType {
    Organization = 'Organization',
    Project = 'Project',
    Routine = 'Routine',
    Standard = 'Standard',
    User = 'User',
}

const linkMap: { [key in ObjectType]: [string, string] } = {
    [ObjectType.Organization]: [APP_LINKS.SearchOrganizations, APP_LINKS.Organization],
    [ObjectType.Project]: [APP_LINKS.SearchProjects, APP_LINKS.Project],
    [ObjectType.Routine]: [APP_LINKS.SearchRoutines, APP_LINKS.Run],
    [ObjectType.Standard]: [APP_LINKS.SearchStandards, APP_LINKS.Standard],
    [ObjectType.User]: [APP_LINKS.SearchUsers, APP_LINKS.Profile],
}

/**
 * Opens search page for given object type
 * @param objectType Object type
 * @param setLocation Function to set location in history
 */
export const openSearchPage = (objectType: ObjectType, setLocation: SetLocation) => {
    const linkBases = linkMap[objectType];
    setLocation(linkBases[0]);
}

/**
 * Opens any object with an id and __typename
 * @param object Object to open
 * @param setLocation Function to set location in history
 */
export const openObject = (object: { id: string, __typename: string }, setLocation: SetLocation) => {
    // Check if __typename is in linkMap
    if (!linkMap.hasOwnProperty(object.__typename)) {
        PubSub.publish(Pubs.Snack, { message: 'Could not parse object type.', severity: 'error' });
        return; 
    }
    // Navigate to object page
    const linkBases = linkMap[object.__typename];
    setLocation(`${linkBases[1]}/${object.id}`);
}