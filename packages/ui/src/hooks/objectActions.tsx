
import { BookmarkFor, CommentFor, CopyType, DUMMY_ID, DeleteType, LINKS, ReactionFor, ReportFor, endpointsActions, endpointsBookmark, endpointsComment, endpointsReaction, endpointsReport, exists, getReactionScore, setDotNotationValue, shapeBookmark, shapeComment, shapeReport, type Bookmark, type BookmarkCreateInput, type BookmarkSearchInput, type BookmarkSearchResult, type Comment, type CommentCreateInput, type CopyInput, type CopyResult, type Count, type DeleteManyInput, type DeleteOneInput, type ListObject, type ModelType, type ReactInput, type Report, type ReportCreateInput, type Role, type Success, type User } from "@vrooli/shared";
import { useCallback, useContext, useMemo, useRef, useState, type Dispatch, type SetStateAction } from "react";
import { fetchLazyWrapper } from "../api/fetchWrapper.js";
import { BulkDeleteDialog } from "../components/dialogs/BulkDeleteDialog/BulkDeleteDialog.js";
import { DeleteAccountDialog } from "../components/dialogs/DeleteAccountDialog/DeleteAccountDialog.js";
import { DeleteDialog } from "../components/dialogs/DeleteDialog/DeleteDialog.js";
import { type ObjectListItemProps } from "../components/lists/types.js";
import { SessionContext } from "../contexts/session.js";
import { useLocation } from "../route/router.js";
import { type SetLocation } from "../route/types.js";
import { useBookmarkListsStore } from "../stores/bookmarkListsStore.js";
import { BulkObjectAction, BulkObjectActionComplete, getAvailableBulkActions } from "../utils/actions/bulkObjectActions.js";
import { ObjectAction, ObjectActionComplete, getAvailableActions, type ActionCompletePayloads, type ActionStartPayloads } from "../utils/actions/objectActions.js";
import { getCurrentUser } from "../utils/authentication/session.js";
import { ELEMENT_IDS } from "../utils/consts.js";
import { getDisplay, getYouDot } from "../utils/display/listTools.js";
import { openObject, openObjectEdit } from "../utils/navigation/openObject.js";
import { PubSub } from "../utils/pubsub.js";
import { useLazyFetch } from "./useFetch.js";

const DEFAULT_BOOKMARK_LIST_LABEL = "Favorites";

type UseBookmarkerProps = {
    objectId: string | null | undefined;
    objectType: `${ModelType}` | undefined;
    onActionComplete: <T extends "Bookmark" | "BookmarkUndo">(action: T, data: ActionCompletePayloads[T]) => unknown;
};

/**
 * Hook for simplifying the use of adding and removing bookmarks on an object
 */
export function useBookmarker({
    objectId,
    objectType,
    onActionComplete,
}: UseBookmarkerProps) {
    const fetchBookmarkLists = useBookmarkListsStore(state => state.fetchBookmarkLists);

    const [addBookmark] = useLazyFetch<BookmarkCreateInput, Bookmark>(endpointsBookmark.createOne);
    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    // In most cases, we must query for bookmarks to remove them, since 
    // we usually only know that an object has a bookmark - not the bookmarks themselves
    const [getData] = useLazyFetch<BookmarkSearchInput, BookmarkSearchResult>(endpointsBookmark.findMany);

    const hasBookmarkingSupport = objectType && objectType in BookmarkFor;

    // Handle dialog for updating a bookmark's lists
    const [isBookmarkDialogOpen, setIsBookmarkDialogOpen] = useState<boolean>(false);
    const closeBookmarkDialog = useCallback(() => { setIsBookmarkDialogOpen(false); }, []);

    const handleAdd = useCallback(async function handleAddCallback() {
        if (!objectType || !objectId) {
            PubSub.get().publish("snack", { messageKey: "NotFound", severity: "Error" });
            return;
        }
        // Find the best bookmark list to add to
        const allBookmarkLists = await fetchBookmarkLists();
        const bookmarkListId = allBookmarkLists.length === 1
            ? allBookmarkLists[0].id
            : allBookmarkLists.find(list => list.label === DEFAULT_BOOKMARK_LIST_LABEL)?.id;
        fetchLazyWrapper<BookmarkCreateInput, Bookmark>({
            fetch: addBookmark,
            inputs: shapeBookmark.create({
                __typename: "Bookmark",
                id: DUMMY_ID,
                to: {
                    __typename: BookmarkFor[objectType],
                    id: objectId,
                },
                list: {
                    __typename: "BookmarkList",
                    __connect: Boolean(bookmarkListId),
                    id: bookmarkListId ?? DUMMY_ID,
                    label: bookmarkListId ? undefined : DEFAULT_BOOKMARK_LIST_LABEL,
                },
            }),
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Bookmark, data); },
        });
    }, [objectType, objectId, addBookmark, fetchBookmarkLists, onActionComplete]);

    const isRemoveProcessingRef = useRef<boolean>(false);
    const handleRemove = useCallback(async () => {
        if (isRemoveProcessingRef.current || !objectType || !objectId) return;
        isRemoveProcessingRef.current = true;

        // Fetch bookmarks for the given objectId and objectType.
        const result = await getData({
            [`${objectType[0].toLowerCase()}${objectType.slice(1)}Id`]: objectId,
        });

        // Extract bookmarks from the result.
        const bookmarks = result?.data?.edges.map(edge => edge.node);

        // If no bookmarks are found, display an error.
        if (!bookmarks || bookmarks.length === 0) {
            PubSub.get().publish("snack", { message: "Could not find bookmark", severity: "Error" });
            isRemoveProcessingRef.current = false;
            return;
        }

        // If there's only one bookmark, delete it.
        if (bookmarks.length === 1) {
            const deletionResult = await deleteOne({
                id: bookmarks[0].id,
                objectType: DeleteType.Bookmark,
            });
            if (deletionResult.data?.success) {
                onActionComplete(ObjectActionComplete.BookmarkUndo, deletionResult.data);
            }
            isRemoveProcessingRef.current = false;
            return;
        }

        // If there are multiple bookmarks, open a dialog for the user to select which to delete.
        setIsBookmarkDialogOpen(true);
        isRemoveProcessingRef.current = false;
    }, [getData, deleteOne, objectId, objectType, onActionComplete]);


    const handleBookmark = useCallback((isAdding: boolean) => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasBookmarkingSupport) {
            PubSub.get().publish("snack", { messageKey: "BookmarkNotSupported", severity: "Error" });
            return;
        }
        if (isAdding) {
            handleAdd();
        } else {
            handleRemove();
        }
    }, [handleAdd, handleRemove, hasBookmarkingSupport, objectId]);

    return {
        isBookmarkDialogOpen,
        handleBookmark,
        closeBookmarkDialog,
        hasBookmarkingSupport,
    };
}

