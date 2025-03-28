import { LINKS, ModelType, ResourceUsedFor } from "@local/shared";
import { Avatar, Palette } from "@mui/material";
import { Icon, IconInfo } from "../../icons/Icons.js";
import { getCookiePartialData } from "../../utils/localStorage.js";
import { parseSingleItemUrl } from "../../utils/navigation/urlTools.js";
import { extractImageUrl } from "./imageTools.js";
import { getDisplay, placeholderColor } from "./listTools.js";

export const ResourceIconMap: { [key in ResourceUsedFor]?: IconInfo } = {
    [ResourceUsedFor.Community]: { name: "Team", type: "Common" },
    [ResourceUsedFor.Context]: { name: "Info", type: "Common" },
    [ResourceUsedFor.Developer]: { name: "Terminal", type: "Common" },
    [ResourceUsedFor.Donation]: { name: "Donate", type: "Common" },
    [ResourceUsedFor.ExternalService]: { name: "Website", type: "Common" },
    [ResourceUsedFor.Feed]: { name: "Article", type: "Common" },
    [ResourceUsedFor.Install]: { name: "Download", type: "Common" },
    [ResourceUsedFor.Learning]: { name: "Learn", type: "Common" },
    [ResourceUsedFor.Notes]: { name: "ListNumber", type: "Text" },
    [ResourceUsedFor.OfficialWebsite]: { name: "Website", type: "Common" },
    [ResourceUsedFor.Proposal]: { name: "Proposal", type: "Common" },
    [ResourceUsedFor.Related]: { name: "Link", type: "Common" },
    [ResourceUsedFor.Researching]: { name: "Research", type: "Common" },
    [ResourceUsedFor.Scheduling]: { name: "Schedule", type: "Common" },
    [ResourceUsedFor.Tutorial]: { name: "Help", type: "Common" },
};

export const ResourceSocialIconMap: { [key: string]: IconInfo } = {
    "default": { name: "DefaultSocial", type: "Service" },
    "facebook": { name: "Facebook", type: "Service" },
    "instagram": { name: "Instagram", type: "Service" },
    "tiktok": { name: "SocialVideo", type: "Common" },
    "odysee": { name: "SocialVideo", type: "Common" },
    "x": { name: "X", type: "Service" },
    "vimeo": { name: "SocialVideo", type: "Common" },
    "youtube": { name: "YouTube", type: "Service" },
    "reddit": { name: "Reddit", type: "Service" },
};

const LinkIconMap: { [key in LINKS]?: IconInfo } = {
    [LINKS.About]: { name: "Info", type: "Common" },
    [LINKS.Api]: { name: "Api", type: "Common" },
    [LINKS.Awards]: { name: "Award", type: "Common" },
    [LINKS.BookmarkList]: { name: "BookmarkFilled", type: "Common" },
    [LINKS.Calendar]: { name: "Month", type: "Common" },
    [LINKS.Chat]: { name: "Comment", type: "Common" },
    [LINKS.DataConverter]: { name: "Terminal", type: "Common" },
    [LINKS.DataStructure]: { name: "Object", type: "Common" },
    [LINKS.Comment]: { name: "Comment", type: "Common" },
    [LINKS.Create]: { name: "Create", type: "Common" },
    [LINKS.History]: { name: "History", type: "Common" },
    [LINKS.Inbox]: { name: "NotificationsAll", type: "Common" },
    [LINKS.MyStuff]: { name: "Grid", type: "Common" },
    [LINKS.Note]: { name: "Note", type: "Common" },
    [LINKS.Pro]: { name: "Premium", type: "Common" },
    [LINKS.Profile]: { name: "User", type: "Common" },
    [LINKS.Project]: { name: "Project", type: "Common" },
    [LINKS.Prompt]: { name: "Article", type: "Common" },
    [LINKS.Question]: { name: "Help", type: "Common" },
    [LINKS.Reminder]: { name: "Reminder", type: "Common" },
    [LINKS.Report]: { name: "Report", type: "Common" },
    [LINKS.RoutineMultiStep]: { name: "Routine", type: "Routine" },
    [LINKS.RoutineSingleStep]: { name: "Routine", type: "Routine" },
    [LINKS.Search]: { name: "Search", type: "Common" },
    [LINKS.Settings]: { name: "Settings", type: "Common" },
    [LINKS.SmartContract]: { name: "SmartContract", type: "Common" },
    [LINKS.Stats]: { name: "Stats", type: "Common" },
    [LINKS.Team]: { name: "Team", type: "Common" },
};

