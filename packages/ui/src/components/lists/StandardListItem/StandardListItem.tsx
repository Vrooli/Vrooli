// Used to display popular/search results of a particular object type
import { ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { StandardListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback } from 'react';
import { APP_LINKS, StandardSortBy, StarFor, VoteFor } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton, TagList, UpvoteDownvote } from '..';
import { LabelledSortOption, labelledSortOptions } from 'utils';
import { Standard } from 'types';

export function StandardListItem({
    session,
    index,
    data,
    isOwn = false,
    onClick,
}: StandardListItemProps) {
    const [, setLocation] = useLocation();

    const handleClick = useCallback(() => {
        // If onClick provided, call if
        if (onClick) onClick(data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Standard}/${data.id}`)
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
                    <UpvoteDownvote
                        session={session}
                        objectId={data.id ?? ''}
                        voteFor={VoteFor.Standard}
                        isUpvoted={data.isUpvoted}
                        score={data.score}
                        onChange={(isUpvoted: boolean | null) => { }}
                    />
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        <ListItemText
                            primary={data.name}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        <ListItemText
                            primary={data.description}
                            sx={{ ...multiLineEllipsis(2), color: (t) => t.palette.text.secondary }}
                        />
                        {/* Tags */}
                        {Array.isArray(data.tags) && data.tags.length > 0 ? <TagList session={session} parentId={data.id ?? ''} tags={data.tags ?? []} /> : null}
                    </Stack>
                    {
                        isOwn ? null : <StarButton
                            session={session}
                            objectId={data.id ?? ''}
                            starFor={StarFor.Standard}
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

export const StandardSortOptions: LabelledSortOption<StandardSortBy>[] = labelledSortOptions(StandardSortBy);
export const standardDefaultSortOption = StandardSortOptions[1];
export const standardOptionLabel = (o: Standard) => o.name ?? '';