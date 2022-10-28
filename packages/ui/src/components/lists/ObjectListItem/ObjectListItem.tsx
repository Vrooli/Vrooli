import { Box, Chip, LinearProgress, ListItem, ListItemText, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { ObjectListItemProps, ObjectListItemType } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { StarFor, VoteFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { TagList, TextLoading, UpvoteDownvote } from '..';
import { getListItemIsStarred, getListItemPermissions, getListItemReportsCount, getListItemStarFor, getListItemStars, getListItemSubtitle, getListItemTitle, getUserLanguages, ObjectAction, ObjectActionComplete, ObjectType, openObject, openObjectEdit, getObjectEditUrl, placeholderColor, usePress, useWindowSize, getObjectUrl } from 'utils';
import { smallHorizontalScrollbar } from '../styles';
import { EditIcon, OrganizationIcon, SvgComponent, UserIcon } from '@shared/icons';
import { CommentsButton, ReportsButton, StarButton } from 'components/buttons';
import { ListProject, ListRoutine, ListStandard } from 'types';
import { ObjectActionMenu } from 'components/dialogs';
import { uuid } from '@shared/uuid';
import { RunStatus } from 'graphql/generated/globalTypes';

function CompletionBar(props) {
    return (
        <Box sx={{ display: 'flex', alignItems: 'center', pointerEvents: 'none' }}>
            <Box sx={{ width: '100%', mr: 1 }}>
                <LinearProgress variant={props.variant} {...props} sx={{ borderRadius: 1, height: 8 }} />
            </Box>
            <Box sx={{ minWidth: 35 }}>
                <Typography variant="body2" color="text.secondary">{`${Math.round(
                    props.value,
                )}%`}</Typography>
            </Box>
        </Box>
    );
}

export function ObjectListItem<T extends ObjectListItemType>({
    beforeNavigation,
    data,
    hideRole,
    index,
    loading,
    session,
    zIndex,
}: ObjectListItemProps<T>) {
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const id = useMemo(() => data?.id ?? uuid(), [data]);

    const [object, setObject] = useState<T | null | undefined>(data);
    useEffect(() => { setObject(data) }, [data]);

    const profileColors = useMemo(() => placeholderColor(), []);
    const permissions = useMemo(() => getListItemPermissions(data, session), [data, session]);
    const { subtitle, title } = useMemo(() => {
        const languages = getUserLanguages(session);
        return {
            subtitle: getListItemSubtitle(data, languages),
            title: getListItemTitle(data, languages),
        };
    }, [data, session]);

    // Context menu
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const handleContextMenu = useCallback((target: EventTarget) => {
        setAnchorEl(target as HTMLElement)
    }, []);
    const closeContextMenu = useCallback(() => setAnchorEl(null), []);

    const link = useMemo(() => data ? getObjectUrl(data) : '', [data]);
    const handleClick = useCallback((target: EventTarget) => {
        if (!target.id || !target.id.startsWith('list-item-')) return;
        // If data not supplied, don't open
        if (link.length === 0) return;
        // If beforeNavigation is supplied, call it
        if (beforeNavigation) {
            const shouldContinue = beforeNavigation(data);
            if (shouldContinue === false) return;
        }
        // Navigate to the object's page
        setLocation(link);
    }, [link, beforeNavigation, setLocation, data]);

    const editUrl = useMemo(() => data ? getObjectEditUrl(data) : '', [data]);
    const handleEditClick = useCallback((event: any) => {
        event.preventDefault();
        const target = event.target;
        if (!target.id || !target.id.startsWith('edit-list-item-')) return;
        // If data not supplied, don't open
        if (!data) return;
        // If beforeNavigation is supplied, call it
        if (beforeNavigation) {
            const shouldContinue = beforeNavigation(data);
            if (shouldContinue === false) return;
        }
        // Navigate to the object's edit page
        setLocation(editUrl);
    }, [beforeNavigation, data, editUrl, setLocation]);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onRightClick: handleContextMenu,
    });

    /**
     * Left column is only shown on wide screens (if not a profile picture). It's either 
     * a vote button, an object icon, or nothing.
     */
    const leftColumn = useMemo(() => {
        if (isMobile && ![ObjectType.Organization, ObjectType.User].includes(object?.__typename as any)) return null;
        // Show icons for organizations and users
        switch (object?.__typename) {
            case ObjectType.Organization:
            case ObjectType.User:
                const Icon: SvgComponent = object?.__typename === ObjectType.Organization ? OrganizationIcon : UserIcon;
                return (
                    <Box
                        width={isMobile ? '40px' : '50px'}
                        minWidth={isMobile ? '40px' : '50px'}
                        height={isMobile ? '40px' : '50px'}
                        marginBottom={isMobile ? 'auto' : 'unset'}
                        borderRadius='100%'
                        bgcolor={profileColors[0]}
                        justifyContent='center'
                        alignItems='center'
                        sx={{
                            display: 'flex',
                            pointerEvents: 'none',
                        }}
                    >
                        <Icon
                            fill={profileColors[1]}
                            width={isMobile ? '25px' : '35px'}
                            height={isMobile ? '25px' : '35px'}
                        />
                    </Box>
                )
            case ObjectType.Project:
            case ObjectType.Routine:
            case ObjectType.Standard:
                return (
                    <UpvoteDownvote
                        disabled={!permissions.canVote}
                        session={session}
                        objectId={object?.id ?? ''}
                        voteFor={object?.__typename as VoteFor}
                        isUpvoted={object?.isUpvoted}
                        score={object?.score}
                        onChange={(isUpvoted: boolean | null, score: number) => { }}
                    />
                )
            default:
                return null;
        }
    }, [isMobile, permissions.canVote, object, profileColors, session]);

    /**
     * Action buttons are shown as a column on wide screens, and 
     * a row on mobile. It displays 
     * the star, comments, and reports buttons.
     */
    const actionButtons = useMemo(() => {
        const commentableObjects: string[] = [ObjectType.Project, ObjectType.Routine, ObjectType.Standard];
        const reportsCount: number = getListItemReportsCount(object);
        const starFor: StarFor | null = getListItemStarFor(object);
        return (
            <Stack
                direction={isMobile ? "row" : "column"}
                spacing={1}
                sx={{
                    pointerEvents: 'none',
                    justifyContent: isMobile ? 'right' : 'center',
                    alignItems: isMobile ? 'center' : 'start',
                }}
            >
                {!hideRole && permissions.canEdit &&
                    <Box
                        id={`edit-list-item-button-${id}`}
                        component="a"
                        href={editUrl}
                        onClick={handleEditClick}
                        sx={{
                            display: 'flex',
                            justifyContent: 'center',
                            alignItems: 'center',
                            cursor: 'pointer',
                            pointerEvents: 'all',
                            paddingBottom: isMobile ? '0px' : '4px',
                        }}>
                        <EditIcon id={`edit-list-item-icon${id}`} fill={palette.secondary.main} />
                    </Box>}
                {/* Add upvote/downvote if mobile */}
                {isMobile && [ObjectType.Project, ObjectType.Routine, ObjectType.Standard].includes(object?.__typename as any) && (
                    <UpvoteDownvote
                        direction='row'
                        disabled={!permissions.canVote}
                        session={session}
                        objectId={object?.id ?? ''}
                        voteFor={(object as any)?.__typename as VoteFor}
                        isUpvoted={(object as any)?.isUpvoted}
                        score={(object as any)?.score}
                        onChange={(isUpvoted: boolean | null, score: number) => { }}
                    />
                )}
                {starFor && <StarButton
                    disabled={!permissions.canStar}
                    session={session}
                    objectId={object?.id ?? ''}
                    starFor={starFor}
                    isStar={getListItemIsStarred(object)}
                    stars={getListItemStars(object)}
                />}
                {commentableObjects.includes(object?.__typename ?? '') && (<CommentsButton
                    commentsCount={(object as ListProject | ListRoutine | ListStandard)?.commentsCount ?? 0}
                    disabled={!permissions.canComment}
                    object={object}
                />)}
                {object?.__typename !== ObjectType.Run && reportsCount > 0 && <ReportsButton
                    reportsCount={reportsCount}
                    object={object}
                />}
            </Stack>
        )
    }, [editUrl, handleEditClick, hideRole, id, isMobile, object, palette.secondary.main, permissions.canComment, permissions.canEdit, permissions.canStar, permissions.canVote, session]);

    /**
     * Run list items may get a progress bar
     */
    const progressBar = useMemo(() => {
        if (!object || object.__typename !== ObjectType.Run) return null;
        const completedComplexity = object?.completedComplexity ?? null;
        const totalComplexity = object?.routine?.complexity ?? null;
        const percentComplete = object?.status === RunStatus.Completed ? 100 :
            (completedComplexity && totalComplexity) ?
                Math.min(Math.round(completedComplexity / totalComplexity * 100), 100) :
                0
        return (<CompletionBar
            color="secondary"
            variant={loading ? 'indeterminate' : 'determinate'}
            value={percentComplete}
            sx={{ height: '15px' }}
        />)
    }, [loading, object]);

    const onMoreActionStart = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Edit:
                // If data not supplied, don't open
                if (!data) return;
                // If beforeNavigation is supplied, call it
                if (beforeNavigation) {
                    const shouldContinue = beforeNavigation(data);
                    if (shouldContinue === false) return;
                }
                // Navigate to the object's edit page
                openObjectEdit(data, setLocation);
                break;
            case ObjectAction.Stats:
                //TODO
                break;
        }
    }, [beforeNavigation, data, setLocation]);

    const onMoreActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
        switch (action) {
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                if (data.success) {
                    setObject({
                        ...object,
                        isUpvoted: action === ObjectActionComplete.VoteUp,
                    } as any)
                }
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.success) {
                    setObject({
                        ...object,
                        isStarred: action === ObjectActionComplete.Star,
                    } as any)
                }
                break;
            case ObjectActionComplete.Fork:
                // Data is in first key with a value
                const forkData: any = Object.values(data).find((v) => typeof v === 'object');
                openObject(forkData, setLocation);
                window.location.reload();
                break;
            case ObjectActionComplete.Copy:
                // Data is in first key with a value
                const copyData: any = Object.values(data).find((v) => typeof v === 'object');
                openObject(copyData, setLocation);
                window.location.reload();
                break;
        }
    }, [object, setLocation]);

    return (
        <>
            {/* Context menu */}
            <ObjectActionMenu
                anchorEl={anchorEl}
                exclude={[ObjectAction.Comment, ObjectAction.FindInPage]} // Find in page only relevant when viewing object - not in list. And shouldn't really comment without viewing full page
                object={object}
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
                onClose={closeContextMenu}
                session={session}
                title='Item Options'
                zIndex={zIndex + 1}
            />
            {/* List item */}
            <ListItem
                id={`list-item-${id}`}
                disablePadding
                button
                component="a"
                href={link}
                {...pressEvents}
                onClick={(e) => { e.preventDefault() }}
                sx={{
                    display: 'flex',
                    background: palette.background.paper,
                    padding: '8px 16px',
                    cursor: 'pointer',
                    borderBottom: `1px solid ${palette.divider}`,
                }}
            >
                {leftColumn}
                <Stack
                    direction="column"
                    spacing={1}
                    pl={2}
                    sx={{
                        width: '-webkit-fill-available',
                        display: 'grid',
                        pointerEvents: 'none',
                    }}
                >
                    {/* Title */}
                    {loading ? <TextLoading /> :
                        (
                            <Stack id={`list-item-title-stack-${id}`} direction="row" spacing={1} sx={{
                                ...smallHorizontalScrollbar(palette),
                            }}>
                                <ListItemText
                                    primary={title}
                                    sx={{
                                        ...multiLineEllipsis(1),
                                        lineBreak: 'anywhere',
                                        pointerEvents: 'none',
                                    }}
                                />
                            </Stack>
                        )
                    }
                    {/* Subtitle */}
                    {loading ? <TextLoading /> : <ListItemText
                        primary={subtitle}
                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: 'none' }}
                    />}
                    {/* Progress bar */}
                    {progressBar}
                    <Stack direction="row" spacing={1} sx={{ pointerEvents: 'none' }}>
                        {/* Incomplete chip */}
                        {
                            data && (data as any).isComplete === false && <Tooltip placement="top" title="Marked as incomplete">
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
                        {/* Internal chip */}
                        {
                            data && (data as any).isInternal === true && <Tooltip placement="top" title="Marked as internal. Only the owner can use this routine">
                                <Chip
                                    label="Internal"
                                    size="small"
                                    sx={{
                                        backgroundColor: palette.warning.main,
                                        color: palette.error.contrastText,
                                        width: 'fit-content',
                                    }} />
                            </Tooltip>
                        }
                        {/* Tags */}
                        {Array.isArray((data as any)?.tags) && (data as any)?.tags.length > 0 ?
                            <TagList
                                session={session}
                                parentId={data?.id ?? ''}
                                tags={(data as any).tags}
                                sx={{ ...smallHorizontalScrollbar(palette) }}
                            /> :
                            null}
                    </Stack>
                    {/* Action buttons if mobile */}
                    {isMobile && actionButtons}
                </Stack>
                {!isMobile && actionButtons}
            </ListItem>
        </>
    )
}