import { ParseSearchParamsResult, stringifySearchParams } from "./searchParams";
import { SetLocation } from "./types";

/**
 * Opens link using routing or a new tab, depending on the link
 * @param setLocation Function to set location in the router
 * @param link Link to open
 * @param queryParams Query parameters to append to the link
 */
export const openLink = (setLocation: SetLocation, link: string, queryParams?: ParseSearchParamsResult) => {
    const linkWithParams = `${link}${stringifySearchParams(queryParams || {})}`;
    // If link is external, open new tab
    if ((link.includes("http:") || link.includes("https:")) && !link.startsWith(window.location.origin)) {
        window.open(linkWithParams, "_blank", "noopener,noreferrer");
    }
    // Otherwise, push to history
    else {
        setLocation(linkWithParams);
    }
};
