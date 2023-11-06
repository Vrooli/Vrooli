import { exists, GqlModelType } from "@local/shared";
import { SessionContext } from "contexts/SessionContext";
import { useCallback, useContext, useMemo, useState } from "react";
import { SetLocation } from "route";
import { BulkObjectAction, BulkObjectActionComplete, getAvailableBulkActions } from "utils/actions/bulkObjectActions";
import { ListObject } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { useBulkDeleter } from "./useBulkDeleter";

export type UseBulkObjectActionsProps<T extends ListObject = ListObject> = {
    allData: T[];
    objectType: T["__typename"];
    selectedData: T[];
    setAllData: (data: T[]) => unknown;
    setSelectedData: (data: T[]) => unknown;
    setLocation: SetLocation;
};

export type UseBulkObjectActionsReturn<T extends ListObject = ListObject> = {
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
    objectType: T["__typename"];
    onBulkActionStart: (action: BulkObjectAction | `${BulkObjectAction}`) => unknown;
    onBulkActionComplete: (action: BulkObjectActionComplete | `${BulkObjectActionComplete}`, data: any) => unknown;
};

const callIfExists = (callback: (() => unknown) | null | undefined) => {
    if (!exists(callback)) {
        PubSub.get().publishSnack({ messageKey: "ActionNotSupported", severity: "Error" });
        return;
    }
    callback();
};

export const useBulkObjectActions = <T extends ListObject = ListObject>({
    allData,
    objectType,
    selectedData,
    setAllData,
    setSelectedData,
    setLocation,
}: UseBulkObjectActionsProps<T>): UseBulkObjectActionsReturn<T> => {
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
