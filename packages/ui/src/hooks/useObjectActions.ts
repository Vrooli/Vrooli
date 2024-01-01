import { Count, exists, GqlModelType, LINKS, setDotNotationValue, Success } from "@local/shared";
import { ObjectListItemProps } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { Dispatch, SetStateAction, useCallback, useContext, useMemo, useState } from "react";
import { SetLocation } from "route";
import { getAvailableActions, ObjectAction, ObjectActionComplete } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay, getYouDot, ListObject } from "utils/display/listTools";
import { openObject, openObjectEdit } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { useBookmarker } from "./useBookmarker";
import { useCopier } from "./useCopier";
import { useDeleter } from "./useDeleter";
import { useVoter } from "./useVoter";

export type UseObjectActionsProps = {
    isListReorderable?: boolean;
    object: ListObject | null | undefined;
    objectType: ListObject["__typename"] | undefined;
    openAddCommentDialog?: () => unknown;
    setLocation: SetLocation;
    setObject: Dispatch<SetStateAction<any>>;
} & Pick<ObjectListItemProps<any>, "canNavigate" | "onClick"> & {
    onAction?: ObjectListItemProps<any>["onAction"];
}

export type UseObjectActionsReturn = {
    availableActions: ObjectAction[];
    closeBookmarkDialog: () => unknown;
    closeDeleteDialog: () => unknown;
    closeDonateDialog: () => unknown;
    closeShareDialog: () => unknown;
    closeStatsDialog: () => unknown;
    closeReportDialog: () => unknown;
    DeleteDialogComponent: JSX.Element | null;
    isBookmarkDialogOpen: boolean;
    isDeleteDialogOpen: boolean;
    isDonateDialogOpen: boolean;
    isShareDialogOpen: boolean;
    isStatsDialogOpen: boolean;
    isReportDialogOpen: boolean;
    objectType: ListObject["__typename"] | undefined;
    onActionStart: (action: ObjectAction | `${ObjectAction}`) => unknown;
    onActionComplete: (action: ObjectActionComplete | `${ObjectActionComplete}`, data: any) => unknown;
}

const callIfExists = (callback: (() => unknown) | null | undefined) => {
    if (!exists(callback)) {
        PubSub.get().publish("snack", { messageKey: "ActionNotSupported", severity: "Error" });
        return;
    }
    callback();
};

const toSuccess = (data: unknown) => {
    if ((data as Success).__typename === "Success") return (data as Success).success === true;
    if ((data as Count).__typename === "Count") return (data as Count).count > 0;
    return exists(data);
};

