// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Box, Stack, Tooltip, Typography } from '@mui/material';
import { ResourceCard, ResourceListItemContextMenu } from 'components';
import { ResourceListHorizontalProps } from '../types';
import { containerShadow } from 'styles';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { Resource } from 'types';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';
import {
    Add as AddIcon,
    ConnectWithoutContact as DefaultSocialIcon,
    Download as InstallIcon,
    EventNote as SchedulingIcon,
    Facebook as FacebookIcon,
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
    Terminal as DeveloperIcon,
    Twitter as TwitterIcon,
    VideoCameraFront as SocialVideoIcon,
    Web as OfficialWebsiteIcon,
    YouTube as YouTubeIcon,
} from '@mui/icons-material';
import { cardRoot } from 'components/cards/styles';

const IconMap = {
    [ResourceUsedFor.Community]: CommunityIcon,
    [ResourceUsedFor.Context]: ContextIcon,
    [ResourceUsedFor.Developer]: DeveloperIcon,
    [ResourceUsedFor.Donation]: DonationIcon,
    [ResourceUsedFor.ExternalService]: OfficialWebsiteIcon,
    [ResourceUsedFor.Install]: InstallIcon,
    [ResourceUsedFor.Learning]: LearningIcon,
    [ResourceUsedFor.Notes]: NotesIcon,
    [ResourceUsedFor.OfficialWebsite]: OfficialWebsiteIcon,
    [ResourceUsedFor.Proposal]: ProposalIcon,
    [ResourceUsedFor.Related]: RelatedIcon,
    [ResourceUsedFor.Scheduling]: SchedulingIcon,
    [ResourceUsedFor.Tutorial]: TutorialIcon,
}

const SocialIconMap = {
    "facebook": FacebookIcon,
    "instagram": InstagramIcon,
    "tiktok": SocialVideoIcon,
    "odysee": SocialVideoIcon,
    "twitter": TwitterIcon,
    "vimeo": SocialVideoIcon,
    "youtube": YouTubeIcon,
    "reddit": RedditIcon,
}

export const ResourceListHorizontal = ({
    title = '📌 Resources',
    canEdit = true,
    handleUpdate,
    resources,
}: ResourceListHorizontalProps) => {

    // Determine icon to display based on resource type
    const getIcon = useCallback((usedFor: ResourceUsedFor, link: string) => {
        // Social media is a special case, as the icon depends 
        // on the url
        if (usedFor === ResourceUsedFor.Social) {
            const url = new URL(link); // eg. https://www.youtube.com/watch?v=dQw4w9WgXcQ
            const host = url.hostname; // eg. www.youtube.com
            // Remove beginning of hostname (usually "www", but sometimes "m")
            const hostParts = host.split('.').filter(p => !['www', 'm'].includes(p)); // eg. ['youtube', 'com']
            // If no host name found, return default icon
            if (hostParts.length === 0) {
                return DefaultSocialIcon;
            }
            const hostName = hostParts[0].toLowerCase();
            if (SocialIconMap[hostName]) return SocialIconMap[hostName];
            return DefaultSocialIcon;
        }
        return IconMap[usedFor];
    }, [])

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selected, setSelected] = useState<any | null>(null);
    const contextId = useMemo(() => `resource-context-menu-${selected?.link}`, [selected]);
    const openContext = useCallback((ev: MouseEvent<HTMLButtonElement>, data: any) => {
        console.log('setting context anchor', ev.currentTarget, data);
        setContextAnchor(ev.currentTarget);
        setSelected(data);
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelected(null);
    }, []);

    return (
        <Box>
            <ResourceListItemContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                resource={selected}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={() => { }}
                onEdit={() => { }}
                onMove={() => { }}
            />
            <Typography component="h2" variant="h5" textAlign="left">{title}</Typography>
            <Box
                sx={{
                    ...containerShadow,
                    borderRadius: '16px',
                    background: (t) => t.palette.background.default,
                    border: (t) => `1px ${t.palette.text.primary}`,
                }}
            >
                <Stack direction="row" spacing={2} p={2} sx={{ 
                    overflowX: 'scroll',
                    "&::-webkit-scrollbar": {
                        width: 5,
                    },
                    "&::-webkit-scrollbar-track": {
                        backgroundColor: 'transparent',
                    },
                    "&::-webkit-scrollbar-thumb": {
                        borderRadius: '100px',
                        backgroundColor: "#409590",
                    },
                }}>
                    {/* Resources */}
                    {resources.map((c: Resource, index) => (
                        <ResourceCard
                            key={`resource-card-${index}`}
                            data={c}
                            Icon={getIcon(c.usedFor ?? ResourceUsedFor.Related, c.link)}
                            onRightClick={openContext}
                            aria-owns={Boolean(selected) ? contextId : undefined}
                        />
                    ))}
                    {/* Add resource button */}
                    { canEdit ? <Tooltip placement="top" title="Add resource">
                        <Box
                            onClick={() => { }}
                            aria-label="Add resource"
                            sx={{
                                ...cardRoot,
                                background: "#cad2e0",
                                width: '120px',
                                minWidth: '120px',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                            }}
                        >
                            <AddIcon color="primary" sx={{ width: '50px', height: '50px' }} />
                        </Box>
                    </Tooltip> : null }
                </Stack>
            </Box>
        </Box>
    )
}