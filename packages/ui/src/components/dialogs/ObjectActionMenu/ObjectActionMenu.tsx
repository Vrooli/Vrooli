import { useMutation } from "@apollo/client";
import { copyVariables, copy_copy } from 'graphql/generated/copy';
import { starVariables, star_star } from 'graphql/generated/star';
import { voteVariables, vote_vote } from 'graphql/generated/vote';
import { copyMutation, starMutation, voteMutation } from "graphql/mutation";
import { useCallback, useMemo, useState } from "react";
import { ReportFor, StarFor, VoteFor } from "@shared/consts";
import { DeleteDialog, ListMenu, ReportDialog, SnackSeverity } from "..";
import { ObjectActionMenuProps } from "../types";
import { mutationWrapper } from "graphql/utils/graphqlWrapper";
import { getActionsDisplayData, getAvailableActions, getListItemTitle, getUserLanguages, ObjectAction, ObjectActionComplete, PubSub } from "utils";
import { DeleteType, CopyType } from "graphql/generated/globalTypes";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";

export const ObjectActionMenu = ({
    anchorEl,
    exclude,
    object,
    onActionComplete,
    onActionStart,
    onClose,
    session,
    zIndex,
}: ObjectActionMenuProps) => {

    const { availableActions, id, name, objectType } = useMemo(() => ({
        availableActions: getAvailableActions(object, session, exclude),
        id: object?.id,
        name: getListItemTitle(object, getUserLanguages(session)),
        objectType: object?.__typename,
    }), [exclude, object, session]);

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
    const [fork] = useMutation(copyMutation);
    const [star] = useMutation(starMutation);
    const [vote] = useMutation(voteMutation);

    const handleFork = useCallback(() => {
        if (!id) return;
        // Check if objectType can be converted to CopyType
        if(!Object.values(CopyType).includes(objectType as CopyType)) {
            PubSub.get().publishSnack({ messageKey: 'CopyNotSupported', severity: SnackSeverity.Error });
            return;
        }
        mutationWrapper<copy_copy, copyVariables>({
            mutation: fork,
            input: { id, intendToPullRequest: true, objectType: objectType as CopyType },
            successMessage: () => ({ key: 'CopySuccess', variables: { objectName: name } }),
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
        }
        onClose();
    }, [handleFork, handleStar, handleVote, objectType, onActionStart, onClose, openDelete, openDonate, openReport, openShare]);

    const data = useMemo(() => getActionsDisplayData(availableActions), [availableActions]);

    return (
        <>
            {/* Delete routine confirmation dialog */}
            {id && Object.values(DeleteType).includes(objectType as DeleteType) && <DeleteDialog
                isOpen={deleteOpen}
                objectId={id}
                objectType={objectType as DeleteType}
                objectName={name}
                handleClose={closeDelete}
                zIndex={zIndex + 1}
            />}
            {/* Report dialog */}
            {id && Object.values(ReportFor).includes(objectType as ReportFor) && <ReportDialog
                forId={id}
                onClose={closeReport}
                open={reportOpen}
                reportFor={objectType as ReportFor}
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
            {/* Actual action dialog */}
            <ListMenu
                anchorEl={anchorEl}
                data={data}
                id={`${objectType}-options-menu-${id}`}
                onClose={onClose}
                onSelect={onSelect}
                zIndex={zIndex}
            />
        </>
    )
}