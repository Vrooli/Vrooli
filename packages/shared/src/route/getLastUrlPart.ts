/**
 * @returns last part of the url path
 * @param offset Number of parts to offset from the end of the path (default: 0)
 * @returns part of the url path that is <offset> parts from the end, or empty string if no path
 */
export const getLastUrlPart = (offset = 0): string => {
    let parts = window.location.pathname.split("/");
    // Remove any empty strings
    parts = parts.filter(part => part !== "");
    // Check to make sure there is a part at the offset
    if (parts.length < offset + 1) return "";
    return parts[parts.length - offset - 1];
};
