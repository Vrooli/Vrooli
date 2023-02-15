import { useCallback, useMemo, useState } from "react";
import { CopyType, DeleteType, ReportFor, BookmarkFor, VoteFor } from "@shared/consts";
import { DeleteDialog, ListMenu, ReportDialog } from "..";
import { ObjectActionMenuProps } from "../types";
import { getActionsDisplayData, getAvailableActions, getDisplay, getUserLanguages, ObjectAction, PubSub, useBookmarker, useVoter } from "utils";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";
import { useCopier } from "utils/hooks/useCopier";

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

    const { handleBookmark } = useBookmarker({
        objectId: id,
        objectType: objectType as BookmarkFor,
        onActionComplete
    });

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
                handleBookmark(action === ObjectAction.Bookmark);
                break;
            case ObjectAction.Stats:
                onActionStart(ObjectAction.Stats);
                break;
            case ObjectAction.VoteDown:
            case ObjectAction.VoteUp:
                handleVote(action === ObjectAction.VoteUp);
        }
        onClose();
    }, [handleCopy, handleBookmark, handleVote, onActionStart, onClose, openDelete, openDonate, openReport, openShare]);

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