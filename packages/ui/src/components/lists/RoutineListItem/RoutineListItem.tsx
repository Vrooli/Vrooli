import { Chip, ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { RoutineListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, StarFor, VoteFor } from '@local/shared';
import { useLocation } from 'wouter';
import { CommentButton, ReportButton, StarButton, TagList, TextLoading, UpvoteDownvote } from '..';
import { getTranslation, listItemColor } from 'utils';

export function RoutineListItem({
    data,
    hideRole,
    index,
    loading,
    session,
    onClick,
    tooltip = 'View details',
}: RoutineListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

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
                        session={session}
                        objectId={data?.id ?? ''}
                        voteFor={VoteFor.Routine}
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
                                    overflow: 'auto',
                                }}>
                                    <ListItemText
                                        primary={title}
                                        sx={{ 
                                            ...multiLineEllipsis(1),
                                            lineBreak: 'anywhere',
                                        }}
                                    />
                                    {!hideRole && data?.role && <ListItemText
                                        primary={`(${data.role})`}
                                        sx={{ 
                                            display: 'flex',
                                            alignItems: 'center',
                                            width: '100%',
                                            color: '#f2a7a7',
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
                        <Stack direction="row" spacing={1}>
                            {/* Incomplete chip */}
                            {
                                data && !data.isComplete && <Tooltip placement="top" title="Marked as incomplete">
                                    <Chip
                                        label="Incomplete"
                                        size="small"
                                        sx={{
                                            backgroundColor: palette.error.main,
                                            color: palette.error.contrastText,
                                            width: 'fit-content',
                                        }} />
                                </Tooltip>
                            }
                            {/* Tags */}
                            {Array.isArray(data?.tags) && (data?.tags as any).length > 0 ? <TagList session={session} parentId={data?.id ?? ''} tags={data?.tags ?? []} /> : null}
                        </Stack>
                    </Stack>
                    {/* Star/Comment/Report */}
                    <Stack direction="column" spacing={1}>
                        <StarButton
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.Routine}
                            isStar={data?.isStarred}
                            stars={data?.stars}
                        />
                        <CommentButton
                            commentsCount={data?.commentsCount ?? 0}
                            object={data}
                        />
                        {(data?.reportsCount ?? 0) > 0 && <ReportButton
                            reportsCount={data?.reportsCount ?? 0}
                            object={data}
                        />}
                    </Stack>
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}