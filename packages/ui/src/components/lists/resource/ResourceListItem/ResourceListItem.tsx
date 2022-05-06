// Used to display popular/search results of a particular object type
import { IconButton, ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { ResourceListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { adaHandleRegex, MemberRole, ResourceSortBy, ResourceUsedFor, urlRegex, walletAddressRegex } from '@local/shared';
import { useLocation } from 'wouter';
import { getTranslation, LabelledSortOption, labelledSortOptions, openLink, Pubs, ResourceType } from 'utils';
import { Resource } from 'types';
import { getResourceIcon } from '..';
import {
    OpenInNew as OpenLinkIcon
} from '@mui/icons-material';

/**
 * Determines if a resource is a URL, wallet payment address, or an ADA handle
 * @param link String to check
 * @returns ResourceType if type found, or null if not
 */
 const getResourceType = (link: string): ResourceType | null => {
    if (urlRegex.test(link)) return ResourceType.Url;
    if (walletAddressRegex.test(link)) return ResourceType.Wallet;
    if (adaHandleRegex.test(link)) return ResourceType.Handle;
    return null;
}

export function ResourceListItem({
    session,
    index,
    data,
    onClick,
}: ResourceListItemProps) {
    const [, setLocation] = useLocation();
    const canEdit = useMemo<boolean>(() => Boolean(data?.role) && [MemberRole.Admin, MemberRole.Owner].includes(data.role), [data]);
    const { description, title } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            title: getTranslation(data, 'title', languages, true),
        };
    }, [data, session]);

    const Icon = useMemo(() => getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link), [data]);

    const handleClick = useCallback((event: any) => {
        event.stopPropagation();
        // Find the resource type
        const resourceType = getResourceType(data.link);
        // If null, show error
        if (!resourceType) {
            PubSub.publish(Pubs.Snack, { message: 'Unable to open link', severity: 'error' });
            return;
        }
        // If URL, open in new tab
        if (resourceType === ResourceType.Url) openLink(setLocation, data.link);
        // If wallet address, open dialog to copy to clipboard
        else if (resourceType === ResourceType.Wallet) {
            PubSub.publish(Pubs.AlertDialog, {
                message: `Wallet address: ${data.link}`,
                buttons: [
                    {
                        text: 'Copy', onClick: () => {
                            navigator.clipboard.writeText(data.link);
                            PubSub.publish(Pubs.Snack, { message: 'Copied.', severity: 'success' });
                        }
                    },
                    { text: 'Close' }
                ]
            });
        }
        // If handle, open ADA Handle payment site
        else if (resourceType === ResourceType.Handle) openLink(setLocation, `https://handle.me/${data.link}`);
    }, [data]);

    return (
        <Tooltip placement="top" title="Open in new tab">
            <ListItem
                disablePadding
                onClick={handleClick}
                sx={{
                    display: 'flex',
                    background: index % 2 === 0 ? '#c8d6e9' : '#e9e9e9',
                    color: 'black',
                }}
            >
                <ListItemButton component="div" onClick={handleClick}>
                    <IconButton sx={{
                        width: "50px",
                        minWidth: "50px",
                        height: "50px",
                    }}>
                        <Icon />
                    </IconButton>
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        {/* Name/Title */}
                        <ListItemText
                            primary={title}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        {/* Bio/Description */}
                        <ListItemText
                            primary={description}
                            sx={{ ...multiLineEllipsis(2), color: (t) => t.palette.text.secondary }}
                        />
                    </Stack>
                    <OpenLinkIcon />
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const ResourceSortOptions: LabelledSortOption<ResourceSortBy>[] = labelledSortOptions(ResourceSortBy);
export const resourceDefaultSortOption = ResourceSortOptions[1];
export const resourceOptionLabel = (o: Resource, languages: readonly string[]) => getTranslation(o, 'title', languages, true) ?? '';