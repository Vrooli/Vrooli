import { Box, LinearProgress, ListItem, ListItemText, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { ObjectListItemProps, ObjectListItemType } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo, useState } from 'react';
import { StarFor, VoteFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { TagList, TextLoading, UpvoteDownvote } from '..';
import { getListItemIsStarred, getListItemPermissions, getListItemReportsCount, getListItemStarFor, getListItemStars, getListItemSubtitle, getListItemTitle, getUserLanguages, listItemColor, ObjectType, openObject, placeholderColor, usePress } from 'utils';
import { smallHorizontalScrollbar } from '../styles';
import { BranchIcon, CopyIcon, DeleteIcon, DonateIcon, DownvoteWideIcon, EditIcon, OpenInNewIcon, OrganizationIcon, ReportIcon, ShareIcon, StarFilledIcon, StarOutlineIcon, StatsIcon, SvgComponent, UpvoteWideIcon, UserIcon } from '@shared/icons';
import { CommentsButton, ReportsButton, StarButton } from 'components/buttons';
import { ListMenuItemData } from 'components/dialogs/types';
import { ListProject, ListRoutine, ListStandard } from 'types';
import { ListMenu } from 'components/dialogs';
import { uuid } from '@shared/uuid';
import { RunStatus } from 'graphql/generated/globalTypes';

enum ListItemAction {
    CopyLink = 'CopyLink',
    Delete = "Delete",
    Donate = "Donate",
    Edit = "Edit",
    Fork = "Fork",
    Open = "Open",
    Report = "Report",
    Share = "Share",
    Star = "Star",
    StarUndo = "StarUndo",
    Stats = "Stats",
    VoteDown = "VoteDown",
    VoteUp = "VoteUp",
}

const contextOptionsMap: { [key in ListItemAction]?: [string, SvgComponent] } = {
    [ListItemAction.CopyLink]: ['Copy Link', CopyIcon],
    [ListItemAction.Delete]: ['Delete', DeleteIcon],
    [ListItemAction.Donate]: ['Donate', DonateIcon],
    [ListItemAction.Edit]: ['Edit', EditIcon],
    [ListItemAction.Fork]: ['Fork', BranchIcon],
    [ListItemAction.Open]: ['Open', OpenInNewIcon],
    [ListItemAction.Report]: ['Report', ReportIcon],
    [ListItemAction.Share]: ['Share', ShareIcon],
    [ListItemAction.Star]: ['Star', StarFilledIcon],
    [ListItemAction.StarUndo]: ['Unstar', StarOutlineIcon],
    [ListItemAction.Stats]: ['Stats', StatsIcon],
    [ListItemAction.VoteDown]: ['Vote Down', DownvoteWideIcon],
    [ListItemAction.VoteUp]: ['Vote Up', UpvoteWideIcon],
}

const listOptions: ListMenuItemData<ListItemAction>[] = Object.keys(contextOptionsMap).map(o => ({
    label: contextOptionsMap[o][0],
    value: o as ListItemAction,
    Icon: contextOptionsMap[o][1]
}));

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
    data,
    hideRole,
    index,
    loading,
    onClick,
    session,
    zIndex,
}: ObjectListItemProps<T>) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const id = useMemo(() => data?.id ?? uuid(), [data]);
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
        console.log('handle long clickkkkkk', target)
        setAnchorEl(target as HTMLElement)
    }, []);
    const closeContextMenu = useCallback(() => setAnchorEl(null), []);

    const handleClick = useCallback((target: EventTarget) => {
        if (!target.id || !target.id.startsWith('list-item-')) return;
        // If data not supplied, don't open
        if (!data) return;
        // If onClick provided, call it
        if (onClick) onClick(data);
        // Otherwise, navigate to the object's page
        else openObject(data, setLocation);
    }, [onClick, data, setLocation]);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onRightClick: handleContextMenu,
    });

    /**
     * Left column is only shown on wide screens. It's either 
     * a vote button, an object icon, or nothing.
     */
    const leftColumn = useMemo(() => {
        // Show icons for organizations and users
        switch (data?.__typename) {
            case ObjectType.Organization:
            case ObjectType.User:
                const Icon: SvgComponent = data?.__typename === ObjectType.Organization ? OrganizationIcon : UserIcon;
                return (
                    <Box
                        width="50px"
                        minWidth="50px"
                        height="50px"
                        borderRadius='100%'
                        bgcolor={profileColors[0]}
                        justifyContent='center'
                        alignItems='center'
                        sx={{
                            display: 'flex',
                            pointerEvents: 'none',
                        }}
                    >
                        <Icon fill={profileColors[1]} width="35px" height="35px" />
                    </Box>
                )
            case ObjectType.Project:
            case ObjectType.Routine:
            case ObjectType.Standard:
                return (
                    <UpvoteDownvote
                        disabled={!permissions.canVote}
                        session={session}
                        objectId={data?.id ?? ''}
                        voteFor={data?.__typename as VoteFor}
                        isUpvoted={data?.isUpvoted}
                        score={data?.score}
                        onChange={(isUpvoted: boolean | null) => { }}
                    />
                )
            default:
                return null;
        }
    }, [data, permissions.canVote, profileColors, session]);

    /**
     * Right column is only shown on wide screens. It displays 
     * the star, comments, and reports buttons.
     */
    const rightColumn = useMemo(() => {
        const commentableObjects: string[] = [ObjectType.Project, ObjectType.Routine, ObjectType.Standard];
        const reportsCount: number = getListItemReportsCount(data);
        const starFor: StarFor | null = getListItemStarFor(data);
        return (
            <Stack direction="column" spacing={1} sx={{ pointerEvents: 'none' }}>
                {starFor && <StarButton
                    disabled={!permissions.canStar}
                    session={session}
                    objectId={data?.id ?? ''}
                    starFor={starFor}
                    isStar={getListItemIsStarred(data)}
                    stars={getListItemStars(data)}
                />}
                {commentableObjects.includes(data?.__typename ?? '') && (<CommentsButton
                    commentsCount={(data as ListProject | ListRoutine | ListStandard)?.commentsCount ?? 0}
                    disabled={!permissions.canComment}
                    object={data}
                />)}
                {data?.__typename !== ObjectType.Run && reportsCount > 0 && <ReportsButton
                    reportsCount={reportsCount}
                    object={data}
                />}
            </Stack>
        )
    }, [data, permissions, session]);

    /**
     * Run list items may get a progress bar
     */
    const progressBar = useMemo(() => {
        if (!data || data.__typename !== ObjectType.Run) return null;
        const completedComplexity = data?.completedComplexity ?? null;
        const totalComplexity = data?.routine?.complexity ?? null;
        const percentComplete = data?.status === RunStatus.Completed ? 100 :
            (completedComplexity && totalComplexity) ?
                Math.min(Math.round(completedComplexity / totalComplexity * 100), 100) :
                0
        return (<CompletionBar
            color="secondary"
            variant={loading ? 'indeterminate' : 'determinate'}
            value={percentComplete}
            sx={{ height: '15px' }}
        />)
    }, [data, loading]);


    return (
        <>
            {/* Context menu */}
            <ListMenu
                anchorEl={anchorEl}
                data={listOptions}
                id={`list-item-options-menu-${id}`}
                onClose={closeContextMenu}
                onSelect={() => { }} //TODO
                zIndex={zIndex + 1}
            />
            {/* List item */}
            <Tooltip placement="top" title={'Press to open, or hold for quick actions'}>
                <ListItem
                    id={`list-item-${id}`}
                    {...pressEvents}
                    disablePadding
                    sx={{
                        display: 'flex',
                        background: listItemColor(index, palette),
                        padding: '8px 16px',
                        cursor: 'pointer',
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
                                    {!hideRole && permissions.canEdit && <ListItemText
                                        primary={`(Can Edit)`}
                                        sx={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            width: '100%',
                                            color: palette.mode === 'light' ? '#fa4f4f' : '#f2a7a7',
                                            flex: 200,
                                            pointerEvents: 'none',
                                        }}
                                    />}
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
                    {rightColumn}
                </ListItem>
            </Tooltip>
        </>
    )
}