type UseBulkDeleterProps = {
    onBulkActionComplete: (action: BulkObjectActionComplete.Delete, objectsDeleted: ListObject[]) => unknown;
    selectedData: ListObject[];
}

export function useBulkDeleter({
    onBulkActionComplete,
    selectedData,
}: UseBulkDeleterProps) {

    const hasDeletingSupport = selectedData.some(item => exists(DeleteType[item.__typename]));
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
    const closeDeleteDialog = useCallback(() => { setIsDeleteDialogOpen(false); }, []);

    const [deleteMany] = useLazyFetch<DeleteManyInput, Count>(endpointsActions.deleteMany);

    const doBulkDelete = useCallback((confirmedForDeletion: ListObject[]) => {
        if (!selectedData || !selectedData.length) {
            return;
        }
        fetchLazyWrapper<DeleteManyInput, Count>({
            fetch: deleteMany,
            inputs: {
                objects: confirmedForDeletion
                    .filter(c => c.id && exists(DeleteType[c.__typename]))
                    .map(c => ({ id: c.id as string, objectType: c.__typename as DeleteType })),
            },
            successMessage: () => ({ messageKey: "ObjectsDeleted", messageVariables: { count: confirmedForDeletion.length } }),
            onSuccess: () => {
                onBulkActionComplete(BulkObjectActionComplete.Delete, confirmedForDeletion);
                setIsDeleteDialogOpen(false);
            },
        });
    }, [deleteMany, selectedData, onBulkActionComplete]);

    const handleBulkDelete = useCallback(() => {
        if (!selectedData || !selectedData.length) {
            return;
        }
        let highestConfirmationLevel: ConfirmationLevel = "none";
        for (const object of selectedData) {
            let confirmationLevel = ObjectsToDeleteConfirmLevel[object.__typename as DeleteType];
            // Special case: Users with "isBot" set to true require minimal confirmation instead of full
            if (object.__typename === "User" && (object as Partial<User>).isBot === true) {
                confirmationLevel = "minimal";
            }
            if (confirmationLevel === "full") {
                highestConfirmationLevel = "full";
                break;
            } else if (confirmationLevel === "minimal" && highestConfirmationLevel === "none") {
                highestConfirmationLevel = "minimal";
            }
        }
        if (highestConfirmationLevel === "none") {
            // Delete without confirmation
            doBulkDelete(selectedData);
            return;
        }
        if (highestConfirmationLevel === "minimal") {
            // Show simple confirmation dialog
            PubSub.get().publish("alertDialog", {
                messageKey: "DeleteConfirmMultiple",
                messageVariables: { count: selectedData.length },
                severity: "Warning",
                buttons: [{
                    labelKey: "Delete",
                    onClick: () => { doBulkDelete(selectedData); },
                }, {
                    labelKey: "Cancel",
                }],
            });
            return;
        }
        // If here, assume full confirmation
        setIsDeleteDialogOpen(true);
    }, [selectedData, doBulkDelete]);

    let BulkDeleteDialogComponent: JSX.Element | null;
    if (hasDeletingSupport) {
        BulkDeleteDialogComponent = <BulkDeleteDialog
            isOpen={isDeleteDialogOpen}
            handleClose={doBulkDelete}
            selectedData={selectedData}
        />;
    } else {
        BulkDeleteDialogComponent = null;
    }

    return {
        closeDeleteDialog,
        handleBulkDelete,
        hasDeletingSupport,
        isDeleteDialogOpen,
        BulkDeleteDialogComponent,
    };
}

