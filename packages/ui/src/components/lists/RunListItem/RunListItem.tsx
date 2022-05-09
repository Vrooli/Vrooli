// Used to display popular/search results of a particular object type
import { Box, ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { RunListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, MemberRole, RunSortBy, StarFor } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton, TagList } from '..';
import { getTranslation, LabelledSortOption, labelledSortOptions } from 'utils';
import { ListRoutine } from 'types';
import { Apartment as ApartmentIcon } from '@mui/icons-material';

// Color options for profile picture
// [background color, silhouette color]
const colorOptions: [string, string][] = [
    ["#197e2c", "#b5ffc4"],
    ["#b578b6", "#fecfea"],
    ["#4044d6", "#e1c7f3"],
    ["#d64053", "#fbb8c5"],
    ["#d69440", "#e5d295"],
    ["#40a4d6", "#79e0ef"],
    ["#6248e4", "#aac3c9"],
    ["#8ec22c", "#cfe7b4"],
]

export function RunListItem({
    session,
    index,
    data,
    onClick,
}: RunListItemProps) {
    const [, setLocation] = useLocation();
    const canEdit = useMemo<boolean>(() => data?.routine?.role ? [MemberRole.Admin, MemberRole.Owner].includes(data.routine.role) : false, [data]);
    const profileColors = useMemo(() => colorOptions[Math.floor(Math.random() * colorOptions.length)], []);
    const { bio, name } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            bio: getTranslation(data, 'bio', languages, true),
            name: getTranslation(data, 'name', languages, true),
        };
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If onClick provided, call it
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Run}/${data.routine?.id ?? ''}?run=${data.id}`);
    }, [onClick, data, setLocation]);

    return (
        <Tooltip placement="top" title="View details">
            <ListItem
                disablePadding
                onClick={handleClick}
                sx={{
                    display: 'flex',
                    background: index % 2 === 0 ? 'default' : '#e9e9e9',
                }}
            >
                <ListItemButton component="div" onClick={handleClick}>
                    <Box
                        width="50px"
                        minWidth="50px"
                        height="50px"
                        borderRadius='100%'
                        bgcolor={profileColors[0]}
                        justifyContent='center'
                        alignItems='center'
                        sx={{
                            display: { xs: 'none', sm: 'flex' },
                        }}
                    >
                        <ApartmentIcon sx={{
                            fill: profileColors[1],
                            width: '80%',
                            height: '80%',
                        }} />
                    </Box>
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        {/* Name/Title */}
                        <ListItemText
                            primary={name}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        {/* Bio/Description */}
                        <ListItemText
                            primary={bio}
                            sx={{ ...multiLineEllipsis(2), color: (t) => t.palette.text.secondary }}
                        />
                        {/* Tags */}
                        {Array.isArray(data.routine?.tags) && (data.routine as any).tags.length > 0 ? <TagList session={session} parentId={data.id ?? ''} tags={(data.routine as any).tags ?? []} /> : null}
                        {/* Progress bar */}
                        {/* TODO */}
                    </Stack>
                    {
                        canEdit ? null : <StarButton
                            session={session}
                            objectId={data.id ?? ''}
                            starFor={StarFor.Organization}
                            isStar={data.routine?.isStarred ?? false}
                            stars={data.routine?.stars ?? 0}
                            onChange={(isStar: boolean) => { }}
                        />
                    }
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const RunSortOptions: LabelledSortOption<RunSortBy>[] = labelledSortOptions(RunSortBy);
export const runDefaultSortOption = RunSortOptions[1];