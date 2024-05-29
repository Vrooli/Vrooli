import { base36ToUuid, handleRegex, LINKS, uuidValidate } from "@local/shared";
import { getLastPathnamePart, SetLocation } from "route";

export type UrlInfo = {
    handleRoot?: string,
    handle?: string,
    idRoot?: string,
    id?: string,
}

/**
 * Finds information in the URL to query for a specific item. 
 * There are multiple ways to specify an item in the URL. 
 * 
 * For non-versioned items, they can be queried by ID, or sometimes by handle. 
 * This is specified as site.com/item/id or site.com/item/handle.
 * 
 * For versioned items, they can be queried by ID, ID of the root item, or handle of the root item.
 * This is specified as site.com/item/rootId, site.com/item/rootId/id, site.com/handle, or site.com/handle/id.
 * 
 * NOTE: This function may sometimes be used for deeper navigation within a single item, 
 * such as site.com/reports/id or site.com/comments/id. In this case, the logic is still the same.
 */
export const parseSingleItemUrl = ({ href, pathname }: { href?: string, pathname?: string }) => {
    // Initialize the return object
    const returnObject: UrlInfo = {};
    // Get the pathname from the href if it's not provided
    if (!pathname && href) {
        try {
            pathname = new URL(href).pathname;
        } catch (error) {
            console.error("Error parsing URL in parseSingleItemUrl", href, error);
        }
    }
    // If no pathname provided, return empty object
    if (!pathname) return returnObject;
    // Helper for checking if a string is a handle
    const isHandle = (text: string) => {
        if (text.startsWith("@")) text = text.slice(1);
        // Test using handle regex, and make sure it's not a word used for other purposes
        return handleRegex.test(text) && !Object.values(LINKS).includes("/" + text as LINKS) && ["add", "edit", "update"].every(word => !text.includes(word));
    };
    // Get the last 2 parts of the URL
    const lastPart = getLastPathnamePart({ pathname });
    const secondLastPart = getLastPathnamePart({ pathname, offset: 1 });
    // Get the list of versioned object names
    const objectsWithVersions = [
        LINKS.Api,
        LINKS.Note,
        LINKS.Project,
        LINKS.Routine,
        LINKS.SmartContract,
        LINKS.Standard,
    ].map(link => link.split("/").pop());
    // Check if any part of the URL contains the name of a versioned object
    const allUrlParts = window.location.pathname.split("/");
    const isVersioned = allUrlParts.some(part => objectsWithVersions.includes(part));
    // If the URL is for a versioned object
    if (isVersioned) {
        let hasRoot = false;
        // Check if the second last part is a root handle or root ID
        if (isHandle(secondLastPart)) {
            returnObject.handleRoot = secondLastPart.replace("@", "");
            hasRoot = true;
        } else if (uuidValidate(base36ToUuid(secondLastPart))) {
            returnObject.idRoot = base36ToUuid(secondLastPart);
            hasRoot = true;
        }
        // Check if the last part is a version handle or version ID
        if (isHandle(lastPart)) {
            if (hasRoot) returnObject.handle = lastPart.replace("@", "");
            else returnObject.handleRoot = lastPart;
        } else if (uuidValidate(base36ToUuid(lastPart))) {
            if (hasRoot) returnObject.id = base36ToUuid(lastPart);
            else returnObject.idRoot = base36ToUuid(lastPart);
        }
    } else {
        // If the URL is for a non-versioned object, check if the last part is a handle or ID
        if (isHandle(lastPart)) {
            returnObject.handle = lastPart.replace("@", "");
        } else if (uuidValidate(base36ToUuid(lastPart))) {
            returnObject.id = base36ToUuid(lastPart);
        }
    }
    // Return the object
    return returnObject;
};

/**
 * If onClose is a function, call it. Otherwise, 
 * try to navigate back if previous url is this site. 
 * Otherwise, navigate to the home page.
 */
export const tryOnClose = (
    onClose: (() => unknown) | null | undefined,
    setLocation: SetLocation,
) => {
    if (typeof onClose === "function") {
        onClose();
        return;
    }
    const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));
    if (hasPreviousPage) window.history.back();
    else setLocation(LINKS.Home);
};
