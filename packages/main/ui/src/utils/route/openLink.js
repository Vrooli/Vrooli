export const openLink = (setLocation, link) => {
    if ((link.includes("http:") || link.includes("https:")) && !link.startsWith(window.location.origin)) {
        window.open(link, "_blank", "noopener,noreferrer");
    }
    else {
        setLocation(link);
    }
};
//# sourceMappingURL=openLink.js.map