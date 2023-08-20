import { BookmarkFor, CopyType, DeleteType, exists, LINKS, ReactionFor, ReportFor, setDotNotationValue } from "@local/shared";
import { SessionContext } from "contexts/SessionContext";
import { Dispatch, SetStateAction, useCallback, useContext, useMemo, useState } from "react";
import { SetLocation } from "route";
import { NavigableObject } from "types";
import { getAvailableActions, ObjectAction, ObjectActionComplete } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay, getYou, getYouDot, ListObject } from "utils/display/listTools";
import { openObject, openObjectEdit } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { useBookmarker } from "./useBookmarker";
import { useCopier } from "./useCopier";
import { useVoter } from "./useVoter";

export type UseObjectActionsProps = {
    /** Checks if navigation away from the current page is allowed */
    canNavigate?: ((item: NavigableObject) => boolean | void) | null | undefined
    object: ListObject | null | undefined;
    objectType: ListObject["__typename"];
    onClick?: (item: NavigableObject) => void;
    openAddCommentDialog?: () => void;
    setLocation: SetLocation;
    setObject: Dispatch<SetStateAction<any>>;
}

export type UseObjectActionsReturn = {
    availableActions: ObjectAction[];
    closeBookmarkDialog: () => unknown;
    closeDeleteDialog: () => unknown;
    closeDonateDialog: () => unknown;
    closeShareDialog: () => unknown;
    closeStatsDialog: () => unknown;
    closeReportDialog: () => unknown;
    hasBookmarkingSupport: boolean;
    hasCopyingSupport: boolean;
    hasDeletingSupport: boolean;
    hasReportingSupport: boolean;
    hasSharingSupport: boolean;
    hasStatsSupport: boolean;
    hasVotingSupport: boolean;
    isBookmarkDialogOpen: boolean;
    isDeleteDialogOpen: boolean;
    isDonateDialogOpen: boolean;
    isShareDialogOpen: boolean;
    isStatsDialogOpen: boolean;
    isReportDialogOpen: boolean;
    objectType: ListObject["__typename"];
    onActionStart: (action: ObjectAction | `${ObjectAction}`) => unknown;
    onActionComplete: (action: ObjectActionComplete | `${ObjectActionComplete}`, data: any) => unknown;
    openDeleteDialog: () => unknown;
    openDonateDialog: () => unknown;
    openShareDialog: () => unknown;
    openStatsDialog: () => unknown;
    openReportDialog: () => unknown;
}

const openDialogIfExists = (dialog: (() => void) | null | undefined) => {
    if (!exists(dialog)) {
        PubSub.get().publishSnack({ messageKey: "ActionNotSupported", severity: "Error" });
        return;
    }
    dialog();
};

/**
 * Hook for updating state and navigating upon completing an action
 */
export const useObjectActions = ({
    canNavigate,
    object,
    objectType,
    onClick,
    openAddCommentDialog,
    setLocation,
    setObject,
}: UseObjectActionsProps): UseObjectActionsReturn => {
    const session = useContext(SessionContext);

    // Callback when an action is completed
    const onActionComplete = useCallback((action: ObjectActionComplete | `${ObjectActionComplete}`, data: any) => {
        if (!exists(object)) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        switch (action) {
            case ObjectActionComplete.Bookmark:
            case ObjectActionComplete.BookmarkUndo: {
                const isBookmarkedLocation = getYouDot(object, "isBookmarked");
                const wasSuccessful = action === ObjectActionComplete.Bookmark ? data.success : exists(data);
                if (wasSuccessful && isBookmarkedLocation && object) setObject(setDotNotationValue(object, isBookmarkedLocation as any, wasSuccessful));
                break;
            }
            case ObjectActionComplete.Fork: {
                // Data is in first key with a value
                const forkData: any = Object.values(data).find((v) => typeof v === "object");
                openObject(forkData, setLocation);
                window.location.reload();
                break;
            }
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp: {
                const reactionLocation = getYouDot(object, "reaction");
                const emoji = action === ObjectActionComplete.VoteUp ? "👍" : action === ObjectActionComplete.VoteDown ? "👎" : null;
                if (data.success && reactionLocation && object) setObject(setDotNotationValue(object, reactionLocation as any, emoji));
                break;
            }
        }
    }, [object, setLocation, setObject]);

    // Hooks for specific actions
    const {
        isBookmarkDialogOpen,
        handleBookmark,
        closeBookmarkDialog,
        hasBookmarkingSupport,
    } = useBookmarker({
        objectId: object?.id,
        objectType: objectType as BookmarkFor,
        onActionComplete,
    });
    const { handleCopy, hasCopyingSupport } = useCopier({
        objectId: object?.id,
        objectName: getDisplay(object).title,
        objectType: objectType as CopyType,
        onActionComplete,
    });
    const { handleVote, hasVotingSupport } = useVoter({
        objectId: object?.id,
        objectType: objectType as ReactionFor,
        onActionComplete,
    });

    // Determine which actions are available    
    const hasDeletingSupport = exists(DeleteType[objectType]);
    const hasReportingSupport = exists(ReportFor[objectType]);
    const hasSharingSupport = useMemo(() => getYou(object).canShare, [object]);
    const hasStatsSupport = useMemo(() => ["Api", "Organization", "Project", "Quiz", "Routine", "SmartContract", "Standard", "User"].includes(objectType), [objectType]);
    const availableActions = useMemo(() => getAvailableActions(object, session), [object, session]);

    // Dialog states
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState<boolean>(false);
    const [isDonateDialogOpen, setIsDonateDialogOpen] = useState<boolean>(false);
    const [isShareDialogOpen, setIsShareDialogOpen] = useState<boolean>(false);
    const [isStatsDialogOpen, setIsStatsDialogOpen] = useState<boolean>(false);
    const [isReportDialogOpen, setIsReportDialogOpen] = useState<boolean>(false);

    // Dialog openers/closers
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

    // Callback when an action is started
    const onActionStart = useCallback((action: ObjectAction | `${ObjectAction}`) => {
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
                if (onClick) onClick(object);
                if (canNavigate && !canNavigate(object)) return;
                if (object.__typename === "User" && getCurrentUser(session).id === object.id) setLocation(LINKS.SettingsProfile);
                else openObjectEdit(object, setLocation);
                break;
            case ObjectAction.FindInPage:
                PubSub.get().publishFindInPage();
                break;
            case ObjectAction.Fork:
                if (canNavigate && !canNavigate(object)) return;
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
    }, [canNavigate, handleBookmark, handleCopy, handleVote, object, onClick, openAddCommentDialog, openDeleteDialog, openDonateDialog, openReportDialog, openShareDialog, openStatsDialog, session, setLocation]);

    return {
        availableActions,
        closeBookmarkDialog,
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
        isBookmarkDialogOpen,
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