import { BookmarkFor, CopyType, DeleteType, GqlModelType, ReportFor, Session, VoteFor } from "@shared/consts";
import { SetLocation } from "@shared/route";
import { exists, setDotNotationValue } from "@shared/utils";
import { Dispatch, SetStateAction, useCallback, useMemo, useState } from "react";
import { NavigableObject } from "types";
import { getAvailableActions, ObjectAction, ObjectActionComplete } from "utils/actions";
import { getDisplay, getYou, getYouDot, ListObjectType } from "utils/display";
import { openObject, openObjectEdit } from "utils/navigation";
import { PubSub } from "utils/pubsub";
import { useBookmarker } from "./useBookmarker";
import { useCopier } from "./useCopier";
import { useVoter } from "./useVoter";

export type UseObjectActionsProps = {
    /**
     * Checks if navigation away from the current page is allowed
     */
    beforeNavigation?: ((item: NavigableObject) => boolean | void) | null | undefined
    object: ListObjectType | null | undefined;
    objectType: GqlModelType | `${GqlModelType}`;
    openAddCommentDialog?: () => void;
    session: Session;
    setLocation: SetLocation;
    setObject: Dispatch<SetStateAction<any>>;
}

export type UseObjectActionsReturn = {
    availableActions: ObjectAction[];
    hasBookmarkingSupport: boolean;
    closeDeleteDialog: () => void;
    closeDonateDialog: () => void;
    closeShareDialog: () => void;
    closeStatsDialog: () => void;
    closeReportDialog: () => void;
    hasCopyingSupport: boolean;
    hasDeletingSupport: boolean;
    hasReportingSupport: boolean;
    hasSharingSupport: boolean;
    hasStatsSupport: boolean;
    hasVotingSupport: boolean;
    isDeleteDialogOpen: boolean;
    isDonateDialogOpen: boolean;
    isShareDialogOpen: boolean;
    isStatsDialogOpen: boolean;
    isReportDialogOpen: boolean;
    objectType: GqlModelType | `${GqlModelType}`;
    onActionStart: (action: ObjectAction | `${ObjectAction}`) => void;
    onActionComplete: (action: ObjectActionComplete | `${ObjectActionComplete}`, data: any) => void;
    openDeleteDialog: () => void;
    openDonateDialog: () => void;
    openShareDialog: () => void;
    openStatsDialog: () => void;
    openReportDialog: () => void;
}

const openDialogIfExists = (dialog: (() => void) | null | undefined) => {
    if (!exists(dialog)) {
        PubSub.get().publishSnack({ messageKey: `ActionNotSupported`, severity: 'Error' });
        return;
    }
    dialog();
}

/**
 * Hook for updating state and navigating upon completing an action
 */
export const useObjectActions = ({
    beforeNavigation,
    object,
    objectType,
    openAddCommentDialog,
    session,
    setLocation,
    setObject,
}: UseObjectActionsProps): UseObjectActionsReturn => {
    // Callback when an action is completed
    const onActionComplete = useCallback((action: ObjectActionComplete | `${ObjectActionComplete}`, data: any) => {
        if (!exists(object)) {
            PubSub.get().publishSnack({ messageKey: `CouldNotReadObject`, severity: 'Error' });
            return;
        }
        switch (action) {
            case ObjectActionComplete.Bookmark:
            case ObjectActionComplete.BookmarkUndo:
                const isBookmarkedLocation = getYouDot(object, 'isBookmarked');
                const wasSuccessful = action === ObjectActionComplete.Bookmark ? data.success : exists(data);
                if (wasSuccessful && isBookmarkedLocation && object) setObject(setDotNotationValue(object, isBookmarkedLocation as any, wasSuccessful));
                break;
            case ObjectActionComplete.Fork:
                // Data is in first key with a value
                const forkData: any = Object.values(data).find((v) => typeof v === 'object');
                openObject(forkData, setLocation);
                window.location.reload();
                break;
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                const isUpvotedLocation = getYouDot(object, 'isUpvoted');
                if (data.success && isUpvotedLocation && object) setObject(setDotNotationValue(object, isUpvotedLocation as any, action === ObjectActionComplete.VoteUp));
                break;
        }
    }, [object, setLocation, setObject]);

    // Hooks for specific actions
    const { handleBookmark, hasBookmarkingSupport } = useBookmarker({
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
        objectType: objectType as VoteFor,
        onActionComplete,
    });

    // Determine which actions are available    
    const hasDeletingSupport = exists(DeleteType[objectType]);
    const hasReportingSupport = exists(ReportFor[objectType]);
    const hasSharingSupport = useMemo(() => getYou(object).canShare, [object]);
    const hasStatsSupport = useMemo(() => ['Api', 'Organization', 'Project', 'Quiz', 'Routine', 'SmartContract', 'Standard', 'User'].includes(objectType), [objectType]);
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
        if (!exists(object)) {
            PubSub.get().publishSnack({ messageKey: `CouldNotReadObject`, severity: 'Error' });
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
                if (beforeNavigation && !beforeNavigation(object)) return;
                openObjectEdit(object, setLocation);
                break;
            case ObjectAction.FindInPage:
                PubSub.get().publishFindInPage();
                break;
            case ObjectAction.Fork:
                if (beforeNavigation && !beforeNavigation(object)) return;
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
                handleVote(action === ObjectAction.VoteUp);
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
}