export type UseBulkObjectActionsProps<T extends ListObject = ListObject> = {
    allData: T[];
    selectedData: T[];
    setAllData: (data: T[]) => unknown;
    setSelectedData: (data: T[]) => unknown;
    setLocation: SetLocation;
};

export type UseBulkObjectActionsReturn = {
    availableActions: BulkObjectAction[];
    closeBookmarkDialog: () => unknown;
    closeDeleteDialog: () => unknown;
    closeProjectAddDialog: () => unknown;
    closeReportDialog: () => unknown;
    BulkDeleteDialogComponent: JSX.Element | null;
    isBookmarkDialogOpen: boolean;
    isDeleteDialogOpen: boolean;
    isProjectAddDialogOpen: boolean;
    isReportDialogOpen: boolean;
    onBulkActionStart: (action: BulkObjectAction | `${BulkObjectAction}`) => unknown;
    onBulkActionComplete: (action: BulkObjectActionComplete | `${BulkObjectActionComplete}`, data: unknown) => unknown;
};

function callIfExists(callback: (() => unknown) | null | undefined) {
    if (!exists(callback)) {
        PubSub.get().publish("snack", { messageKey: "ActionNotSupported", severity: "Error" });
        return;
    }
    callback();
}

export function useBulkObjectActions<T extends ListObject = ListObject>({
    allData,
    selectedData,
    setAllData,
    setSelectedData,
    _setLocation,
}: UseBulkObjectActionsProps<T>): UseBulkObjectActionsReturn {
    const session = useContext(SessionContext);

    // Callback when an action is completed
    const onBulkActionComplete = useCallback((action: BulkObjectActionComplete | `${BulkObjectActionComplete}`, data: unknown) => {
        console.log("onBulkActionComplete", action, data, selectedData);
        if (selectedData.length === 0) {
            return;
        }
        switch (action) {
            case BulkObjectActionComplete.Bookmark:
            case BulkObjectActionComplete.BookmarkUndo:
                // TODO
                break;
            case BulkObjectActionComplete.Delete: {
                setAllData(allData.filter(object => !(data as ListObject[]).find(deleted => deleted.id === object.id)));
                setSelectedData([]);
                break;
            }
            case BulkObjectActionComplete.Report:
                //TODO
                break;
        }
    }, [allData, selectedData, setAllData, setSelectedData]);

    const {
        closeDeleteDialog,
        handleBulkDelete,
        isDeleteDialogOpen,
        BulkDeleteDialogComponent,
    } = useBulkDeleter({
        selectedData,
        onBulkActionComplete,
    });

    // Determine which actions are available   
    const availableActions = useMemo(() => getAvailableBulkActions(selectedData, session), [selectedData, session]);

    // Dialog states
    const [isBookmarkDialogOpen, setIsBookmarkDialogOpen] = useState<boolean>(false);
    const [isProjectAddDialogOpen, setIsProjectAddDialogOpen] = useState<boolean>(false);
    const [isReportDialogOpen, setIsReportDialogOpen] = useState<boolean>(false);

    // Dialog openers/closers
    const _openBookmarkDialog = useCallback(() => setIsBookmarkDialogOpen(true), [setIsBookmarkDialogOpen]);
    const closeBookmarkDialog = useCallback(() => setIsBookmarkDialogOpen(false), [setIsBookmarkDialogOpen]);
    const openProjectAddDialog = useCallback(() => setIsProjectAddDialogOpen(true), [setIsProjectAddDialogOpen]);
    const closeProjectAddDialog = useCallback(() => setIsProjectAddDialogOpen(false), [setIsProjectAddDialogOpen]);
    const openReportDialog = useCallback(() => setIsReportDialogOpen(true), [setIsReportDialogOpen]);
    const closeReportDialog = useCallback(() => setIsReportDialogOpen(false), [setIsReportDialogOpen]);

    // Callback when an action is started
    const onBulkActionStart = useCallback((action: BulkObjectAction | `${BulkObjectAction}`) => {
        switch (action) {
            case BulkObjectAction.Bookmark:
            case BulkObjectAction.BookmarkUndo:
                //TODO
                break;
            case BulkObjectAction.Delete:
                callIfExists(handleBulkDelete);
                break;
            case BulkObjectAction.Export:
                //TODO
                break;
            case BulkObjectAction.ProjectAdd:
                callIfExists(openProjectAddDialog);
                break;
            case BulkObjectAction.Report:
                callIfExists(openReportDialog);
                break;
        }
    }, [handleBulkDelete, openProjectAddDialog, openReportDialog]);

    return {
        availableActions,
        closeBookmarkDialog,
        closeDeleteDialog,
        closeProjectAddDialog,
        closeReportDialog,
        BulkDeleteDialogComponent,
        isBookmarkDialogOpen,
        isDeleteDialogOpen,
        isProjectAddDialogOpen,
        isReportDialogOpen,
        onBulkActionStart,
        onBulkActionComplete,
    };
}

