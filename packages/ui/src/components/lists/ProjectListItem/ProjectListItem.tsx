import { Chip, ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { ProjectListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, StarFor, VoteFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { CommentButton, ReportButton, StarButton, TagList, TextLoading, UpvoteDownvote } from 'components';
import { getTranslation, listItemColor } from 'utils';

export function ProjectListItem({
    data,
    hideRole,
    index,
    loading,
    session,
    onClick,
    tooltip = 'View details',
}: ProjectListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { canComment, canEdit, canStar, canReport, canVote, description, name, reportsCount } = useMemo(() => {
        const permissions = data?.permissionsProject;
        const languages = session?.languages ?? navigator.languages;
        return {
            canComment: permissions?.canComment === true,
            canEdit: permissions?.canEdit === true,
            canStar: permissions?.canStar === true,
            canReport: permissions?.canReport === true,
            canVote: permissions?.canVote === true,
            description: getTranslation(data, 'description', languages, true),
            name: getTranslation(data, 'name', languages, true),
            reportsCount: data?.reportsCount ?? 0,
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
        else {
            // Prefer using handle if available
            const link = data.handle ?? data.id;
            setLocation(`${APP_LINKS.Project}/${link}`);
        }
    }, [onClick, setLocation, data]);

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
                    {canVote && <UpvoteDownvote
                        session={session}
                        objectId={data?.id ?? ''}
                        voteFor={VoteFor.Project}
                        isUpvoted={data?.isUpvoted}
                        score={data?.score}
                        onChange={(isUpvoted: boolean | null) => { }}
                    />}
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
                                        primary={name}
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
                        {canStar && <StarButton
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.Project}
                            isStar={data?.isStarred}
                            stars={data?.stars}
                        />}
                        {canComment && <CommentButton
                            commentsCount={data?.commentsCount ?? 0}
                            object={data}
                        />}
                        {canReport && reportsCount > 0 && <ReportButton
                            reportsCount={data?.reportsCount ?? 0}
                            object={data}
                        />}
                    </Stack>
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}