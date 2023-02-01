import { useMutation } from "api/hooks";
import { useCallback, useMemo, useState } from "react";
import { CopyInput, CopyResult, CopyType, DeleteType, ReportFor, StarFor, StarInput, Success, VoteFor, VoteInput } from "@shared/consts";
import { DeleteDialog, ListMenu, ReportDialog, SnackSeverity } from "..";
import { ObjectActionMenuProps } from "../types";
import { mutationWrapper } from "api/utils";
import { getActionsDisplayData, getAvailableActions, getDisplay, getUserLanguages, ObjectAction, ObjectActionComplete, PubSub } from "utils";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";
import { copyCopy } from "api/generated/endpoints/copy";
import { starStar } from "api/generated/endpoints/star";
import { voteVote } from "api/generated/endpoints/vote";

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
        name: getDisplay(object, getUserLanguages(session)).title,
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
    const [fork] = useMutation<CopyResult, CopyInput, 'copy'>(copyCopy, 'copy');
    const [star] = useMutation<Success, StarInput, 'star'>(starStar, 'star');
    const [vote] = useMutation<Success, VoteInput, 'vote'>(voteVote, 'vote');

    const handleFork = useCallback(() => {
        if (!id) return;
        // Check if objectType can be converted to CopyType
        if(!Object.values(CopyType).includes(objectType as unknown as CopyType)) {
            PubSub.get().publishSnack({ messageKey: 'CopyNotSupported', severity: SnackSeverity.Error });
            return;
        }
        mutationWrapper<CopyResult, CopyInput>({
            mutation: fork,
            input: { id, intendToPullRequest: true, objectType: objectType as unknown as CopyType },
            successMessage: () => ({ key: 'CopySuccess', variables: { objectName: name } }),
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Fork, data) },
        })
    }, [fork, id, name, objectType, onActionComplete]);

    const handleStar = useCallback((isStar: boolean, starFor: StarFor) => {
        if (!id) return;
        mutationWrapper<Success, StarInput>({
            mutation: star,
            input: { isStar, starFor, forConnect: id },
            onSuccess: (data) => { onActionComplete(isStar ? ObjectActionComplete.Star : ObjectActionComplete.StarUndo, data) },
        })
    }, [id, onActionComplete, star]);

    const handleVote = useCallback((isUpvote: boolean | null, voteFor: VoteFor) => {
        if (!id) return;
        mutationWrapper<Success, VoteInput>({
            mutation: vote,
            input: { isUpvote, voteFor, forConnect: id },
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
            {id && Object.values(DeleteType).includes(objectType as unknown as DeleteType) && <DeleteDialog
                isOpen={deleteOpen}
                objectId={id}
                objectType={objectType as unknown as DeleteType}
                objectName={name}
                handleClose={closeDelete}
                zIndex={zIndex + 1}
            />}
            {/* Report dialog */}
            {id && Object.values(ReportFor).includes(objectType as unknown as ReportFor) && <ReportDialog
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