// Used to display popular/search results of a particular object type
import { ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { StandardListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback } from 'react';
import { APP_LINKS, StandardSortBy, StarFor, VoteFor } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton, UpvoteDownvote } from '..';
import { useMutation } from '@apollo/client';
import { starMutation, voteMutation } from 'graphql/mutation';
import { vote } from 'graphql/generated/vote';
import { star } from 'graphql/generated/star';
import { LabelledSortOption, labelledSortOptions } from 'utils';
import { Standard } from 'types';

export function StandardListItem({
    session,
    data,
    isOwn = false,
    onClick,
}: StandardListItemProps) {
    const [, setLocation] = useLocation();
    const [vote] = useMutation<vote>(voteMutation);
    const [star] = useMutation<star>(starMutation);

    const handleClick = useCallback(() => {
        // If onClick provided, call if
        if (onClick) onClick(data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Standard}/${data.id}`)
    }, [onClick, data, setLocation]);

    const handleVote = useCallback((e: any, isUpvote: boolean | null) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        // Send vote mutation
        vote({ variables: { input: {
            isUpvote,
            voteFor: VoteFor.Project,
            forId: data.id
        } } });
    }, [data.id, vote]);

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
                    <UpvoteDownvote
                        session={session}
                        score={data.score}
                        isUpvoted={data.isUpvoted}
                        onVote={handleVote}
                    />
                    <Stack direction="column" spacing={1} pl={2} sx={{width: '-webkit-fill-available'}}>
                        <ListItemText
                            primary={data.name}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        <ListItemText
                            primary={data.description}
                            sx={{ ...multiLineEllipsis(2) }}
                        />
                    </Stack>
                    { isOwn ? null : <StarButton
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

export const StandardSortOptions: LabelledSortOption<StandardSortBy>[] = labelledSortOptions(StandardSortBy);
export const standardDefaultSortOption = StandardSortOptions[1];
export const standardOptionLabel = (o: Standard) => o.name ?? '';