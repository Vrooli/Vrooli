import { exists, GqlModelType } from "@local/shared";
import { SessionContext } from "contexts/SessionContext";
import { Dispatch, SetStateAction, useCallback, useContext, useMemo, useState } from "react";
import { SetLocation } from "route";
import { BulkObjectAction, BulkObjectActionComplete, getAvailableBulkActions } from "utils/actions/bulkObjectActions";
import { ListObject } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { useBulkDeleter } from "./useBulkDeleter";

export type UseBulkObjectActionsProps = {
    objectType: ListObject["__typename"];
    selectedData: ListObject[];
    setAllData: Dispatch<SetStateAction<ListObject[]>>;
    setSelectedData: Dispatch<SetStateAction<ListObject[]>>;
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
    objectType: ListObject["__typename"];
    onBulkActionStart: (action: BulkObjectAction | `${BulkObjectAction}`) => unknown;
    onBulkActionComplete: (action: BulkObjectActionComplete | `${BulkObjectActionComplete}`, data: any) => unknown;
};

const callIfExists = (callback: (() => void) | null | undefined) => {
    if (!exists(callback)) {
        PubSub.get().publishSnack({ messageKey: "ActionNotSupported", severity: "Error" });
        return;
    }
    callback();
};

export const useBulkObjectActions = ({
    objectType,
    selectedData,
    setAllData,
    setSelectedData,
    setLocation,
}: UseBulkObjectActionsProps): UseBulkObjectActionsReturn => {
    const session = useContext(SessionContext);

    // Callback when an action is completed
    const onBulkActionComplete = useCallback((action: BulkObjectActionComplete | `${BulkObjectActionComplete}`, data: any) => {
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
                setAllData(all => all.filter(object => !(data as ListObject[]).find(deleted => deleted.id === object.id)));
                setSelectedData([]);
                break;
            }
            case BulkObjectActionComplete.Report:
                //TODO
                break;
        }
    }, [selectedData, setAllData, setSelectedData]);

    const {
        closeDeleteDialog,
        handleBulkDelete,
        isDeleteDialogOpen,
        BulkDeleteDialogComponent,
    } = useBulkDeleter({
        selectedData,
        objectType: objectType as GqlModelType,
        onBulkActionComplete,
    });

    // Determine which actions are available   
    const availableActions = useMemo(() => getAvailableBulkActions(selectedData, objectType, session), [selectedData, objectType, session]);

    // Dialog states
    const [isBookmarkDialogOpen, setIsBookmarkDialogOpen] = useState<boolean>(false);
    const [isProjectAddDialogOpen, setIsProjectAddDialogOpen] = useState<boolean>(false);
    const [isReportDialogOpen, setIsReportDialogOpen] = useState<boolean>(false);

    // Dialog openers/closers
    const openBookmarkDialog = useCallback(() => setIsBookmarkDialogOpen(true), [setIsBookmarkDialogOpen]);
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
        objectType,
        onBulkActionStart,
        onBulkActionComplete,
    };
};