function getRoute(pathname: string): LINKS | undefined {
    const pathSegments = pathname.split("/").filter(segment => segment !== "");
    for (const key of Object.keys(LinkIconMap)) {
        const keySegments = key.split("/").filter(segment => segment !== "");
        // If the number of segments don't match, continue to the next key.
        if (pathSegments.length < keySegments.length) {
            continue;
        }
        const doesMatch = keySegments.every((segment, index) => segment === pathSegments[index]);
        if (doesMatch) {
            return key as LINKS;
        }
    }
    return undefined;
}

/**
 * Maps resource type to icon
 * @param usedFor Resource used for type
 * @param link Resource's link, to check if it is a social media link
 * @returns Icon to display
 */
export function getResourceIcon(usedFor: ResourceUsedFor, link?: string, palette?: Palette): JSX.Element {
    // Determine default icon
    const defaultIcon = usedFor === ResourceUsedFor.Social ? ResourceSocialIconMap.default : (ResourceIconMap[usedFor] ?? LinkIconMap[usedFor]);
    // Create URL object from link safely
    let url: URL | null = null;
    try {
        if (link) {
            url = new URL(link);
        }
    } catch (err) {
        // Invalid URL, return default icon
        console.error(`Invalid URL passed to getResourceIcon: ${link}`, err);
        return defaultIcon;
    }
    if (!url) {
        return defaultIcon;
    }
    // Find host name
    const host = url.hostname; // eg. www.youtube.com
    // Remove beginning of hostname (usually "www", but sometimes "m")
    const hostParts = host.split(".").filter(p => !["www", "m"].includes(p)); // eg. ['youtube', 'com']
    // If no host name found, return default icon
    if (hostParts.length === 0) {
        return defaultIcon;
    }
    const hostName = hostParts[0].toLowerCase();
    // ResourceUsedFor.Context is a special case, as we can replace it with a Vrooli route's icon
    if (usedFor === ResourceUsedFor.Context && hostName === (process.env.PROD ? "vrooli.com" : "localhost")) {
        // Get route info
        const route = getRoute(url.pathname);
        const routeKey = Object.keys(LINKS).find(key => LINKS[key as LINKS] === route);
        // Check if it corresponds to a cached item
        const urlParams = parseSingleItemUrl({ href: link });
        const cachedItem = getCookiePartialData({ __typename: routeKey as ModelType, ...urlParams }) as { __typename: ModelType, isBot?: boolean, profileImage?: string, updated_at?: string };
        const profileIconInfo: IconInfo = cachedItem.isBot ?
            { name: "Bot", type: "Common" }
            : routeKey === "User"
                ? { name: "User", type: "Common" }
                : { name: "Team", type: "Common" };
        // If cached item has a profileImage, return it as an Avatar
        if (cachedItem.profileImage) {
            return (<Avatar
                alt={`${getDisplay(cachedItem).title}'s profile picture`}
                src={extractImageUrl(cachedItem.profileImage, cachedItem.updated_at, 50)}
                sx={{
                    backgroundColor: placeholderColor()[0],
                    width: "24px",
                    height: "24px",
                    pointerEvents: "none",
                    ...(cachedItem.isBot ? { borderRadius: "4px" } : {}),
                }}
            >
                <Icon
                    decorative
                    fill={palette?.background?.textPrimary ?? "white"}
                    info={profileIconInfo}
                    size={24}
                />
            </Avatar>);
        }
        // If cached item is a bot, return bot icon
        if ((cachedItem as { isBot?: boolean }).isBot) {
            return <Icon
                decorative
                fill={palette?.background?.textPrimary ?? "white"}
                info={profileIconInfo}
                size={24}
            />;
        }
        // Otherwise, return route icon or default icon
        return LinkIconMap[route as LINKS] ?? defaultIcon;
    }
    // ResourceUsedFor.Social is a special case, as the icon depends on the url
    if (usedFor === ResourceUsedFor.Social) {
        const iconInfo = ResourceSocialIconMap[hostName] ?? ResourceSocialIconMap.default;
        return <Icon
            decorative
            fill={palette?.background?.textPrimary ?? "white"}
            info={iconInfo}
            size={24}
        />;
    }
    return <Icon
        decorative
        fill={palette?.background?.textPrimary ?? "white"}
        info={defaultIcon}
        size={24}
    />;
}
