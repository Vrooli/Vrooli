import { LINKS, ModelType, ResourceUsedFor } from "@local/shared";
import { Avatar, Palette } from "@mui/material";
import { Icon, IconFavicon, IconInfo } from "../../icons/Icons.js";
import { getCookiePartialData } from "../../utils/localStorage.js";
import { parseSingleItemUrl } from "../../utils/navigation/urlTools.js";
import { extractImageUrl } from "./imageTools.js";
import { getDisplay, placeholderColor } from "./listTools.js";

// Constants
const ICON_SIZE = 24;
const AVATAR_SIZE = 50;

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

type GetResourceIconProps = {
    /** Override the fill color of the icon */
    fill?: string;
    /** Set the link for the resource */
    link?: string;
    /** Used to get the proper fill color for the icon, if not being provided directly */
    palette?: Palette;
    /** The type of resource */
    usedFor: ResourceUsedFor;
}

/**
 * Maps resource type to icon
 * 
 * @returns Icon to display
 */
export function getResourceIcon({ fill, link, palette, usedFor }: GetResourceIconProps): JSX.Element {
    const fillColor = fill ?? palette?.background?.textPrimary ?? "white";

    // Determine default icon
    const defaultIcon: IconInfo = usedFor === ResourceUsedFor.Social
        ? { name: "DefaultSocial", type: "Service" }
        : (ResourceIconMap[usedFor] ?? LinkIconMap[usedFor] ?? { name: "Website", type: "Common" });

    // If no link provided, return default icon
    if (!link) {
        return <Icon
            decorative
            fill={fillColor}
            info={defaultIcon}
            size={ICON_SIZE}
        />;
    }

    // ResourceUsedFor.Context is a special case, as we can replace it with a Vrooli route's icon
    try {
        const url = new URL(link);
        const hostName = url.hostname.split(":")[0].toLowerCase(); // Split by colon to handle ports

        if (usedFor === ResourceUsedFor.Context && (
            process.env.PROD ? hostName === "vrooli.com" : hostName === "localhost"
        )) {
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
                    src={extractImageUrl(cachedItem.profileImage, cachedItem.updated_at, AVATAR_SIZE)}
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
                        fill={fillColor}
                        info={profileIconInfo}
                        size={ICON_SIZE}
                    />
                </Avatar>);
            }

            // If cached item is a bot, return bot icon
            if (cachedItem.isBot) {
                return <Icon
                    decorative
                    fill={fillColor}
                    info={profileIconInfo}
                    size={ICON_SIZE}
                />;
            }

            // Otherwise, return route icon or default icon
            const routeIcon = LinkIconMap[route as LINKS];
            return <Icon
                decorative
                fill={fillColor}
                info={routeIcon ?? defaultIcon}
                size={ICON_SIZE}
            />;
        }

        // For all other cases, use IconFavicon with the appropriate fallback
        return <IconFavicon
            href={link}
            size={ICON_SIZE}
            fill={fillColor}
            decorative
            fallbackIcon={defaultIcon}
        />;
    } catch (err) {
        // Invalid URL, return default icon
        console.error(`Invalid URL passed to getResourceIcon: ${link}`, err);
        return <Icon
            decorative
            fill={fillColor}
            info={defaultIcon}
            size={ICON_SIZE}
        />;
    }
}
