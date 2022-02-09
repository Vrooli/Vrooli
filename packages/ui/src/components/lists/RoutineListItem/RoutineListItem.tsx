// Used to display popular/search results of a particular object type
import { ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { RoutineListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback } from 'react';
import { APP_LINKS, RoutineSortBy, StarFor, VoteFor } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton, TagList, UpvoteDownvote } from '..';
import { useMutation } from '@apollo/client';
import { starMutation, voteMutation } from 'graphql/mutation';
import { vote } from 'graphql/generated/vote';
import { star } from 'graphql/generated/star';
import { LabelledSortOption, labelledSortOptions } from 'utils';
import { Routine } from 'types';

export function RoutineListItem({
    session,
    index,
    data,
    isOwn = false,
    onClick,
}: RoutineListItemProps) {
    const [, setLocation] = useLocation();
    const [vote] = useMutation<vote>(voteMutation);
    const [star] = useMutation<star>(starMutation);

    const handleClick = useCallback(() => {
        // If onClick provided, call if
        if (onClick) onClick(data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Routine}/${data.id}`)
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
                    background: index % 2 === 0 ? 'default' : '#e9e9e9',
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
                            primary={data.title}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        <ListItemText
                            primary={data.description}
                            sx={{ ...multiLineEllipsis(2), color: (t) => t.palette.text.secondary }}
                        />
                        {/* Tags */}
                        {Array.isArray(data.tags) && data.tags.length > 0 ? <TagList session={session} parentId={data.id ?? ''} tags={data.tags ?? []} /> : null}
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

export const RoutineSortOptions: LabelledSortOption<RoutineSortBy>[] = labelledSortOptions(RoutineSortBy);
export const routineDefaultSortOption = RoutineSortOptions[1];
export const routineOptionLabel = (o: Routine) => o.title ?? '';