type UseCopierProps = {
    objectId: string | null | undefined;
    objectName: string | null | undefined;
    objectType: `${ModelType}` | undefined;
    onActionComplete: <T extends "Fork">(action: T, data: ActionCompletePayloads[T]) => unknown;
}

/**
 * Hook for simplifying the use of voting on an object
 */
export function useCopier({
    objectId,
    objectName,
    objectType,
    onActionComplete,
}: UseCopierProps) {
    const [copy] = useLazyFetch<CopyInput, CopyResult>(endpointsActions.copy);

    const hasCopyingSupport = objectType && objectType in CopyType;

    const handleCopy = useCallback(() => {
        // Validate objectId and objectType
        if (!objectType || !objectId) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasCopyingSupport) {
            console.error("Cannot copy this object type", objectType, objectId);
            PubSub.get().publish("snack", { messageKey: "CopyNotSupported", severity: "Error" });
            return;
        }
        fetchLazyWrapper<CopyInput, CopyResult>({
            fetch: copy,
            inputs: { id: objectId, intendToPullRequest: true, objectType: CopyType[objectType] },
            successMessage: () => ({ messageKey: "CopySuccess", messageVariables: { objectName: objectName ?? "" } }),
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Fork, data); },
        });
    }, [copy, hasCopyingSupport, objectId, objectName, objectType, onActionComplete]);

    return { handleCopy, hasCopyingSupport };
}

export type ConfirmationLevel = "none" | "minimal" | "full";

export const ObjectsToDeleteConfirmLevel: Record<DeleteType, ConfirmationLevel> = {
    Api: "full",
    ApiKey: "full",
    ApiKeyExternal: "full",
    ApiVersion: "minimal",
    Bookmark: "none",
    BookmarkList: "full",
    Chat: "minimal",
    ChatInvite: "none",
    ChatMessage: "minimal",
    ChatParticipant: "none",
    Code: "full",
    CodeVersion: "minimal",
    Comment: "minimal",
    Email: "minimal",
    Issue: "minimal",
    Member: "minimal",
    MemberInvite: "none",
    Meeting: "minimal",
    MeetingInvite: "none",
    Note: "minimal",
    NoteVersion: "minimal",
    Notification: "none",
    Phone: "minimal",
    Project: "full",
    ProjectVersion: "minimal",
    PullRequest: "minimal",
    PushDevice: "none",
    Reminder: "none",
    ReminderList: "minimal",
    Report: "minimal",
    Resource: "none",
    Role: "full",
    Routine: "full",
    RoutineVersion: "minimal",
    RunProject: "none",
    RunRoutine: "none",
    Schedule: "none",
    Standard: "full",
    StandardVersion: "minimal",
    Team: "full",
    Transfer: "minimal",
    User: "full",
    Wallet: "minimal",
};