/** Hook for updating state and navigating upon completing an action */
export const useObjectActions = ({
    canNavigate,
    isListReorderable, //TODO: Implement reordering
    object,
    objectType,
    onClick,
    onAction,
    openAddCommentDialog,
    setLocation,
    setObject,
}: UseObjectActionsProps): UseObjectActionsReturn => {
    const session = useContext(SessionContext);

    // Callback when an action is completed
    const onActionComplete = useCallback((action: ObjectActionComplete | `${ObjectActionComplete}`, data: any) => {
        if (!exists(object)) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        switch (action) {
            case ObjectActionComplete.Bookmark:
            case ObjectActionComplete.BookmarkUndo: {
                const isBookmarkedLocation = getYouDot(object, "isBookmarked");
                const wasSuccessful = toSuccess(data);
                console.log("completedbookmark", action, data);
                if (wasSuccessful && isBookmarkedLocation) {
                    console.log("updating object", object);
                    setObject(setDotNotationValue(object, isBookmarkedLocation as any, action === ObjectActionComplete.Bookmark));
                }
                break;
            }
            case ObjectActionComplete.Delete:
                console.log("onactioncomplete delete", object, onAction);
                if (typeof onAction === "function") onAction("Deleted", object.id ?? "");
                break;
            case ObjectActionComplete.Fork: {
                // Data is in first key with a value
                const forkData: any = Object.values(data).find((v) => typeof v === "object");
                openObject(forkData, setLocation);
                break;
            }
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp: {
                const reactionLocation = getYouDot(object, "reaction");
                const emoji = action === ObjectActionComplete.VoteUp ? "👍" : action === ObjectActionComplete.VoteDown ? "👎" : null;
                if (toSuccess(data) && reactionLocation) setObject(setDotNotationValue(object, reactionLocation as any, emoji));
                break;
            }
        }
    }, [object, onAction, setLocation, setObject]);

    // Hooks for specific actions
    const {
        closeBookmarkDialog,
        handleBookmark,
        isBookmarkDialogOpen,
    } = useBookmarker({
        objectId: object?.root?.id ?? object?.id, // Can only bookmark root objects
        objectType: objectType?.replace("Version", "") as GqlModelType | undefined,
        onActionComplete,
    });
    const { handleCopy } = useCopier({
        objectId: object?.id,
        objectName: getDisplay(object).title,
        objectType: objectType as GqlModelType | undefined,
        onActionComplete,
    });
    const { handleVote } = useVoter({
        objectId: object?.root?.id ?? object?.id, // Can only vote on root objects
        objectType: objectType?.replace("Version", "") as GqlModelType | undefined,
        onActionComplete,
    });
    const {
        closeDeleteDialog,
        handleDelete,
        isDeleteDialogOpen,
        DeleteDialogComponent,
    } = useDeleter({
        object,
        objectType: objectType as GqlModelType,
        onActionComplete,
    });

    // Determine which actions are available    
    const availableActions = useMemo(() => getAvailableActions(object, session), [object, session]);

    // Dialog states
    const [isDonateDialogOpen, setIsDonateDialogOpen] = useState<boolean>(false);
    const [isShareDialogOpen, setIsShareDialogOpen] = useState<boolean>(false);
    const [isStatsDialogOpen, setIsStatsDialogOpen] = useState<boolean>(false);
    const [isReportDialogOpen, setIsReportDialogOpen] = useState<boolean>(false);

    // Dialog openers/closers
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
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        switch (action) {
            case ObjectAction.Bookmark:
            case ObjectAction.BookmarkUndo:
                handleBookmark(action === ObjectAction.Bookmark);
                break;
            case ObjectAction.Comment:
                callIfExists(openAddCommentDialog);
                break;
            case ObjectAction.Delete:
                callIfExists(handleDelete);
                break;
            case ObjectAction.Donate:
                callIfExists(openDonateDialog);
                break;
            case ObjectAction.Edit:
                if (onClick) onClick(object);
                if (canNavigate && !canNavigate(object)) return;
                if (object.__typename === "User" && getCurrentUser(session).id === object.id) setLocation(LINKS.SettingsProfile);
                else openObjectEdit(object, setLocation);
                break;
            case ObjectAction.FindInPage:
                PubSub.get().publish("findInPage");
                break;
            case ObjectAction.Fork:
                if (canNavigate && !canNavigate(object)) return;
                handleCopy();
                break;
            case ObjectAction.Report:
                callIfExists(openReportDialog);
                break;
            case ObjectAction.Share:
                callIfExists(openShareDialog);
                break;
            case ObjectAction.Stats:
                callIfExists(openStatsDialog);
                break;
            case ObjectAction.VoteDown:
            case ObjectAction.VoteUp:
                handleVote(action === ObjectAction.VoteUp ? "👍" : "👎");
                break;
        }
    }, [canNavigate, handleBookmark, handleCopy, handleDelete, handleVote, object, onClick, openAddCommentDialog, openDonateDialog, openReportDialog, openShareDialog, openStatsDialog, session, setLocation]);

    return {
        availableActions,
        closeBookmarkDialog,
        closeDeleteDialog,
        closeDonateDialog,
        closeShareDialog,
        closeStatsDialog,
        closeReportDialog,
        DeleteDialogComponent,
        isBookmarkDialogOpen,
        isDeleteDialogOpen,
        isDonateDialogOpen,
        isShareDialogOpen,
        isStatsDialogOpen,
        isReportDialogOpen,
        objectType,
        onActionStart,
        onActionComplete,
    };
};
