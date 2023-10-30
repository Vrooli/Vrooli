import { GqlModelType, LINKS, ResourceUsedFor } from "@local/shared";
import { Avatar, Palette } from "@mui/material";
import { ApiIcon, ArticleIcon, AwardIcon, BookmarkFilledIcon, BotIcon, CommentIcon, CreateIcon, DefaultSocialIcon, DonateIcon, DownloadIcon, FacebookIcon, GridIcon, HelpIcon, HistoryIcon, InfoIcon, InstagramIcon, LearnIcon, LinkIcon, ListNumberIcon, MonthIcon, NoteIcon, NotificationsAllIcon, OrganizationIcon, PremiumIcon, ProjectIcon, ProposalIcon, RedditIcon, ReminderIcon, ReportIcon, ResearchIcon, RoutineIcon, ScheduleIcon, SearchIcon, SettingsIcon, SmartContractIcon, SocialVideoIcon, StandardIcon, StatsIcon, TerminalIcon, UserIcon, WebsiteIcon, XIcon, YouTubeIcon } from "icons";
import { SvgComponent } from "types";
import { getCookiePartialData } from "utils/cookies";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { extractImageUrl } from "./imageTools";
import { getDisplay, placeholderColor } from "./listTools";

export const ResourceIconMap: { [key in ResourceUsedFor]?: SvgComponent } = {
    [ResourceUsedFor.Community]: OrganizationIcon,
    [ResourceUsedFor.Context]: InfoIcon,
    [ResourceUsedFor.Developer]: TerminalIcon,
    [ResourceUsedFor.Donation]: DonateIcon,
    [ResourceUsedFor.ExternalService]: WebsiteIcon,
    [ResourceUsedFor.Feed]: ArticleIcon,
    [ResourceUsedFor.Install]: DownloadIcon,
    [ResourceUsedFor.Learning]: LearnIcon,
    [ResourceUsedFor.Notes]: ListNumberIcon,
    [ResourceUsedFor.OfficialWebsite]: WebsiteIcon,
    [ResourceUsedFor.Proposal]: ProposalIcon,
    [ResourceUsedFor.Related]: LinkIcon,
    [ResourceUsedFor.Researching]: ResearchIcon,
    [ResourceUsedFor.Scheduling]: ScheduleIcon,
    [ResourceUsedFor.Tutorial]: HelpIcon,
};

export const ResourceSocialIconMap: { [key: string]: SvgComponent } = {
    "default": DefaultSocialIcon,
    "facebook": FacebookIcon,
    "instagram": InstagramIcon,
    "tiktok": SocialVideoIcon,
    "odysee": SocialVideoIcon,
    "x": XIcon,
    "vimeo": SocialVideoIcon,
    "youtube": YouTubeIcon,
    "reddit": RedditIcon,
};

const LinkIconMap: { [key in LINKS]?: SvgComponent } = {
    [LINKS.About]: InfoIcon,
    [LINKS.Api]: ApiIcon,
    [LINKS.Awards]: AwardIcon,
    [LINKS.BookmarkList]: BookmarkFilledIcon,
    [LINKS.Calendar]: MonthIcon,
    [LINKS.Chat]: CommentIcon,
    [LINKS.Comment]: CommentIcon,
    [LINKS.Create]: CreateIcon,
    [LINKS.History]: HistoryIcon,
    [LINKS.Inbox]: NotificationsAllIcon,
    [LINKS.MyStuff]: GridIcon,
    [LINKS.Note]: NoteIcon,
    [LINKS.Organization]: OrganizationIcon,
    [LINKS.Premium]: PremiumIcon,
    [LINKS.Profile]: UserIcon,
    [LINKS.Project]: ProjectIcon,
    [LINKS.Question]: HelpIcon,
    [LINKS.Reminder]: ReminderIcon,
    [LINKS.Report]: ReportIcon,
    [LINKS.Routine]: RoutineIcon,
    [LINKS.Search]: SearchIcon,
    [LINKS.Settings]: SettingsIcon,
    [LINKS.SmartContract]: SmartContractIcon,
    [LINKS.Standard]: StandardIcon,
    [LINKS.Stats]: StatsIcon,
    [LINKS.User]: UserIcon,
};

const getRoute = (pathname: string): LINKS | undefined => {
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
};

/**
 * Maps resource type to icon
 * @param usedFor Resource used for type
 * @param link Resource's link, to check if it is a social media link
 * @returns Icon to display
 */
export const getResourceIcon = (usedFor: ResourceUsedFor, link?: string, palette?: Palette): SvgComponent | JSX.Element => {
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
    if (usedFor === ResourceUsedFor.Context && hostName === (import.meta.env.PROD ? "vrooli.com" : "localhost")) {
        // Get route info
        const route = getRoute(url.pathname);
        const routeKey = Object.keys(LINKS).find(key => LINKS[key as LINKS] === route);
        // Check if it corresponds to a cached item
        const urlParams = parseSingleItemUrl({ href: link });
        const cachedItem = getCookiePartialData({ __typename: routeKey as GqlModelType, ...urlParams }) as { __typename: GqlModelType, isBot?: boolean, profileImage?: string, updated_at?: string };
        console.log("urlParams", urlParams);
        console.log("cachedItem", cachedItem);
        // If cached item has a profileImage, return it as an Avatar
        if (cachedItem.profileImage) {
            return (<Avatar
                src={extractImageUrl(cachedItem.profileImage, cachedItem.updated_at, 50)}
                alt={`${getDisplay(cachedItem).title}'s profile picture`}
                sx={{
                    backgroundColor: placeholderColor()[0],
                    width: "24px",
                    height: "24px",
                    pointerEvents: "none",
                    ...(cachedItem.isBot ? { borderRadius: "4px" } : {}),
                }}
            >
                {cachedItem.isBot ?
                    <BotIcon width="75%" height="75%" fill={palette?.background?.textPrimary ?? "white"} /> :
                    routeKey === "User" ? <UserIcon width="75%" height="75%" fill={palette?.background.textPrimary ?? "white"} /> :
                        <OrganizationIcon width="75%" height="75%" fill={palette?.background?.textPrimary ?? "white"} />}
            </Avatar>);
        }
        // If cached item is a bot, return bot icon
        if ((cachedItem as { isBot?: boolean }).isBot) {
            return BotIcon;
        }
        // Otherwise, return route icon or default icon
        return LinkIconMap[route as LINKS] ?? defaultIcon;
    }
    // ResourceUsedFor.Social is a special case, as the icon depends on the url
    if (usedFor === ResourceUsedFor.Social) {
        return ResourceSocialIconMap[hostName] ?? ResourceSocialIconMap.defaul;
    }
    return defaultIcon;
};
