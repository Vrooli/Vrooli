import { ParseSearchParamsResult, stringifySearchParams } from "@local/shared";
import { SetLocation } from "./types";

/**
 * Opens link using routing or a new tab, depending on the link
 * @param setLocation Function to set location in the router
 * @param link Link to open
 * @param searchParams Query parameters to append to the link
 */
export function openLink(setLocation: SetLocation, link: string, searchParams?: ParseSearchParamsResult) {
    // If link is external, open new tab
    if ((link.includes("http:") || link.includes("https:")) && !link.startsWith(window.location.origin)) {
        const linkWithParams = `${link}${stringifySearchParams(searchParams || {})}`;
        window.open(linkWithParams, "_blank", "noopener,noreferrer");
    }
    // Otherwise, push to history
    else {
        setLocation(link, { searchParams });
    }
};
