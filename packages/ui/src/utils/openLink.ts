// Automatically determines whether to open a link in a new tab, or push to history.
// This is useful for a list of links, where some may lead to the landing page, while
import { NavigateFunction } from "react-router-dom";

// others lead to the main application
export const openLink = (navigate: NavigateFunction, link: string) => {
    // If link is external, open new tab
    if (link.includes('http:') || link.includes('https')) {
        window.open(link, '_blank', 'noopener,noreferrer');
    } 
    // Otherwise, push to history
    else {
        navigate(link);
    }
};