export function useDeleter({
    object,
    objectType,
    onActionComplete,
}: {
    object: ListObject | null | undefined;
    objectType: `${ModelType}`;
    onActionComplete: <T extends "Delete">(action: T, data: ActionCompletePayloads[T]) => unknown;
}) {
    const [, setLocation] = useLocation();

    const hasDeletingSupport = objectType in DeleteType;
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
    const closeDeleteDialog = useCallback(() => { setIsDeleteDialogOpen(false); }, []);

    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const doDelete = useCallback(() => {
        if (!object || !objectType) {
            console.error("Missing object or objectType");
            return;
        }
        // Check if there are multiple versions
        const isVersioned = objectType.endsWith("Version");
        const hasOneVersion = isVersioned && Object.prototype.hasOwnProperty.call(object, "root") && ((object.root as { versionsCount?: number }).versionsCount ?? 0) === 1;
        // If there's only one version, delete the root object instead
        const deleteId = (hasOneVersion ? (object.root as { id: string }).id : object.id) as string;
        const deleteType = (hasOneVersion ? objectType.replace("Version", "") : objectType) as DeleteType;

        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteOne,
            inputs: { id: deleteId, objectType: deleteType },
            successCondition: (data) => data.success,
            successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { objectName: getDisplay(object).title } }),
            onSuccess: () => {
                onActionComplete(ObjectActionComplete.Delete, { __typename: "Success", success: true });
                // If we're on the page for the object being deleted, navigate away
                const onObjectsPage = window.location.pathname.startsWith(LINKS[objectType]);
                setIsDeleteDialogOpen(false);
                if (!onObjectsPage) return;
                const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));
                if (hasPreviousPage) {
                    window.history.back();
                } else {
                    setLocation(LINKS.Home);
                }
            },
            errorMessage: () => ({ messageKey: "FailedToDelete" }),
            onError: () => {
                onActionComplete(ObjectActionComplete.Delete, { __typename: "Success", success: false });
            },
        });
    }, [deleteOne, object, objectType, onActionComplete, setLocation]);

    const handleDelete = useCallback((objectOverride?: ListObject) => {
        const objectToDelete = objectOverride || object;
        // Find confirmation level for this object type
        let confirmationType = ObjectsToDeleteConfirmLevel[objectType as DeleteType];

        // Handle special cases
        // Case 1: Users with "isBot" set to true require minimal confirmation instead of full
        if (objectType === "User") {
            const user = objectToDelete as Partial<User>;
            if (user.isBot === true) {
                confirmationType = "minimal";
            }
        }
        // Case 2: non-admin roles require minimal confirmation instead of full
        if (objectType === "Role") {
            const role = objectToDelete as Partial<Role>;
            if (role.name !== "Admin") {
                confirmationType = "minimal";
            }
        }

        if (confirmationType === "none") {
            // Delete without confirmation
            doDelete();
            return;
        }
        if (confirmationType === "minimal") {
            // Show simple confirmation dialog
            PubSub.get().publish("alertDialog", {
                messageKey: "DeleteConfirm",
                buttons: [{
                    labelKey: "Delete",
                    onClick: doDelete,
                }, {
                    labelKey: "Cancel",
                }],
            });
            return;
        }
        // If here, assume full confirmation
        setIsDeleteDialogOpen(true);
    }, [objectType, object, doDelete]);

    let DeleteDialogComponent: JSX.Element | null;
    if (objectType === "User") {
        DeleteDialogComponent = <DeleteAccountDialog
            isOpen={isDeleteDialogOpen}
            handleClose={closeDeleteDialog}
        />;
    } else if (hasDeletingSupport) {
        DeleteDialogComponent = <DeleteDialog
            isOpen={isDeleteDialogOpen}
            handleClose={closeDeleteDialog}
            handleDelete={doDelete}
            objectName={getDisplay(object).title}
        />;
    } else {
        DeleteDialogComponent = null;
    }

    return {
        closeDeleteDialog,
        handleDelete,
        hasDeletingSupport,
        isDeleteDialogOpen,
        DeleteDialogComponent,
    };
}

export type UseObjectActionsProps = {
    isListReorderable?: boolean;
    object: ListObject | null | undefined;
    objectType: ListObject["__typename"] | undefined;
    setLocation: SetLocation;
    setObject: Dispatch<SetStateAction<ListObject>>;
} & Pick<ObjectListItemProps<ListObject>, "canNavigate" | "onClick"> & {
    onAction?: ObjectListItemProps<ListObject>["onAction"];
}

export type UseObjectActionsReturn = {
    availableActions: ObjectAction[];
    closeBookmarkDialog: () => unknown;
    closeCommentDialog: () => unknown;
    closeDeleteDialog: () => unknown;
    closeDonateDialog: () => unknown;
    closeShareDialog: () => unknown;
    closeStatsDialog: () => unknown;
    closeReportDialog: () => unknown;
    DeleteDialogComponent: JSX.Element | null;
    isBookmarkDialogOpen: boolean;
    isCommentDialogOpen: boolean;
    isDeleteDialogOpen: boolean;
    isDonateDialogOpen: boolean;
    isShareDialogOpen: boolean;
    isStatsDialogOpen: boolean;
    isReportDialogOpen: boolean;
    objectType: ListObject["__typename"] | undefined;
    onActionStart: <T extends keyof ActionStartPayloads>(action: T) => unknown;
    onActionComplete: <T extends keyof ActionCompletePayloads>(action: T, data: ActionCompletePayloads[T]) => unknown;
}

