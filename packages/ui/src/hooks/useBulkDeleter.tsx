import { Count, DeleteManyInput, DeleteType, endpointPostDeleteMany, exists, ListObject, User } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { BulkDeleteDialog } from "components/dialogs/BulkDeleteDialog/BulkDeleteDialog";
import { useCallback, useState } from "react";
import { BulkObjectActionComplete } from "utils/actions/bulkObjectActions";
import { PubSub } from "utils/pubsub";
import { ConfirmationLevel, ObjectsToDeleteConfirmLevel } from "./useDeleter";
import { useLazyFetch } from "./useLazyFetch";

export const useBulkDeleter = ({
    onBulkActionComplete,
    selectedData,
}: {
    onBulkActionComplete: (action: BulkObjectActionComplete.Delete, objectsDeleted: ListObject[]) => unknown;
    selectedData: ListObject[];
}) => {

    const hasDeletingSupport = selectedData.some(item => exists(DeleteType[item.__typename]));
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
    const closeDeleteDialog = useCallback(() => { setIsDeleteDialogOpen(false); }, []);

    const [deleteMany] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);

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
};
