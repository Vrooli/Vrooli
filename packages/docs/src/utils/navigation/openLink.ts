import { SetLocation } from "types";

/**
 * Automatically determines whether to open a link in a new tab, or push to history.
 * This is useful for a list of links, where some may lead to the landing page, while
 * others lead to the main application
 */
export const openLink = (setLocation: SetLocation, link: string) => {
    // If link is external, open new tab
    if (link.includes('http:') || link.includes('https')) {
        window.open(link, '_blank', 'noopener,noreferrer');
    } 
    // Otherwise, push to history
    else {
        setLocation(link);
    }
};