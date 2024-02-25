import { DeleteOneInput, DeleteType, endpointPostDeleteOne, GqlModelType, LINKS, Role, Success, User } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { DeleteAccountDialog } from "components/dialogs/DeleteAccountDialog/DeleteAccountDialog";
import { DeleteDialog } from "components/dialogs/DeleteDialog/DeleteDialog";
import { useCallback, useState } from "react";
import { useLocation } from "route";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { getDisplay, ListObject } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

export type ConfirmationLevel = "none" | "minimal" | "full";

export const ObjectsToDeleteConfirmLevel: Record<DeleteType, ConfirmationLevel> = {
    Api: "full",
    ApiKey: "full",
    ApiVersion: "minimal",
    Bookmark: "none",
    BookmarkList: "full",
    Chat: "minimal",
    ChatInvite: "none",
    ChatMessage: "minimal",
    ChatParticipant: "none",
    Comment: "minimal",
    Email: "minimal",
    FocusMode: "minimal",
    Issue: "minimal",
    Member: "minimal",
    MemberInvite: "none",
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
    Role: "full",
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
    object,
    objectType,
    onActionComplete,
}: {
    object: ListObject | null | undefined;
    objectType: `${GqlModelType}`;
    onActionComplete: (action: ObjectActionComplete.Delete, data: boolean) => unknown;
}) => {
    const [, setLocation] = useLocation();

    const hasDeletingSupport = objectType in DeleteType;
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
    const closeDeleteDialog = useCallback(() => { setIsDeleteDialogOpen(false); }, []);

    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const doDelete = useCallback(() => {
        if (!object || !object.id || !objectType) {
            console.error("Missing object or objectType");
            return;
        }
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteOne,
            inputs: { id: object.id, objectType: objectType as DeleteType },
            successCondition: (data) => data.success,
            successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { objectName: getDisplay(object).title } }),
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
    }, [deleteOne, object, objectType, onActionComplete, setLocation]);

    const handleDelete = useCallback(() => {
        // Find confirmation level for this object type
        let confirmationType = ObjectsToDeleteConfirmLevel[objectType as DeleteType];

        // Handle special cases
        // Case 1: Users with "isBot" set to true require minimal confirmation instead of full
        if (objectType === "User") {
            const user = object as Partial<User>;
            if (user.isBot === true) {
                confirmationType = "minimal";
            }
        }
        // Case 2: non-admin roles require minimal confirmation instead of full
        if (objectType === "Role") {
            const role = object as Partial<Role>;
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
};