/** Hook for updating state and navigating upon completing an action */
export function useObjectActions({
    canNavigate,
    _isListReorderable, //TODO: Implement reordering
    object,
    objectType,
    onClick,
    onAction,
    setLocation,
    setObject,
}: UseObjectActionsProps): UseObjectActionsReturn {
    const session = useContext(SessionContext);

    // Callback when an action is completed
    const onActionComplete = useCallback(<T extends keyof ActionCompletePayloads>(
        action: T,
        data: ActionCompletePayloads[T],
    ) => {
        if (!exists(object)) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        switch (action) {
            case ObjectActionComplete.Bookmark: {
                const isBookmarkedLocation = getYouDot(object, "isBookmarked");
                if (isBookmarkedLocation) {
                    setObject(setDotNotationValue(object, isBookmarkedLocation, true));
                } else {
                    console.error("Bookmark location not found", { object, data });
                }
                break;
            }
            case ObjectActionComplete.BookmarkUndo: {
                const isBookmarkedLocation = getYouDot(object, "isBookmarked");
                if ((data as ActionCompletePayloads["BookmarkUndo"]).success === true && isBookmarkedLocation) {
                    setObject(setDotNotationValue(object, isBookmarkedLocation, false));
                } else {
                    console.error("Bookmark location not found, or bookmark undo failed", { object, isBookmarkedLocation, data });
                }
                break;
            }
            case ObjectActionComplete.Delete: {
                if (typeof onAction === "function" && object.id) {
                    onAction("Deleted", object.id);
                } else {
                    console.error("onAction not found, or object ID not found", { object, onAction });
                }
                break;
            }
            case ObjectActionComplete.Fork: {
                // Data is in first key with a value
                const forkData = Object.values(data as ActionCompletePayloads["Fork"]).find((v) => typeof v === "object");
                if (forkData) {
                    openObject(forkData as ListObject, setLocation);
                } else {
                    console.error("Fork data not found", { object, data });
                }
                break;
            }
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp: {
                const reactionLocation = getYouDot(object, "reaction");
                const emoji = action === ObjectActionComplete.VoteUp ? "👍" : action === ObjectActionComplete.VoteDown ? "👎" : null;
                if ((data as ActionCompletePayloads["VoteDown"] | ActionCompletePayloads["VoteUp"]).success === true && reactionLocation) {
                    setObject(setDotNotationValue(object, reactionLocation, emoji));
                } else {
                    console.error("Vote data not found, or vote failed", { object, data });
                }
                break;
            }
            case ObjectActionComplete.Comment: {
                // Comment added successfully - could update comment count or trigger refresh
                console.log("Comment added successfully", { object, data });
                // No specific UI update needed as the comment will appear in the comment list
                break;
            }
            case ObjectActionComplete.Report: {
                // Report submitted successfully
                console.log("Report submitted successfully", { object, data });
                // No specific UI update needed
                break;
            }
            case ObjectActionComplete.Share: {
                // Share completed successfully
                console.log("Share completed successfully", { object, data });
                // No specific UI update needed
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
        objectType: objectType?.replace("Version", "") as ModelType | undefined,
        onActionComplete,
    });
    const { handleCopy } = useCopier({
        objectId: object?.id,
        objectName: getDisplay(object).title,
        objectType: objectType as ModelType | undefined,
        onActionComplete,
    });
    const { handleVote } = useVoter({
        objectId: object?.root?.id ?? object?.id, // Can only vote on root objects
        objectType: objectType?.replace("Version", "") as ModelType | undefined,
        onActionComplete,
    });
    const {
        closeDeleteDialog,
        handleDelete,
        isDeleteDialogOpen,
        DeleteDialogComponent,
    } = useDeleter({
        object,
        objectType: objectType as ModelType,
        onActionComplete,
    });
    const {
        closeCommentDialog,
        handleAddComment: _handleAddComment,
        isCommentDialogOpen,
        openCommentDialog,
    } = useCommenter({
        objectId: object?.root?.id ?? object?.id, // Can only comment on root objects
        objectType: objectType?.replace("Version", "") as ModelType | undefined,
        onActionComplete,
    });
    const {
        closeReportDialog,
        handleReport: _handleReport,
        isReportDialogOpen,
        openReportDialog,
    } = useReporter({
        objectId: object?.root?.id ?? object?.id, // Can only report root objects
        objectType: objectType?.replace("Version", "") as ModelType | undefined,
        onActionComplete,
    });
    const {
        closeShareDialog,
        handleShare: _handleShare,
        isShareDialogOpen,
        openShareDialog,
    } = useSharer({
        objectId: object?.root?.id ?? object?.id, // Can only share root objects
        objectType: objectType?.replace("Version", "") as ModelType | undefined,
        onActionComplete,
    });

    // Determine which actions are available    
    const availableActions = useMemo(() => getAvailableActions(object, session), [object, session]);

    // Dialog states for hooks not yet implemented
    const [isDonateDialogOpen, setIsDonateDialogOpen] = useState<boolean>(false);
    const [isStatsDialogOpen, setIsStatsDialogOpen] = useState<boolean>(false);

    // Dialog openers/closers for hooks not yet implemented
    const openDonateDialog = useCallback(() => setIsDonateDialogOpen(true), [setIsDonateDialogOpen]);
    const closeDonateDialog = useCallback(() => setIsDonateDialogOpen(false), [setIsDonateDialogOpen]);
    const openStatsDialog = useCallback(() => setIsStatsDialogOpen(true), [setIsStatsDialogOpen]);
    const closeStatsDialog = useCallback(() => setIsStatsDialogOpen(false), [setIsStatsDialogOpen]);

    // Callback when an action is started
    const onActionStart = useCallback(<T extends keyof ActionStartPayloads>(
        action: T,
    ) => {
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
                callIfExists(openCommentDialog);
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
                PubSub.get().publish("menu", { id: ELEMENT_IDS.FindInPage, isOpen: true });
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
    }, [canNavigate, handleBookmark, handleCopy, handleDelete, handleVote, object, onClick, openCommentDialog, openDonateDialog, openReportDialog, openShareDialog, openStatsDialog, session, setLocation]);

    return {
        availableActions,
        closeBookmarkDialog,
        closeCommentDialog,
        closeDeleteDialog,
        closeDonateDialog,
        closeShareDialog,
        closeStatsDialog,
        closeReportDialog,
        DeleteDialogComponent,
        isBookmarkDialogOpen,
        isCommentDialogOpen,
        isDeleteDialogOpen,
        isDonateDialogOpen,
        isShareDialogOpen,
        isStatsDialogOpen,
        isReportDialogOpen,
        objectType,
        onActionStart,
        onActionComplete,
    };
}

type UseVoterProps = {
    objectId: string | null | undefined;
    objectType: `${ModelType}` | undefined;
    onActionComplete: <T extends "VoteDown" | "VoteUp">(action: T, data: ActionCompletePayloads[T]) => unknown;
}

/**
 * Hook for simplifying the use of voting on an object
 */
export function useVoter({
    objectId,
    objectType,
    onActionComplete,
}: UseVoterProps) {
    const [fetch] = useLazyFetch<ReactInput, Success>(endpointsReaction.createOne);

    const hasVotingSupport = objectType && objectType in ReactionFor;

    const handleVote = useCallback((emoji: string | null) => {
        // Validate objectId and objectType
        if (!objectType || !objectId) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasVotingSupport) {
            PubSub.get().publish("snack", { messageKey: "VoteNotSupported", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ReactInput, Success>({
            fetch,
            inputs: {
                emoji,
                forConnect: objectId,
                reactionFor: ReactionFor[objectType],
            },
            onSuccess: (data) => { onActionComplete(getReactionScore(emoji) > 0 ? ObjectActionComplete.VoteUp : ObjectActionComplete.VoteDown, data); },
        });
    }, [hasVotingSupport, fetch, objectId, objectType, onActionComplete]);

    return { handleVote, hasVotingSupport };
}

type UseCommenterProps = {
    objectId: string | null | undefined;
    objectType: `${ModelType}` | undefined;
    onActionComplete: <T extends "Comment">(action: T, data: ActionCompletePayloads[T]) => unknown;
};

/**
 * Hook for simplifying the use of adding comments on an object
 */
export function useCommenter({
    objectId,
    objectType,
    onActionComplete,
}: UseCommenterProps) {
    const [createComment] = useLazyFetch<CommentCreateInput, Comment>(endpointsComment.createOne);

    const hasCommentingSupport = objectType && objectType in CommentFor;
    const [isCommentDialogOpen, setIsCommentDialogOpen] = useState<boolean>(false);
    const closeCommentDialog = useCallback(() => { setIsCommentDialogOpen(false); }, []);

    const handleAddComment = useCallback((commentText: string, language = "en") => {
        // Validate objectId and objectType
        if (!objectType || !objectId) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasCommentingSupport) {
            PubSub.get().publish("snack", { messageKey: "CommentNotSupported", severity: "Error" });
            return;
        }
        fetchLazyWrapper<CommentCreateInput, Comment>({
            fetch: createComment,
            inputs: shapeComment.create({
                __typename: "Comment",
                id: DUMMY_ID,
                commentedOn: {
                    __typename: CommentFor[objectType],
                    id: objectId,
                },
                translations: [{
                    __typename: "CommentTranslation",
                    id: DUMMY_ID,
                    language,
                    text: commentText,
                }],
            }),
            successMessage: () => ({ messageKey: "CommentAdded" }),
            onSuccess: (data) => {
                onActionComplete(ObjectActionComplete.Comment as "Comment", data);
                setIsCommentDialogOpen(false);
            },
        });
    }, [objectType, objectId, createComment, hasCommentingSupport, onActionComplete]);

    const openCommentDialog = useCallback(() => {
        if (!hasCommentingSupport) {
            PubSub.get().publish("snack", { messageKey: "CommentNotSupported", severity: "Error" });
            return;
        }
        setIsCommentDialogOpen(true);
    }, [hasCommentingSupport]);

    return {
        isCommentDialogOpen,
        handleAddComment,
        openCommentDialog,
        closeCommentDialog,
        hasCommentingSupport,
    };
}

type UseReporterProps = {
    objectId: string | null | undefined;
    objectType: `${ModelType}` | undefined;
    onActionComplete: <T extends "Report">(action: T, data: ActionCompletePayloads[T]) => unknown;
};

/**
 * Hook for simplifying reporting an object
 */
export function useReporter({
    objectId,
    objectType,
    onActionComplete,
}: UseReporterProps) {
    const [createReport] = useLazyFetch<ReportCreateInput, Report>(endpointsReport.createOne);

    const hasReportingSupport = objectType && objectType in ReportFor;
    const [isReportDialogOpen, setIsReportDialogOpen] = useState<boolean>(false);
    const closeReportDialog = useCallback(() => { setIsReportDialogOpen(false); }, []);

    const handleReport = useCallback((reason: string, details?: string, language = "en") => {
        // Validate objectId and objectType
        if (!objectType || !objectId) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasReportingSupport) {
            PubSub.get().publish("snack", { messageKey: "ReportNotSupported", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ReportCreateInput, Report>({
            fetch: createReport,
            inputs: shapeReport.create({
                __typename: "Report",
                id: DUMMY_ID,
                createdFor: {
                    __typename: ReportFor[objectType],
                    id: objectId,
                },
                reason,
                details: details || "",
                language,
            }),
            successMessage: () => ({ messageKey: "ReportSubmitted" }),
            onSuccess: (data) => {
                onActionComplete(ObjectActionComplete.Report as "Report", data);
                setIsReportDialogOpen(false);
            },
        });
    }, [objectType, objectId, createReport, hasReportingSupport, onActionComplete]);

    const openReportDialog = useCallback(() => {
        if (!hasReportingSupport) {
            PubSub.get().publish("snack", { messageKey: "ReportNotSupported", severity: "Error" });
            return;
        }
        setIsReportDialogOpen(true);
    }, [hasReportingSupport]);

    return {
        isReportDialogOpen,
        handleReport,
        openReportDialog,
        closeReportDialog,
        hasReportingSupport,
    };
}

type UseSharerProps = {
    objectId: string | null | undefined;
    objectType: `${ModelType}` | undefined;
    onActionComplete?: <T extends "Share">(action: T, data: ActionCompletePayloads[T]) => unknown;
};

/**
 * Hook for simplifying sharing an object
 */
export function useSharer({
    objectId,
    objectType,
    onActionComplete,
}: UseSharerProps) {
    const [isShareDialogOpen, setIsShareDialogOpen] = useState<boolean>(false);
    const closeShareDialog = useCallback(() => { setIsShareDialogOpen(false); }, []);

    // Most objects can be shared via URL, so we have basic support for all objects
    const hasSharingSupport = Boolean(objectId && objectType);

    const handleShare = useCallback(async (shareData?: { 
        url?: string;
        title?: string;
        text?: string;
    }) => {
        if (!objectId || !objectType) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }

        // Use Web Share API if available
        if (navigator.share && shareData) {
            try {
                await navigator.share(shareData);
                PubSub.get().publish("snack", { messageKey: "ShareSuccess", severity: "Success" });
                if (onActionComplete) {
                    onActionComplete(ObjectActionComplete.Share as "Share", { success: true });
                }
                setIsShareDialogOpen(false);
                return;
            } catch (err) {
                // User cancelled or error occurred, fallback to dialog
                console.warn("Web Share API failed:", err);
            }
        }

        // Fallback to copy to clipboard
        if (shareData?.url) {
            try {
                await navigator.clipboard.writeText(shareData.url);
                PubSub.get().publish("snack", { messageKey: "LinkCopied", severity: "Success" });
                if (onActionComplete) {
                    onActionComplete(ObjectActionComplete.Share as "Share", { success: true });
                }
                setIsShareDialogOpen(false);
                return;
            } catch (err) {
                console.warn("Clipboard API failed:", err);
            }
        }

        // If all else fails, open the share dialog
        setIsShareDialogOpen(true);
    }, [objectId, objectType, onActionComplete]);

    const openShareDialog = useCallback(() => {
        if (!hasSharingSupport) {
            PubSub.get().publish("snack", { messageKey: "ShareNotSupported", severity: "Error" });
            return;
        }
        setIsShareDialogOpen(true);
    }, [hasSharingSupport]);

    return {
        isShareDialogOpen,
        handleShare,
        openShareDialog,
        closeShareDialog,
        hasSharingSupport,
    };
}
