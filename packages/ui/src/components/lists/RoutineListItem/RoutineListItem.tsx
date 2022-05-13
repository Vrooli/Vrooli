// Used to display popular/search results of a particular object type
import { ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { RoutineListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, RoutineSortBy, StarFor, VoteFor } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton, TagList, UpvoteDownvote } from '..';
import { getTranslation, LabelledSortOption, labelledSortOptions, listItemColor } from 'utils';
import { owns } from 'utils/authentication';

export function RoutineListItem({
    data,
    index,
    loading,
    session,
    onClick,
}: RoutineListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const canEdit = useMemo<boolean>(() => owns(data?.role), [data]);

    const { description, title } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            title: getTranslation(data, 'title', languages, true),
        }
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If data not supplied, don't open
        if (!data) return;
        // If onClick provided, call it
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Routine}/${data.id}`)
    }, [onClick, data, setLocation]);

    return (
        <Tooltip placement="top" title="View details">
            <ListItem
                disablePadding
                onClick={handleClick}
                sx={{
                    display: 'flex',
                    background: listItemColor(index, palette),
                }}
            >
                <ListItemButton component="div" onClick={handleClick}>
                    <UpvoteDownvote
                        session={session}
                        objectId={data?.id ?? ''}
                        voteFor={VoteFor.Routine}
                        isUpvoted={data?.isUpvoted}
                        score={data?.score}
                        onChange={(isUpvoted: boolean | null) => { }}
                    />
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        <ListItemText
                            primary={title}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        <ListItemText
                            primary={description}
                            sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                        />
                        {/* Tags */}
                        {Array.isArray(data?.tags) && (data?.tags as any).length > 0 ? <TagList session={session} parentId={data?.id ?? ''} tags={data?.tags ?? []} /> : null}
                    </Stack>
                    {
                        canEdit ? null : <StarButton
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.Routine}
                            isStar={data?.isStarred}
                            stars={data?.stars}
                            onChange={(isStar: boolean) => { }}
                        />
                    }
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const RoutineSortOptions: LabelledSortOption<RoutineSortBy>[] = labelledSortOptions(RoutineSortBy);
export const routineDefaultSortOption = RoutineSortOptions[1];