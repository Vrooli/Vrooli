// Used to display popular/search results of a particular object type
import { ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { OrganizationListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback } from 'react';
import { APP_LINKS, StarFor } from '@local/shared';
import { useLocation } from 'wouter';
import { useMutation } from '@apollo/client';
import { star } from 'graphql/generated/star';
import { starMutation } from 'graphql/mutation';
import { StarButton } from '..';

export function OrganizationListItem({
    session,
    data,
    isOwn = false,
    onClick,
}: OrganizationListItemProps) {
    const [, setLocation] = useLocation();
    const [star] = useMutation<star>(starMutation);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If onClick provided, call it
        if (onClick) onClick(data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Organization}/${data.id}`)
    }, [onClick, data, setLocation]);

    const handleStar = useCallback((e: any, isStar: boolean) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        // Send star mutation
        star({
            variables: {
                input: {
                    isStar,
                    starFor: StarFor.Project,
                    forId: data.id
                }
            }
        });
    }, [data.id, star]);

    return (
        <Tooltip placement="top" title="View details">
            <ListItem
                disablePadding
                onClick={handleClick}
                sx={{
                    display: 'flex',
                }}
            >
                <ListItemButton component="div" onClick={handleClick}>
                    <ListItemText
                        primary={data.name}
                        sx={{ ...multiLineEllipsis(2) }}
                    />
                    <StarButton
                        session={session}
                        isStar={data.isStarred}
                        stars={data.stars}
                        onStar={handleStar}
                    />
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}