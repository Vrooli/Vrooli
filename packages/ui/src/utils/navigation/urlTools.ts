import { handleRegex, LINKS, uuidValidate } from "@local/shared";
import { getLastUrlPart, SetLocation } from "route";
import { PubSub } from "utils/pubsub";

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
        PubSub.get().publishSnack({ messageKey: "CouldNotConvertId", severity: "Error", data: { uuid } });
        return "";
    }
};

/**
 * Converts a base 36 string without dashes into a UUID.
 * @param base36 base 36 string without dashes
 * @param showError Whether to show an error snack if the conversion fails
 * @returns v4 UUID
 */
export const base36ToUuid = (base36: string, showError = true): string => {
    try {
        // Convert to base 16. If the ID is less than 32 characters, pad start with 0s. 
        // Then, insert dashes
        const uuid = toBigInt(base36, 36).toString(16).padStart(32, "0").replace(/(.{8})(.{4})(.{4})(.{4})(.{12})/, "$1-$2-$3-$4-$5");
        return uuid === "0" ? "" : uuid;
    } catch (error) {
        if (showError) PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error", data: { base36 } });
        return "";
    }
};

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
export const parseSingleItemUrl = ({
    url,
}: {
    url?: string,
}) => {
    // Initialize the return object
    const returnObject: {
        handleRoot?: string,
        handle?: string,
        idRoot?: string,
        id?: string,
    } = {};
    // Helper for checking if a string is a handle
    const isHandle = (text: string) => {
        if (text.startsWith("@")) text = text.slice(1);
        // Test using handle regex, and make sure it's not a word used for other purposes
        return handleRegex.test(text) && !Object.values(LINKS).includes("/" + text as LINKS) && ["add", "edit", "update"].every(word => !text.includes(word));
    };
    // Get the last 2 parts of the URL
    const lastPart = getLastUrlPart({ url });
    const secondLastPart = getLastUrlPart({ url, offset: 1 });
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
        } else if (uuidValidate(base36ToUuid(secondLastPart, false))) {
            returnObject.idRoot = base36ToUuid(secondLastPart);
            hasRoot = true;
        }
        // Check if the last part is a version handle or version ID
        if (isHandle(lastPart)) {
            if (hasRoot) returnObject.handle = lastPart.replace("@", "");
            else returnObject.handleRoot = lastPart;
        } else if (uuidValidate(base36ToUuid(lastPart, false))) {
            if (hasRoot) returnObject.id = base36ToUuid(lastPart, false);
            else returnObject.idRoot = base36ToUuid(lastPart, false);
        }
    } else {
        // If the URL is for a non-versioned object, check if the last part is a handle or ID
        if (isHandle(lastPart)) {
            returnObject.handle = lastPart.replace("@", "");
        } else if (uuidValidate(base36ToUuid(lastPart, false))) {
            returnObject.id = base36ToUuid(lastPart, false);
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
    onClose: (() => void) | null | undefined,
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
