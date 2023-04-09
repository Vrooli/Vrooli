import { SetLocation } from "@shared/route";

/**
 * Opens link using routing or a new tab, depending on the link
 * @param setLocation Function to set location in the router
 * @param link Link to open
 */
export const openLink = (setLocation: SetLocation, link: string) => {
    // If link is external, open new tab
    if ((link.includes('http:') || link.includes('https:')) && !link.startsWith(window.location.origin)) {
        window.open(link, '_blank', 'noopener,noreferrer');
    } 
    // Otherwise, push to history
    else {
        setLocation(link);
    }
};