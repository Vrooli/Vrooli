import { DeleteOneInput, DeleteType, endpointPostDeleteOne, exists, GqlModelType, LINKS, Success } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { DeleteAccountDialog } from "components/dialogs/DeleteAccountDialog/DeleteAccountDialog";
import { DeleteDialog } from "components/dialogs/DeleteDialog/DeleteDialog";
import { useCallback, useState } from "react";
import { useLocation } from "route";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

type ConfirmationLevel = "none" | "minimal" | "full";

const ObjectsToConfirmLevel: Record<DeleteType, ConfirmationLevel> = {
    Api: "full",
    ApiVersion: "minimal",
    Bookmark: "none",
    Chat: "minimal",
    ChatInvite: "none",
    ChatMessage: "none",
    ChatParticipant: "none",
    Comment: "minimal",
    Email: "minimal",
    FocusMode: "minimal",
    Issue: "minimal",
    Meeting: "minimal",
    MeetingInvite: "none",
    Node: "none",
    Note: "minimal",
    NoteVersion: "minimal",
    Notification: "none",
    Organization: "full",
    Post: "minimal",
    Project: "full",
    ProjectVersion: "minimal",
    PullRequest: "minimal",
    PushDevice: "none",
    Question: "minimal",
    QuestionAnswer: "none",
    Quiz: "minimal",
    Reminder: "none",
    ReminderList: "minimal",
    Report: "minimal",
    Resource: "none",
    Routine: "full",
    RoutineVersion: "minimal",
    RunProject: "none",
    RunRoutine: "none",
    Schedule: "none",
    SmartContract: "full",
    SmartContractVersion: "minimal",
    Standard: "full",
    StandardVersion: "minimal",
    Transfer: "minimal",
    User: "full",
    Wallet: "minimal",
};

export const useDeleter = ({
    objectId,
    objectType,
    objectName,
    onActionComplete,
}: {
    objectId: string | null | undefined;
    objectType: `${GqlModelType}`;
    objectName: string;
    onActionComplete: (action: ObjectActionComplete.Delete, data: boolean) => unknown;
}) => {
    const [, setLocation] = useLocation();

    const hasDeletingSupport = exists(DeleteType[objectType]);
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
    const closeDeleteDialog = useCallback(() => { setIsDeleteDialogOpen(false); }, []);

    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const doDelete = useCallback(() => {
        if (!objectId || !objectType) {
            console.error("Missing objectId or objectType");
            return;
        }
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteOne,
            inputs: { id: objectId, objectType: objectType as DeleteType },
            successCondition: (data) => data.success,
            successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { objectName } }),
            onSuccess: () => {
                onActionComplete(ObjectActionComplete.Delete, true);
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
                onActionComplete(ObjectActionComplete.Delete, false);
            },
        });
    }, [deleteOne, objectId, objectName, objectType, onActionComplete, setLocation]);

    const handleDelete = useCallback(() => {
        const confirmationType = ObjectsToConfirmLevel[objectType as DeleteType];
        console.log();
        if (confirmationType === "none") {
            // Delete without confirmation
            doDelete();
            return;
        }
        if (confirmationType === "minimal") {
            // Show simple confirmation dialog
            PubSub.get().publishAlertDialog({
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
    }, [objectType, doDelete]);

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
            objectName={objectName}
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
};
