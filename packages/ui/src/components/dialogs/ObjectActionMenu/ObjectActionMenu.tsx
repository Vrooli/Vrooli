import { useMutation } from "@apollo/client";
import { forkVariables, fork_fork } from 'graphql/generated/fork';
import { starVariables, star_star } from 'graphql/generated/star';
import { voteVariables, vote_vote } from 'graphql/generated/vote';
import { forkMutation, starMutation, voteMutation } from "graphql/mutation";
import { useCallback, useMemo, useState } from "react";
import { ReportFor, StarFor, VoteFor } from "@shared/consts";
import { DeleteDialog, ListMenu, ReportDialog, SnackSeverity } from "..";
import { ObjectActionMenuProps } from "../types";
import { mutationWrapper } from "graphql/utils/graphqlWrapper";
import { getActionsDisplayData, getAvailableActions, getListItemTitle, getUserLanguages, ObjectAction, ObjectActionComplete, ObjectType, PubSub } from "utils";
import { DeleteOneType, ForkType } from "graphql/generated/globalTypes";
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
        objectType: object?.__typename as ObjectType,
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
    const [fork] = useMutation(forkMutation);
    const [star] = useMutation(starMutation);
    const [vote] = useMutation(voteMutation);

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