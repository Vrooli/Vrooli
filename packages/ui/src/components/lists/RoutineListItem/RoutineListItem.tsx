import { Chip, ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { RoutineListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo, useState } from 'react';
import { StarFor, VoteFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { TagList, TextLoading, UpvoteDownvote } from '..';
import { getTranslation, getUserLanguages, listItemColor, openObject, usePress } from 'utils';
import { smallHorizontalScrollbar } from '../styles';
import { CommentsButton, ReportsButton, StarButton } from 'components/buttons';
import { CopyIcon, SvgComponent } from '@shared/icons';
import { ListMenu } from 'components/dialogs';
import { ListMenuItemData } from 'components/dialogs/types';
import { uuid } from '@shared/uuid';

enum ListItemAction {
    CopyLink = 'CopyLink',
    CopyObject = 'CopyObject',
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
    // [BuildAction.AddIncomingLink]: ['Add incoming link', AddIncomingLinkIcon],
    // [BuildAction.AddOutgoingLink]: ['Add outgoing link', AddOutgoingLinkIcon],
    // [BuildAction.AddListBeforeNode]: ['Add routine list before', AddRoutineListBeforeIcon],
    // [BuildAction.AddListAfterNode]: ['Add routine list after', AddRoutineListAfterIcon],
    // [BuildAction.AddEndAfterNode]: ['Add end node after', AddEndNodeAfterIcon],
    // [BuildAction.DeleteNode]: ['Delete node', DeleteNodeIcon],
    // [BuildAction.MoveNode]: ['Move node', MoveNodeIcon],
    // [BuildAction.UnlinkNode]: ['Unlink node', UnlinkNodeIcon],
    // [BuildAction.EditSubroutine]: ['Edit subroutine', EditIcon],
    // [BuildAction.DeleteSubroutine]: ['Delete subroutine', DeleteIcon],
}

const listOptions: ListMenuItemData<ListItemAction>[] = Object.keys(contextOptionsMap).map(o => ({
    label: contextOptionsMap[o][0],
    value: o as ListItemAction,
    Icon: contextOptionsMap[o][1]
}));

export function RoutineListItem({
    data,
    hideRole,
    index,
    loading,
    session,
    onClick,
    tooltip = 'View details',
    // zIndex,
}: RoutineListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { canComment, canEdit, canStar, canVote, description, reportsCount, title } = useMemo(() => {
        const permissions = data?.permissionsRoutine;
        const languages = getUserLanguages(session);
        return {
            canComment: permissions?.canComment === true,
            canEdit: permissions?.canEdit === true,
            canStar: permissions?.canStar === true,
            canVote: permissions?.canVote === true,
            description: getTranslation(data, 'description', languages, true),
            reportsCount: data?.reportsCount ?? 0,
            title: getTranslation(data, 'title', languages, true),
        }
    }, [data, session]);

    // Context menu
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const handleContextMenu = useCallback((target: EventTarget) => {
        setAnchorEl(target as HTMLElement)
    }, []);
    const closeContextMenu = useCallback(() => setAnchorEl(null), []);

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

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onRightClick: handleContextMenu,
    });

    return (
        <>
            {/* Context menu */}
            <ListMenu
                anchorEl={anchorEl}
                data={listOptions}
                id={`routine-list-item-options-menu-${data?.id ?? uuid()}`}
                onClose={closeContextMenu}
                onSelect={() => {}} //TODO
                zIndex={201} //TODO
            />
            {/* List item */}
            <Tooltip placement="top" title={tooltip ?? 'View Details'}>
                <ListItem
                    {...pressEvents}
                    disablePadding
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
                                        ...smallHorizontalScrollbar(palette),
                                    }}>
                                        <ListItemText
                                            primary={title}
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
                            <Stack direction="row" spacing={1} sx={{
                                ...smallHorizontalScrollbar(palette),
                            }}>
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
                                {/* Internal chip */}
                                {
                                    data && data.isInternal && <Tooltip placement="top" title="Marked as internal. Only the owner can use this routine">
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
                                {Array.isArray(data?.tags) && (data?.tags as any).length > 0 ? <TagList session={session} parentId={data?.id ?? ''} tags={data?.tags ?? []} /> : null}
                            </Stack>
                        </Stack>
                        {/* Star/Comment/Report */}
                        <Stack direction="column" spacing={1}>
                            <StarButton
                                disabled={!canStar}
                                session={session}
                                objectId={data?.id ?? ''}
                                starFor={StarFor.Routine}
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
        </>
    )
}