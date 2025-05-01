import { handleRegex, LINKS, validatePublicId } from "@local/shared";
import { getLastPathnamePart } from "../../route/getLastPathnamePart.js";
import { SetLocation } from "../../route/types.js";

export type UrlInfo = {
    handleRoot?: string,
    handle?: string,
    idRoot?: string,
    id?: string,
    versionLabel?: string,
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
export function parseSingleItemUrl({ href, pathname }: { href?: string, pathname?: string }) {
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
    function isHandle(text: string) {
        if (text.startsWith("@")) text = text.slice(1);
        // Test using handle regex, and make sure it's not a word used for other purposes
        return handleRegex.test(text) && !Object.values(LINKS).includes("/" + text as LINKS) && ["add", "edit", "update"].every(word => !text.includes(word));
    }
    // Get the last 2 parts of the URL
    const lastPart = getLastPathnamePart({ pathname });
    const secondLastPart = getLastPathnamePart({ pathname, offset: 1 });
    // Get the list of versioned object names
    const objectsWithVersions = [
        LINKS.Api,
        LINKS.DataConverter,
        LINKS.DataStructure,
        LINKS.Note,
        LINKS.Project,
        LINKS.Prompt,
        LINKS.RoutineMultiStep,
        LINKS.RoutineSingleStep,
        LINKS.SmartContract,
    ].map(link => link.split("/").pop());
    // Check if any part of the URL contains the name of a versioned object
    const allUrlParts = pathname.split("/");
    const isVersioned = allUrlParts.some(part => objectsWithVersions.includes(part));
    // If the URL is for a versioned object, parse root and version parts
    if (isVersioned) {
        let hasRoot = false;
        // Check if the second last part is a root handle or root public ID
        if (isHandle(secondLastPart)) {
            returnObject.handleRoot = secondLastPart.replace("@", "");
            hasRoot = true;
        } else if (validatePublicId(secondLastPart)) {
            returnObject.idRoot = secondLastPart;
            hasRoot = true;
        }
        // Check if the last part is a version handle, public ID, or version label
        if (isHandle(lastPart)) {
            if (hasRoot) {
                // version handle
                returnObject.handle = lastPart.replace("@", "");
            } else {
                // root handle only
                returnObject.handleRoot = lastPart.replace("@", "");
            }
        } else if (validatePublicId(lastPart)) {
            if (hasRoot) {
                // version public ID
                returnObject.id = lastPart;
            } else {
                // root public ID (single segment)
                returnObject.idRoot = lastPart;
            }
        } else if (hasRoot) {
            // Treat remaining part as version label
            returnObject.versionLabel = lastPart;
        }
    } else {
        // If the URL is for a non-versioned object, check if the last part is a handle or public ID
        if (isHandle(lastPart)) {
            returnObject.handle = lastPart.replace("@", "");
        } else if (validatePublicId(lastPart)) {
            returnObject.id = lastPart;
        }
    }
    // Return the object
    return returnObject;
}

/**
 * If onClose is a function, call it. Otherwise, 
 * try to navigate back if previous url is this site. 
 * Otherwise, navigate to the home page.
 */
export function tryOnClose(
    onClose: (() => unknown) | null | undefined,
    setLocation: SetLocation,
) {
    if (typeof onClose === "function") {
        onClose();
        return;
    }
    const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));
    if (hasPreviousPage) window.history.back();
    else setLocation(LINKS.Home);
}
