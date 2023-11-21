import { Count, DeleteManyInput, DeleteType, endpointPostDeleteMany, exists, GqlModelType, User } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { BulkDeleteDialog } from "components/dialogs/BulkDeleteDialog/BulkDeleteDialog";
import { useCallback, useState } from "react";
import { BulkObjectActionComplete } from "utils/actions/bulkObjectActions";
import { ListObject } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { ObjectsToDeleteConfirmLevel } from "./useDeleter";
import { useLazyFetch } from "./useLazyFetch";

export const useBulkDeleter = ({
    objectType,
    onBulkActionComplete,
    selectedData,
}: {
    objectType: GqlModelType;
    onBulkActionComplete: (action: BulkObjectActionComplete.Delete, objectsDeleted: ListObject[]) => unknown;
    selectedData: ListObject[];
}) => {

    const hasDeletingSupport = exists(DeleteType[objectType]);
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
    const closeDeleteDialog = useCallback(() => { setIsDeleteDialogOpen(false); }, []);

    const [deleteMany] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);

    const doBulkDelete = useCallback((confirmedForDeletion: ListObject[]) => {
        if (!selectedData || !selectedData.length || !objectType) {
            return;
        }
        fetchLazyWrapper<DeleteManyInput, Count>({
            fetch: deleteMany,
            inputs: { ids: confirmedForDeletion.map(c => c.id) as string[], objectType: objectType as unknown as DeleteType },
            successMessage: () => ({ messageKey: "ObjectsDeleted", messageVariables: { count: confirmedForDeletion.length } }),
            onSuccess: () => {
                onBulkActionComplete(BulkObjectActionComplete.Delete, confirmedForDeletion);
                setIsDeleteDialogOpen(false);
            },
        });
    }, [deleteMany, selectedData, objectType, onBulkActionComplete]);

    const handleBulkDelete = useCallback(() => {
        if (!selectedData || !selectedData.length || !objectType) {
            return;
        }
        // Find confirmation level for this object type
        let confirmationType = ObjectsToDeleteConfirmLevel[objectType as unknown as DeleteType];
        // Special case: Users with "isBot" set to true require minimal confirmation instead of full
        if (objectType === "User") {
            if (selectedData.every(o => (o as Partial<User>).isBot === true)) {
                confirmationType = "minimal";
            }
        }
        console.log();
        if (confirmationType === "none") {
            // Delete without confirmation
            doBulkDelete(selectedData);
            return;
        }
        if (confirmationType === "minimal") {
            // Show simple confirmation dialog
            PubSub.get().publish("alertDialog", {
                messageKey: "DeleteConfirmMultiple",
                messageVariables: { count: selectedData.length },
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
    }, [objectType, selectedData, doBulkDelete]);

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
