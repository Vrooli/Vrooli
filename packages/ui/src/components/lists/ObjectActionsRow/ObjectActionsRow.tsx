import { IconButton, Palette, Stack, Tooltip, useTheme } from "@mui/material";
import { DeleteType, CopyType, ReportFor, BookmarkFor, VoteFor } from "@shared/consts";
import { EllipsisIcon } from "@shared/icons";
import { DeleteDialog, ObjectActionMenu, ReportDialog, ShareObjectDialog } from "components/dialogs";
import React, { useCallback, useMemo, useState } from "react";
import { getActionsDisplayData, getAvailableActions, getDisplay, getUserLanguages, ObjectAction, PubSub, useVoter } from "utils";
import { ObjectActionsRowProps, ObjectActionsRowObject } from "../types";
import { useCopier } from "utils/hooks/useCopier";

const commonButtonSx = (palette: Palette) => ({
    color: 'inherit',
    width: '48px',
    height: '100%',
})

const commonIconProps = (palette: Palette) => ({
    width: '30px',
    height: '30px',
})

/**
 * Horizontal list of action icons displayed on an object's view page. 
 * Available icons are same as ObjectActionMenu. Actions that are not available are hidden.
 * If there are more than 5 actions, rest are hidden in an overflow menu (i.e. ObjectActionMenu).
 */
