import { ResourceUsedFor } from "@local/consts";
import { ArticleIcon, DefaultSocialIcon, DonateIcon, DownloadIcon, FacebookIcon, HelpIcon, InfoIcon, InstagramIcon, LearnIcon, LinkIcon, ListNumberIcon, OrganizationIcon, ProposalIcon, RedditIcon, ResearchIcon, ScheduleIcon, SocialVideoIcon, TerminalIcon, TwitterIcon, WebsiteIcon, YouTubeIcon } from "@local/icons";
export const ResourceIconMap = {
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
export const ResourceSocialIconMap = {
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
export const getResourceIcon = (usedFor, link) => {
    if (usedFor === ResourceUsedFor.Social) {
        if (!link)
            return ResourceSocialIconMap.default;
        const url = new URL(link);
        const host = url.hostname;
        const hostParts = host.split(".").filter(p => !["www", "m"].includes(p));
        if (hostParts.length === 0) {
            return ResourceSocialIconMap.default;
        }
        const hostName = hostParts[0].toLowerCase();
        if (hostName in ResourceSocialIconMap)
            return ResourceSocialIconMap[hostName];
        return ResourceSocialIconMap.default;
    }
    if (usedFor in ResourceIconMap)
        return ResourceIconMap[usedFor];
    return LinkIcon;
};
//# sourceMappingURL=getResourceIcon.js.map