import { DeleteType, ReportFor } from "@local/consts";
import { exists, setDotNotationValue } from "@local/utils";
import { useCallback, useContext, useMemo, useState } from "react";
import { getAvailableActions, ObjectAction, ObjectActionComplete } from "../actions/objectActions";
import { getDisplay, getYou, getYouDot } from "../display/listTools";
import { openObject, openObjectEdit } from "../navigation/openObject";
import { PubSub } from "../pubsub";
import { SessionContext } from "../SessionContext";
import { useBookmarker } from "./useBookmarker";
import { useCopier } from "./useCopier";
import { useVoter } from "./useVoter";
const openDialogIfExists = (dialog) => {
    if (!exists(dialog)) {
        PubSub.get().publishSnack({ messageKey: "ActionNotSupported", severity: "Error" });
        return;
    }
    dialog();
};
export const useObjectActions = ({ beforeNavigation, object, objectType, openAddCommentDialog, setLocation, setObject, }) => {
    const session = useContext(SessionContext);
    const onActionComplete = useCallback((action, data) => {
        if (!exists(object)) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        switch (action) {
            case ObjectActionComplete.Bookmark:
            case ObjectActionComplete.BookmarkUndo:
                const isBookmarkedLocation = getYouDot(object, "isBookmarked");
                const wasSuccessful = action === ObjectActionComplete.Bookmark ? data.success : exists(data);
                if (wasSuccessful && isBookmarkedLocation && object)
                    setObject(setDotNotationValue(object, isBookmarkedLocation, wasSuccessful));
                break;
            case ObjectActionComplete.Fork:
                const forkData = Object.values(data).find((v) => typeof v === "object");
                openObject(forkData, setLocation);
                window.location.reload();
                break;
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                const reactionLocation = getYouDot(object, "reaction");
                const emoji = action === ObjectActionComplete.VoteUp ? "👍" : action === ObjectActionComplete.VoteDown ? "👎" : null;
                if (data.success && reactionLocation && object)
                    setObject(setDotNotationValue(object, reactionLocation, emoji));
                break;
        }
    }, [object, setLocation, setObject]);
    const { handleBookmark, hasBookmarkingSupport } = useBookmarker({
        objectId: object?.id,
        objectType: objectType,
        onActionComplete,
    });
    const { handleCopy, hasCopyingSupport } = useCopier({
        objectId: object?.id,
        objectName: getDisplay(object).title,
        objectType: objectType,
        onActionComplete,
    });
    const { handleVote, hasVotingSupport } = useVoter({
        objectId: object?.id,
        objectType: objectType,
        onActionComplete,
    });
    console.log("objectName", getDisplay(object));
    const hasDeletingSupport = exists(DeleteType[objectType]);
    const hasReportingSupport = exists(ReportFor[objectType]);
    const hasSharingSupport = useMemo(() => getYou(object).canShare, [object]);
    const hasStatsSupport = useMemo(() => ["Api", "Organization", "Project", "Quiz", "Routine", "SmartContract", "Standard", "User"].includes(objectType), [objectType]);
    const availableActions = useMemo(() => getAvailableActions(object, session), [object, session]);
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
    const [isDonateDialogOpen, setIsDonateDialogOpen] = useState(false);
    const [isShareDialogOpen, setIsShareDialogOpen] = useState(false);
    const [isStatsDialogOpen, setIsStatsDialogOpen] = useState(false);
    const [isReportDialogOpen, setIsReportDialogOpen] = useState(false);
    const openDeleteDialog = useCallback(() => setIsDeleteDialogOpen(true), [setIsDeleteDialogOpen]);
    const closeDeleteDialog = useCallback(() => setIsDeleteDialogOpen(false), [setIsDeleteDialogOpen]);
    const openDonateDialog = useCallback(() => setIsDonateDialogOpen(true), [setIsDonateDialogOpen]);
    const closeDonateDialog = useCallback(() => setIsDonateDialogOpen(false), [setIsDonateDialogOpen]);
    const openShareDialog = useCallback(() => setIsShareDialogOpen(true), [setIsShareDialogOpen]);
    const closeShareDialog = useCallback(() => setIsShareDialogOpen(false), [setIsShareDialogOpen]);
    const openStatsDialog = useCallback(() => setIsStatsDialogOpen(true), [setIsStatsDialogOpen]);
    const closeStatsDialog = useCallback(() => setIsStatsDialogOpen(false), [setIsStatsDialogOpen]);
    const openReportDialog = useCallback(() => setIsReportDialogOpen(true), [setIsReportDialogOpen]);
    const closeReportDialog = useCallback(() => setIsReportDialogOpen(false), [setIsReportDialogOpen]);
    const onActionStart = useCallback((action) => {
        console.log("onActionStart", action);
        if (!exists(object)) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        switch (action) {
            case ObjectAction.Comment:
                openDialogIfExists(openAddCommentDialog);
                break;
            case ObjectAction.Delete:
                openDialogIfExists(openDeleteDialog);
                break;
            case ObjectAction.Donate:
                openDialogIfExists(openDonateDialog);
                break;
            case ObjectAction.Edit:
                if (beforeNavigation && !beforeNavigation(object))
                    return;
                openObjectEdit(object, setLocation);
                break;
            case ObjectAction.FindInPage:
                PubSub.get().publishFindInPage();
                break;
            case ObjectAction.Fork:
                if (beforeNavigation && !beforeNavigation(object))
                    return;
                handleCopy();
                break;
            case ObjectAction.Report:
                openDialogIfExists(openReportDialog);
                break;
            case ObjectAction.Share:
                openDialogIfExists(openShareDialog);
                break;
            case ObjectAction.Bookmark:
            case ObjectAction.BookmarkUndo:
                handleBookmark(action === ObjectAction.Bookmark);
                break;
            case ObjectAction.Stats:
                openDialogIfExists(openStatsDialog);
                break;
            case ObjectAction.VoteDown:
            case ObjectAction.VoteUp:
                handleVote(action === ObjectAction.VoteUp ? "👍" : "👎");
                break;
        }
    }, [beforeNavigation, handleBookmark, handleCopy, handleVote, object, openAddCommentDialog, openDeleteDialog, openDonateDialog, openReportDialog, openShareDialog, openStatsDialog, setLocation]);
    return {
        availableActions,
        closeDeleteDialog,
        closeDonateDialog,
        closeShareDialog,
        closeStatsDialog,
        closeReportDialog,
        hasBookmarkingSupport,
        hasCopyingSupport,
        hasDeletingSupport,
        hasReportingSupport,
        hasSharingSupport,
        hasStatsSupport,
        hasVotingSupport,
        isDeleteDialogOpen,
        isDonateDialogOpen,
        isShareDialogOpen,
        isStatsDialogOpen,
        isReportDialogOpen,
        objectType,
        onActionStart,
        onActionComplete,
        openDeleteDialog,
        openDonateDialog,
        openShareDialog,
        openStatsDialog,
        openReportDialog,
    };
};
//# sourceMappingURL=useObjectActions.js.map