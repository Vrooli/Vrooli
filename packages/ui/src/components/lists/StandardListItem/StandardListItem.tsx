import { ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { StandardListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { StarFor, VoteFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { TagList, TextLoading, UpvoteDownvote } from '..';
import { getTranslation, getUserLanguages, listItemColor, openObject } from 'utils';
import { smallHorizontalScrollbar } from '../styles';
import { CommentsButton, ReportsButton, StarButton } from 'components/buttons';

export function StandardListItem({
    data,
    hideRole,
    index,
    loading,
    session,
    onClick,
    tooltip = 'View details',
}: StandardListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { canComment, canEdit, canStar, canVote, description, reportsCount } = useMemo(() => {
        const { canComment, canEdit, canStar, canVote } = data?.permissionsStandard ?? {};
        const languages = getUserLanguages(session);
        const { description } = getTranslation(data, languages, true);
        return { canComment, canEdit, canStar, canVote, description, reportsCount: data?.reportsCount ?? 0 }
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If data not supplied, don't open
        if (!data) return;
        // If onClick provided, call it
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the object's page
        else openObject(data, setLocation);
    }, [onClick, data, setLocation]);

    return (
        <Tooltip placement="top" title={tooltip ?? 'View Details'}>
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
                        disabled={!canVote}
                        session={session}
                        objectId={data?.id ?? ''}
                        voteFor={VoteFor.Standard}
                        isUpvoted={data?.isUpvoted}
                        score={data?.score}
                        onChange={(isUpvoted: boolean | null) => { }}
                    />
                    <Stack
                        direction="column"
                        spacing={1}
                        pl={2}
                        sx={{
                            width: '-webkit-fill-available',
                            display: 'grid',
                        }}
                    >
                        {/* Name/Title and role */}
                        {loading ? <TextLoading /> :
                            (
                                <Stack direction="row" spacing={1} sx={{
                                    ...smallHorizontalScrollbar(palette),
                                }}>
                                    <ListItemText
                                        primary={data?.name}
                                        sx={{
                                            ...multiLineEllipsis(1),
                                            lineBreak: 'anywhere',
                                        }}
                                    />
                                    {!hideRole && canEdit && <ListItemText
                                        primary={`(Can Edit)`}
                                        sx={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            width: '100%',
                                            color: palette.mode === 'light' ? '#fa4f4f' : '#f2a7a7',
                                            flex: 200,
                                        }}
                                    />}
                                </Stack>
                            )
                        }
                        {loading ? <TextLoading /> : <ListItemText
                            primary={description}
                            sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                        />}
                        {/* Tags */}
                        {Array.isArray(data?.tags) && (data?.tags as any).length > 0 ?
                            <TagList
                                session={session}
                                parentId={data?.id ?? ''}
                                tags={data?.tags ?? []}
                                sx={{ ...smallHorizontalScrollbar(palette) }}
                            /> :
                            null}
                    </Stack>
                    {/* Star/Comment/Report */}
                    <Stack direction="column" spacing={1}>
                        <StarButton
                            disabled={!canStar}
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.Standard}
                            isStar={data?.isStarred}
                            stars={data?.stars}
                        />
                        <CommentsButton
                            commentsCount={data?.commentsCount ?? 0}
                            disabled={!canComment}
                            object={data}
                        />
                        {reportsCount > 0 && <ReportsButton
                            reportsCount={data?.reportsCount ?? 0}
                            object={data}
                        />}
                    </Stack>
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}