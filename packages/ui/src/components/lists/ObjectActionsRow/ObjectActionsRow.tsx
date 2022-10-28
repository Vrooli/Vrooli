import { useMutation } from "@apollo/client";
import { IconButton, Palette, Stack, Tooltip, useTheme } from "@mui/material";
import { CopyType, DeleteOneType, ForkType, ReportFor, StarFor, VoteFor } from "@shared/consts";
import { EllipsisIcon } from "@shared/icons";
import { DeleteDialog, ObjectActionMenu, ReportDialog, ShareObjectDialog, SnackSeverity } from "components/dialogs";
import { copyVariables, copy_copy } from "graphql/generated/copy";
import { forkVariables, fork_fork } from "graphql/generated/fork";
import { starVariables, star_star } from "graphql/generated/star";
import { voteVariables, vote_vote } from "graphql/generated/vote";
import { copyMutation, forkMutation, starMutation, voteMutation } from "graphql/mutation";
import { mutationWrapper } from "graphql/utils";
import React, { useCallback, useMemo, useState } from "react";
import { getActionsDisplayData, getAvailableActions, getListItemTitle, getUserLanguages, ObjectAction, ObjectActionComplete, ObjectType, PubSub } from "utils";
import { ObjectActionsRowProps } from "../types";

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
export const ObjectActionsRow = ({
    exclude,
    onActionComplete,
    onActionStart,
    object,
    session,
    zIndex,
}: ObjectActionsRowProps) => {
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
            name: getListItemTitle(object, getUserLanguages(session)),
            objectType: object?.__typename as ObjectType,
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

    // Mutations
    const [copy] = useMutation(copyMutation);
    const [fork] = useMutation(forkMutation);
    const [star] = useMutation(starMutation);
    const [vote] = useMutation(voteMutation);

    const handleCopy = useCallback(() => {
        if (!id) return;
        // Check if objectType can be converted to CopyType
        const copyType = CopyType[objectType];
        if (!copyType) {
            PubSub.get().publishSnack({ message: 'Copy not supported on this object type.', severity: SnackSeverity.Error });
            return;
        }
        mutationWrapper<copy_copy, copyVariables>({
            mutation: copy,
            input: { id, objectType: copyType },
            successMessage: () => `${name} copied.`,
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Copy, data) },
        })
    }, [copy, id, name, objectType, onActionComplete]);

    const handleFork = useCallback(() => {
        if (!id) return;
        // Check if objectType can be converted to ForkType
        const forkType = ForkType[objectType];
        if (!forkType) {
            PubSub.get().publishSnack({ message: 'Fork not supported on this object type.', severity: SnackSeverity.Error });
            return;
        }
        mutationWrapper<fork_fork, forkVariables>({
            mutation: fork,
            input: { id, objectType: forkType },
            successMessage: () => `${name} forked.`,
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Fork, data) },
        })
    }, [fork, id, name, objectType, onActionComplete]);

    const handleStar = useCallback((isStar: boolean, starFor: StarFor) => {
        if (!id) return;
        mutationWrapper<star_star, starVariables>({
            mutation: star,
            input: { isStar, starFor, forId: id },
            onSuccess: (data) => { onActionComplete(isStar ? ObjectActionComplete.Star : ObjectActionComplete.StarUndo, data) },
        })
    }, [id, onActionComplete, star]);

    const handleVote = useCallback((isUpvote: boolean | null, voteFor: VoteFor) => {
        if (!id) return;
        mutationWrapper<vote_vote, voteVariables>({
            mutation: vote,
            input: { isUpvote, voteFor, forId: id },
            onSuccess: (data) => { onActionComplete(isUpvote ? ObjectActionComplete.VoteUp : ObjectActionComplete.VoteDown, data) },
        })
    }, [id, onActionComplete, vote]);

    const onSelect = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Comment:
                onActionStart(ObjectAction.Comment);
                break;
            case ObjectAction.Copy:
                handleCopy();
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
                handleFork();
                break;
            case ObjectAction.Report:
                openReport();
                break;
            case ObjectAction.Share:
                openShare();
                break;
            case ObjectAction.Star:
            case ObjectAction.StarUndo:
                handleStar(action === ObjectAction.Star, objectType as string as StarFor);
                break;
            case ObjectAction.Stats:
                onActionStart(ObjectAction.Stats);
                break;
            case ObjectAction.VoteDown:
            case ObjectAction.VoteUp:
                handleVote(action === ObjectAction.VoteUp, objectType as string as VoteFor);
                break;
        }
    }, [handleCopy, handleFork, handleStar, handleVote, objectType, onActionStart, openDelete, openDonate, openReport, openShare]);

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
            {id && objectType in DeleteOneType && <DeleteDialog
                isOpen={deleteOpen}
                objectId={id}
                objectType={objectType as any}
                objectName={name}
                handleClose={closeDelete}
                zIndex={zIndex + 1}
            />}
            {/* Report dialog */}
            {id && objectType in ReportFor && <ReportDialog
                forId={id}
                onClose={closeReport}
                open={reportOpen}
                reportFor={objectType as any}
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
                title='Item Options'
                zIndex={zIndex + 1}
            />}
        </Stack>
    )
}