import { ArticleIcon, DefaultSocialIcon, DonateIcon, DownloadIcon, FacebookIcon, HelpIcon, InfoIcon, InstagramIcon, LearnIcon, LinkIcon, ListNumberIcon, OrganizationIcon, ProposalIcon, RedditIcon, ResearchIcon, ResourceUsedFor, ScheduleIcon, SocialVideoIcon, SvgComponent, TerminalIcon, TwitterIcon, WebsiteIcon, YouTubeIcon } from "@local/shared";

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
    "twitter": TwitterIcon,
    "vimeo": SocialVideoIcon,
    "youtube": YouTubeIcon,
    "reddit": RedditIcon,
};

/**
 * Maps resource type to icon
 * @param usedFor Resource used for type
 * @param link Resource's link, to check if it is a social media link
 * @returns Icon to display
 */
export const getResourceIcon = (usedFor: ResourceUsedFor, link?: string): any => {
    // Social media is a special case, as the icon depends 
    // on the url
    if (usedFor === ResourceUsedFor.Social) {
        if (!link) return ResourceSocialIconMap.default;
        const url = new URL(link); // eg. https://www.youtube.com/watch?v=dQw4w9WgXcQ
        const host = url.hostname; // eg. www.youtube.com
        // Remove beginning of hostname (usually "www", but sometimes "m")
        const hostParts = host.split(".").filter(p => !["www", "m"].includes(p)); // eg. ['youtube', 'com']
        // If no host name found, return default icon
        if (hostParts.length === 0) {
            return ResourceSocialIconMap.default;
        }
        const hostName = hostParts[0].toLowerCase();
        if (hostName in ResourceSocialIconMap) return ResourceSocialIconMap[hostName];
        return ResourceSocialIconMap.default;
    }
    if (usedFor in ResourceIconMap) return ResourceIconMap[usedFor];
    return LinkIcon;
};
