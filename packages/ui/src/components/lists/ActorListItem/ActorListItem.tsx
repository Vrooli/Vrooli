// Used to display popular/search results of a particular object type
import { Box, ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { ActorListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, StarFor, UserSortBy } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton } from '..';
import { LabelledSortOption, labelledSortOptions } from 'utils';
import { User } from 'types';
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

export const ActorListItem = ({
    session,
    index,
    data,
    isOwn = false,
    onClick,
}: ActorListItemProps) => {
    const [, setLocation] = useLocation();

    const profileColors = useMemo(() => colorOptions[Math.floor(Math.random() * colorOptions.length)], []);

    const handleClick = useCallback(() => {
        // If onClick provided, call it
        if (onClick) onClick(data);
        // Otherwise, navigate to the actor's profile
        else setLocation(`${APP_LINKS.User}/${data.id}`)
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
                        <PersonIcon sx={{
                            fill: profileColors[1],
                            width: '80%',
                            height: '80%',
                        }} />
                    </Box>
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        <ListItemText
                            primary={data.username}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        <ListItemText
                            primary={data.bio}
                            sx={{ ...multiLineEllipsis(2), color: (t) => t.palette.text.secondary }}
                        />
                    </Stack>
                    {
                        isOwn ? null : <StarButton
                            session={session}
                            objectId={data.id ?? ''}
                            starFor={StarFor.User}
                            isStar={data.isStarred}
                            stars={data.stars}
                            onChange={(isStar: boolean) => { }}
                        />
                    }
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const ActorSortOptions: LabelledSortOption<UserSortBy>[] = labelledSortOptions(UserSortBy);
export const actorDefaultSortOption = ActorSortOptions[1];
export const actorOptionLabel = (o: User) => o.username ?? '';