export const ObjectActionsRow = <T extends ObjectActionsRowObject>({
    exclude,
    onActionComplete,
    onActionStart,
    object,
    session,
    zIndex,
}: ObjectActionsRowProps<T>) => {
    const { palette } = useTheme();

    const { actionsDisplayed, actionsExtra, id, name, objectType } = useMemo(() => {
        let availableActions = getAvailableActions(object, session, exclude);
        let actionsDisplayed: ObjectAction[];
        let actionsExtra: ObjectAction[];
        // If there are more than 5 actions, display the first 4 in the row, and the rest in the overflow menu
        if (availableActions.length > 5) {
            actionsDisplayed = availableActions.slice(0, 4);
            actionsExtra = availableActions.slice(4);
        }
        // If there are 5 or less actions, display them all in the row
        else {
            actionsDisplayed = availableActions;
            actionsExtra = [];
        }
        return {
            actionsDisplayed,
            actionsExtra,
            id: object?.id,
            name: getDisplay(object, getUserLanguages(session)).title,
            objectType: object?.__typename,
        }
    }, [exclude, object, session]);

    // States
    const [deleteOpen, setDeleteOpen] = useState<boolean>(false);
    const [donateOpen, setDonateOpen] = useState<boolean>(false);
    const [shareOpen, setShareOpen] = useState<boolean>(false);
    const [reportOpen, setReportOpen] = useState<boolean>(false);

    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);

    const openDonate = useCallback(() => setDonateOpen(true), [setDonateOpen]);
    const closeDonate = useCallback(() => setDonateOpen(false), [setDonateOpen]);

    const openShare = useCallback(() => setShareOpen(true), [setShareOpen]);
    const closeShare = useCallback(() => setShareOpen(false), [setShareOpen]);

    const openReport = useCallback(() => setReportOpen(true), [setReportOpen]);
    const closeReport = useCallback(() => setReportOpen(false), [setReportOpen]);

    const { handleCopy } = useCopier({
        objectId: id,
        objectName: name,
        objectType: objectType as CopyType,
        onActionComplete
    });

    const { handleVote } = useVoter({
        objectId: id,
        objectType: objectType as VoteFor,
        onActionComplete
    });

    const handleBookmark = useCallback((isBookmarked: boolean, bookmarkFor: BookmarkFor) => {
        if (!id) return;
        // mutationWrapper<Success, StarInput>({
        //     mutation: star,
        //     input: { isBookmarked, bookmarkFor, forConnect: id },
        //     onSuccess: (data) => { onActionComplete(isBookmarked ? ObjectActionComplete.Star : ObjectActionComplete.StarUndo, data) },
        // })
    }, [id, onActionComplete]);

    const onSelect = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Comment:
                onActionStart(ObjectAction.Comment);
                break;
            case ObjectAction.Delete:
                openDelete();
                break;
            case ObjectAction.Donate:
                openDonate();
                break;
            case ObjectAction.Edit:
                onActionStart(ObjectAction.Edit);
                break;
            case ObjectAction.FindInPage:
                PubSub.get().publishFindInPage();
                break;
            case ObjectAction.Fork:
                handleCopy();
                break;
            case ObjectAction.Report:
                openReport();
                break;
            case ObjectAction.Share:
                openShare();
                break;
            case ObjectAction.Bookmark:
            case ObjectAction.BookmarkUndo:
                handleBookmark(action === ObjectAction.Bookmark, objectType as string as BookmarkFor);
                break;
            case ObjectAction.Stats:
                onActionStart(ObjectAction.Stats);
                break;
            case ObjectAction.VoteDown:
            case ObjectAction.VoteUp:
                handleVote(action === ObjectAction.VoteUp);
                break;
        }
    }, [handleCopy, handleBookmark, handleVote, objectType, onActionStart, openDelete, openDonate, openReport, openShare]);

    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const openOverflowMenu = useCallback((event: React.MouseEvent) => {
        setAnchorEl(event.target as HTMLElement)
    }, []);
    const closeOverflowMenu = useCallback(() => setAnchorEl(null), []);

    const actions = useMemo(() => {
        const displayData = getActionsDisplayData(actionsDisplayed);
        const displayedActions = displayData.map((action, index) => {
            const { Icon, iconColor, label, value } = action;
            if (!Icon) return null;
            return <Tooltip title={label} key={index}>
                <IconButton sx={commonButtonSx(palette)} onClick={() => { onSelect(value) }}>
                    <Icon {...commonIconProps(palette)} fill={iconColor === 'default' ? palette.secondary.main : iconColor} />
                </IconButton>
            </Tooltip>
        })
        // If there are extra actions, display an ellipsis button
        if (actionsExtra.length > 0) {
            displayedActions.push(
                <Tooltip title="More" key={displayedActions.length}>
                    <IconButton sx={commonButtonSx(palette)} onClick={openOverflowMenu}>
                        <EllipsisIcon {...commonIconProps(palette)} fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
            )
        }
        return displayedActions;
    }, [actionsDisplayed, actionsExtra.length, onSelect, openOverflowMenu, palette]);

    return (
        <Stack
            direction="row"
            spacing={1}
            sx={{
                marginTop: 1,
                marginBottom: 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
            }}
        >
            {/* Delete routine confirmation dialog */}
            {id && objectType !== undefined && objectType in DeleteType && <DeleteDialog
                isOpen={deleteOpen}
                objectId={id}
                objectType={objectType as unknown as DeleteType}
                objectName={name}
                handleClose={closeDelete}
                zIndex={zIndex + 1}
            />}
            {/* Report dialog */}
            {id && objectType !== undefined && objectType in ReportFor && <ReportDialog
                forId={id}
                onClose={closeReport}
                open={reportOpen}
                reportFor={objectType as unknown as ReportFor}
                session={session}
                zIndex={zIndex + 1}
            />}
            {/* Share dialog */}
            <ShareObjectDialog
                object={object}
                open={shareOpen}
                onClose={closeShare}
                zIndex={zIndex + 1}
            />
            {/* Displayed actions */}
            {actions}
            {/* Overflow menu */}
            {actionsExtra.length > 0 && <ObjectActionMenu
                anchorEl={anchorEl}
                exclude={[...(exclude ?? []), ...actionsDisplayed]}
                object={object}
                onActionStart={onSelect}
                onActionComplete={onActionComplete}
                onClose={closeOverflowMenu}
                session={session}
                zIndex={zIndex + 1}
            />}
        </Stack>
    )
}