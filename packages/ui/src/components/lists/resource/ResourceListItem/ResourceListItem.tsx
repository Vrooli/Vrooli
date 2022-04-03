// Used to display popular/search results of a particular object type
import { IconButton, ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { ResourceListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { MemberRole, ResourceSortBy, ResourceUsedFor } from '@local/shared';
import { useLocation } from 'wouter';
import { getTranslation, LabelledSortOption, labelledSortOptions } from 'utils';
import { Resource } from 'types';
import { getResourceIcon } from '..';
import {
    OpenInNew as OpenLinkIcon
} from '@mui/icons-material';

export function ResourceListItem({
    session,
    index,
    data,
    onClick,
}: ResourceListItemProps) {
    const [, setLocation] = useLocation();
    const canEdit: boolean = useMemo(() => [MemberRole.Admin, MemberRole.Owner].includes(data?.role ?? ''), [data]);
    const { description, title } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            title: getTranslation(data, 'title', languages, true),
        };
    }, [data, session]);

    const Icon = useMemo(() => getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link), [data]);

    const handleClick = useCallback((e: any) => {
        e.stopPropagation();
        window.open(data.link, '_blank', 'noopener,noreferrer')
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