/**
 * @returns last part of a pathname
 * @param pathname Pathname to get the last part of (default: window.location.pathname)
 * @param offset Number of parts to offset from the end of the path (default: 0)
 * @returns part of the pathname that is <offset> parts from the end, or empty string if no path
 */
export const getLastPathnamePart = ({
    pathname,
    offset = 0,
}: {
    pathname?: string,
    offset?: number,
}): string => {
    let parts = (pathname ?? window.location.pathname).split("/");
    // Remove any empty strings
    parts = parts.filter(part => part !== "");
    // Check to make sure there is a part at the offset
    if (parts.length < offset + 1) return "";
    return parts[parts.length - offset - 1];
};
