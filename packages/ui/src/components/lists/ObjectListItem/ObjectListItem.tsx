import { Box, Chip, LinearProgress, ListItem, ListItemText, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { ObjectListItemProps, ObjectListItemType } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { RunProject, RunRoutine, RunStatus, VoteFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { TagList, TextLoading, UpvoteDownvote } from '..';
import { getYou, getDisplay, getUserLanguages, ObjectAction, ObjectActionComplete, openObject, openObjectEdit, getObjectEditUrl, placeholderColor, usePress, useWindowSize, getObjectUrl, getCounts, getStarFor, getYouDot } from 'utils';
import { smallHorizontalScrollbar } from '../styles';
import { EditIcon, OrganizationIcon, SvgComponent, UserIcon } from '@shared/icons';
import { CommentsButton, ReportsButton, StarButton } from 'components/buttons';
import { ObjectActionMenu } from 'components/dialogs';
import { uuid } from '@shared/uuid';
import { setDotNotationValue } from '@shared/utils';
import { useTranslation } from 'react-i18next';

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
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const id = useMemo(() => data?.id ?? uuid(), [data]);

    const [object, setObject] = useState<T | null | undefined>(data);
    useEffect(() => { setObject(data) }, [data]);

    const profileColors = useMemo(() => placeholderColor(), []);
    const { canComment, canEdit, canVote, canStar, isStarred, isUpvoted } = useMemo(() => getYou(data), [data]);
    const { subtitle, title } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);
    const { score } = useMemo(() => getCounts(data), [data]);

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
        if (data === null || link.length === 0) return;
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
        if (isMobile && !['Organization', 'User'].includes(object?.__typename as any)) return null;
        // Show icons for organizations and users
        switch (object?.__typename) {
            case 'Organization':
            case 'User':
                const Icon: SvgComponent = object?.__typename === 'Organization' ? OrganizationIcon : UserIcon;
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
            case 'Project':
            case 'Routine':
            case 'Standard':
                return (
                    <UpvoteDownvote
                        disabled={!canVote}
                        session={session}
                        objectId={object?.id ?? ''}
                        voteFor={object?.__typename as VoteFor}
                        isUpvoted={isUpvoted}
                        score={score}
                        onChange={(isUpvoted: boolean | null, score: number) => { }}
                    />
                )
            default:
                return null;
        }
    }, [isMobile, object?.__typename, object?.id, score, profileColors, canVote, isUpvoted, session]);

    /**
     * Action buttons are shown as a column on wide screens, and 
     * a row on mobile. It displays 
     * the star, comments, and reports buttons.
     */
    const actionButtons = useMemo(() => {
        const commentableObjects: string[] = ['Project', 'Routine', 'Standard'];
        const reportsCount: number = getCounts(object).reports;
        const { starFor, starForId } = getStarFor(object);
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
                {!hideRole && canEdit &&
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
                {isMobile && ['Project', 'Routine', 'Standard'].includes(object?.__typename!) && (
                    <UpvoteDownvote
                        direction='row'
                        disabled={!canVote}
                        session={session}
                        objectId={object?.id ?? ''}
                        voteFor={object?.__typename as VoteFor}
                        isUpvoted={isUpvoted}
                        score={score}
                        onChange={(isUpvoted: boolean | null, score: number) => { }}
                    />
                )}
                {starFor && <StarButton
                    disabled={!canStar}
                    session={session}
                    objectId={starForId}
                    starFor={starFor}
                    isStar={isStarred}
                    stars={getCounts(object).stars}
                />}
                {commentableObjects.includes(object?.__typename ?? '') && (<CommentsButton
                    commentsCount={getCounts(object).comments}
                    disabled={!canComment}
                    object={object}
                />)}
                {!['RunRoutine', 'RunProject'].includes(object?.__typename!) && reportsCount > 0 && <ReportsButton
                    reportsCount={reportsCount}
                    object={object}
                />}
            </Stack>
        )
    }, [object, isMobile, hideRole, canEdit, id, editUrl, handleEditClick, palette.secondary.main, canVote, session, isUpvoted, score, canStar, isStarred, canComment]);

    /**
     * Run list items may get a progress bar
     */
    const progressBar = useMemo(() => {
        if (!['RunProject', 'RunRoutine'].includes(object?.__typename!)) return null;
        const completedComplexity = object.completedComplexity;
        const totalComplexity = (object as RunProject).projectVersion?.complexity ?? (object as RunRoutine).routineVersion?.complexity ?? null;
        const percentComplete = object.status === RunStatus.Completed ? 100 :
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
                const isUpvotedLocation = getYouDot(object, 'isUpvoted');
                if (data.success && isUpvotedLocation && object) setDotNotationValue(object, isUpvotedLocation as any, action === ObjectActionComplete.VoteUp);
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                const isStarredLocation = getYouDot(object, 'isStarred');
                if (data.success && isStarredLocation && object) setDotNotationValue(object, isStarredLocation as any, action === ObjectActionComplete.Star);
                break;
            case ObjectActionComplete.Fork:
                // Data is in first key with a value
                const forkData: any = Object.values(data).find((v) => typeof v === 'object');
                openObject(forkData, setLocation);
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
                            data && (data as any).isComplete === false && <Tooltip placement="top" title={t('common:MarkedIncomplete', { lng: getUserLanguages(session)[0] })}>
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
                            data && (data as any).isInternal === true && <Tooltip placement="top" title={t('common:MarkedInternal', { lng: getUserLanguages(session)[0] })}>
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