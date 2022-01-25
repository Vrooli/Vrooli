// Used to display popular/search results of a particular object type
import { ListItem, ListItemButton, ListItemText, Tooltip } from '@mui/material';
import { ActorListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback } from 'react';
import { APP_LINKS, StarFor, UserSortBy } from '@local/shared';
import { useLocation } from 'wouter';
import { useMutation } from '@apollo/client';
import { star } from 'graphql/generated/star';
import { starMutation } from 'graphql/mutation';
import { StarButton } from '..';
import { LabelledSortOption, labelledSortOptions } from 'utils';
import { User } from 'types';

export const ActorListItem = ({
    session,
    data,
    isOwn = false,
    onClick,
}: ActorListItemProps) => {
    const [, setLocation] = useLocation();
    const [star] = useMutation<star>(starMutation);

    const handleClick = useCallback(() => {
        // If onClick provided, call it
        if (onClick) onClick(data);
        // Otherwise, navigate to the actor's profile
        else setLocation(`${APP_LINKS.Profile}/${data.id}`)
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
                        primary={data.username}
                        sx={{ ...multiLineEllipsis(2) }}
                    />
                    {isOwn ? null : <StarButton
                        session={session}
                        isStar={data.isStarred}
                        stars={data.stars}
                        onStar={handleStar}
                    /> }
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const ActorSortOptions: LabelledSortOption<UserSortBy>[] = labelledSortOptions(UserSortBy);
export const actorDefaultSortOption = ActorSortOptions[1];
export const actorOptionLabel = (o: User) => o.username ?? '';