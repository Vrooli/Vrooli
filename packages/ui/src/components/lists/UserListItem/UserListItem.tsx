// Used to display popular/search results of a particular object type
import { Box, ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { UserListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, StarFor, UserSortBy } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton } from '..';
import { getTranslation, LabelledSortOption, labelledSortOptions } from 'utils';
import { Person as PersonIcon } from '@mui/icons-material';

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

export const UserListItem = ({
    data,
    index,
    loading,
    onClick,
    session,
}: UserListItemProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const isOwn = useMemo(() => data?.id === session?.id, [data, session]);

    const profileColors = useMemo(() => colorOptions[Math.floor(Math.random() * colorOptions.length)], []);

    const { bio, name } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            bio: getTranslation(data, 'bio', languages, true),
            name: data?.name ?? (data?.handle ? `$${data.handle}` : ''),
        }
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If data not supplied, don't open
        if (!data) return;
        // If onClick provided, call it
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the user's profile
        else {
            // Prefer using handle if available
            const link = data.handle ?? data.id;
            setLocation(`${APP_LINKS.Profile}/${link}`);
        }
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
                        alignItems='center'
                        bgcolor={profileColors[0]}
                        borderRadius='100%'
                        height="50px"
                        justifyContent='center'
                        minWidth="50px"
                        width="50px"
                        sx={{
                            display: { xs: 'none', sm: 'flex' },
                        }}
                    >
                        <PersonIcon sx={{
                            fill: profileColors[1],
                            height: '80%',
                            width: '80%',
                        }} />
                    </Box>
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        <ListItemText
                            primary={name}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        <ListItemText
                            primary={bio}
                            sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                        />
                    </Stack>
                    {
                        !isOwn && <StarButton
                            isStar={data?.isStarred}
                            objectId={data?.id ?? ''}
                            onChange={(isStar: boolean) => { }}
                            session={session}
                            starFor={StarFor.User}
                            stars={data?.stars}
                        />
                    }
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const UserSortOptions: LabelledSortOption<UserSortBy>[] = labelledSortOptions(UserSortBy);
export const userDefaultSortOption = UserSortOptions[1];