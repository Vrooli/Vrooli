// Used to display popular/search results of a particular object type
import { Box, List, ListItem, ListItemButton, ListItemText, Tooltip, Typography } from '@mui/material';
import { ActorListItemProps } from '../types';
import { centeredText, containerShadow } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS } from '@local/shared';

export function ActorListItem({
    data,
    isStarred = false,
    isOwn = false,
    onClick = () => { },
    onStarClick = () => { },
}: ActorListItemProps) {
    const handleClick = useCallback(() => onClick(data.id), [onClick, data.id]);
    const handleStarClick = useCallback(() => onStarClick(data.id, isStarred), [onStarClick, data.id, isStarred]);

    return (
        <Tooltip placement="right" title="Open">
            <ListItem disablePadding>
                <ListItemButton component="a" href={`${APP_LINKS.Profile}/${data.id}`}>
                    <ListItemText 
                        primary={data.username} 
                        sx={{maxWidth: '80%'}}
                    />
                    <ListItemText 
                        primary={data.stars} 
                        sx={{maxWidth: '20%'}}
                    />
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}