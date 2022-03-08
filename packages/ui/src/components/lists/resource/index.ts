import { ResourceUsedFor } from 'graphql/generated/globalTypes';
import {
    ConnectWithoutContact as DefaultSocialIcon,
    Download as InstallIcon,
    EventNote as SchedulingIcon,
    Facebook as FacebookIcon,
    Feed as FeedIcon,
    FormatListNumbered as NotesIcon,
    Help as TutorialIcon,
    HowToVote as ProposalIcon,
    Info as ContextIcon,
    Instagram as InstagramIcon,
    Link as RelatedIcon,
    Public as CommunityIcon,
    Reddit as RedditIcon,
    Redeem as DonationIcon,
    School as LearningIcon,
    Science as ResearchingIcon,
    Terminal as DeveloperIcon,
    Twitter as TwitterIcon,
    VideoCameraFront as SocialVideoIcon,
    Web as OfficialWebsiteIcon,
    YouTube as YouTubeIcon,
} from '@mui/icons-material';
import { OverridableComponent } from '@mui/material/OverridableComponent';
import { SvgIconTypeMap } from '@mui/material';

export const ResourceIconMap = {
    [ResourceUsedFor.Community]: CommunityIcon,
    [ResourceUsedFor.Context]: ContextIcon,
    [ResourceUsedFor.Developer]: DeveloperIcon,
    [ResourceUsedFor.Donation]: DonationIcon,
    [ResourceUsedFor.ExternalService]: OfficialWebsiteIcon,
    [ResourceUsedFor.Feed]: FeedIcon,
    [ResourceUsedFor.Install]: InstallIcon,
    [ResourceUsedFor.Learning]: LearningIcon,
    [ResourceUsedFor.Notes]: NotesIcon,
    [ResourceUsedFor.OfficialWebsite]: OfficialWebsiteIcon,
    [ResourceUsedFor.Proposal]: ProposalIcon,
    [ResourceUsedFor.Related]: RelatedIcon,
    [ResourceUsedFor.Researching]: ResearchingIcon,
    [ResourceUsedFor.Scheduling]: SchedulingIcon,
    [ResourceUsedFor.Tutorial]: TutorialIcon,
}

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
}

/**
 * Maps resource type to icon
 * @param usedFor Resource used for type
 * @param link Resource's link, to check if it is a social media link
 * @returns Icon to display
 */
export const getResourceIcon = (usedFor: ResourceUsedFor, link: string): OverridableComponent<SvgIconTypeMap<{}, "svg">> & {muiName: string} => {
    // Social media is a special case, as the icon depends 
    // on the url
    if (usedFor === ResourceUsedFor.Social) {
        const url = new URL(link); // eg. https://www.youtube.com/watch?v=dQw4w9WgXcQ
        const host = url.hostname; // eg. www.youtube.com
        // Remove beginning of hostname (usually "www", but sometimes "m")
        const hostParts = host.split('.').filter(p => !['www', 'm'].includes(p)); // eg. ['youtube', 'com']
        // If no host name found, return default icon
        if (hostParts.length === 0) {
            return ResourceSocialIconMap.default;
        }
        const hostName = hostParts[0].toLowerCase();
        if (hostName in ResourceSocialIconMap) return ResourceSocialIconMap[hostName];
        return ResourceSocialIconMap.default;
    }
    if (usedFor in ResourceIconMap) return ResourceIconMap[usedFor];
    return RelatedIcon;
}

export * from './ResourceCard/ResourceCard';
export * from './ResourceListHorizontal/ResourceListHorizontal';
export * from './ResourceListItem/ResourceListItem';
export * from './ResourceListItemContextMenu/ResourceListItemContextMenu';
export * from './ResourceListVertical/ResourceListVertical';