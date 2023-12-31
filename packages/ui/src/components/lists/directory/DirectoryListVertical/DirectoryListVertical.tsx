import { Count, DeleteManyInput, DeleteType, ProjectVersionDirectory, endpointPostDeleteMany } from "@local/shared";
import { Box, Button } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { useBulkObjectActions } from "hooks/useBulkObjectActions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSelectableList } from "hooks/useSelectableList";
import { AddIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ArgsType } from "types";
import { ListObject } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { initializeDirectoryList } from "../common";
import { DirectoryItem, DirectoryListProps } from "../types";

export const DirectoryListVertical = ({
    canUpdate = true,
    directory,
    handleUpdate,
    loading,
    mutate,
    sortBy,
}: DirectoryListProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    const list = useMemo(() => initializeDirectoryList(directory, sortBy, session), [directory, session, sortBy]);

    // Add item dialog
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); }, []);

    const onAdd = useCallback((item: DirectoryItem) => {
        console.log("directoryyy onAdd", item);
        if (!directory) return;
        // Dont add duplicates
        if (list.some(r => r.id === item.id)) {
            PubSub.get().publish("snack", { message: "Item already added", severity: "Warning" });
            return;
        }
        if (handleUpdate) {
            handleUpdate({
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? [...directory.childApiVersions, item] : directory.childApiVersions as any[],
                childNoteVersions: item.__typename === "NoteVersion" ? [...directory.childNoteVersions, item] : directory.childNoteVersions as any[],
                childOrganizations: item.__typename === "Organization" ? [...directory.childOrganizations, item] : directory.childOrganizations as any[],
                childProjectVersions: item.__typename === "ProjectVersion" ? [...directory.childProjectVersions, item] : directory.childProjectVersions as any[],
                childRoutineVersions: item.__typename === "RoutineVersion" ? [...directory.childRoutineVersions, item] : directory.childRoutineVersions as any[],
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? [...directory.childSmartContractVersions, item] : directory.childSmartContractVersions as any[],
                childStandardVersions: item.__typename === "StandardVersion" ? [...directory.childStandardVersions, item] : directory.childStandardVersions as any[],
            });
        }
        closeDialog();
    }, [closeDialog, directory, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
    const onDelete = useCallback((index: number) => {
        if (!directory) return;
        const item = list[index];
        if (mutate) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { objects: [{ id: item.id, objectType: item.__typename as DeleteType }] },
                onSuccess: () => {
                    handleUpdate?.({
                        ...directory,
                        childApiVersions: item.__typename === "ApiVersion" ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                        childNoteVersions: item.__typename === "NoteVersion" ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                        childOrganizations: item.__typename === "Organization" ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations,
                        childProjectVersions: item.__typename === "ProjectVersion" ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                        childRoutineVersions: item.__typename === "RoutineVersion" ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                        childSmartContractVersions: item.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions,
                        childStandardVersions: item.__typename === "StandardVersion" ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
                    });
                },
            });
        }
        else {
            handleUpdate?.({
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                childNoteVersions: item.__typename === "NoteVersion" ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                childOrganizations: item.__typename === "Organization" ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations,
                childProjectVersions: item.__typename === "ProjectVersion" ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                childRoutineVersions: item.__typename === "RoutineVersion" ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions,
                childStandardVersions: item.__typename === "StandardVersion" ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
            });
        }
    }, [deleteMutation, directory, handleUpdate, list, mutate]);

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<DirectoryItem>();
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<DirectoryItem>({
        allData: list,
        selectedData,
        setAllData: (data) => {
            if (!directory) return;
            handleUpdate?.({
                ...directory,
                childApiVersions: data.filter(i => i.__typename === "ApiVersion"),
                childNoteVersions: data.filter(i => i.__typename === "NoteVersion"),
                childOrganizations: data.filter(i => i.__typename === "Organization"),
                childProjectVersions: data.filter(i => i.__typename === "ProjectVersion"),
                childRoutineVersions: data.filter(i => i.__typename === "RoutineVersion"),
                childSmartContractVersions: data.filter(i => i.__typename === "SmartContractVersion"),
                childStandardVersions: data.filter(i => i.__typename === "StandardVersion"),
            } as ProjectVersionDirectory);
        },
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });
    const onAction = useCallback((action: keyof ObjectListActions<DirectoryItem>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                const [id] = data as ArgsType<ObjectListActions<DirectoryItem>["Deleted"]>;
                const index = list.findIndex(r => r.id === id);
                onDelete(index);
                break;
            }
            case "Updated": {
                if (!directory) return;
                const [updatedItem] = data as ArgsType<ObjectListActions<DirectoryItem>["Updated"]>;
                handleUpdate?.({
                    ...directory,
                    childApiVersions: updatedItem.__typename === "ApiVersion" ? directory.childApiVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childApiVersions,
                    childNoteVersions: updatedItem.__typename === "NoteVersion" ? directory.childNoteVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childNoteVersions,
                    childOrganizations: updatedItem.__typename === "Organization" ? directory.childOrganizations.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childOrganizations,
                    childProjectVersions: updatedItem.__typename === "ProjectVersion" ? directory.childProjectVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childProjectVersions,
                    childRoutineVersions: updatedItem.__typename === "RoutineVersion" ? directory.childRoutineVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childRoutineVersions,
                    childSmartContractVersions: updatedItem.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childSmartContractVersions,
                    childStandardVersions: updatedItem.__typename === "StandardVersion" ? directory.childStandardVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childStandardVersions,
                } as ProjectVersionDirectory);
                break;
            }
        }
    }, [directory, handleUpdate, list, onDelete]);
    const onClick = useCallback((data: ListObject) => {
        //TODO
    }, []);

    return (
        <>
            {/* Add item dialog */}
            <FindObjectDialog
                find="List"
                isOpen={isDialogOpen}
                handleCancel={closeDialog}
                handleComplete={onAdd as any}
            />
            {/* Right-click context menu */}
            {/* <DirectoryListItemContextMenu
                canUpdate={canUpdate}
                id={contextId}
                anchorEl={contextAnchor}
                index={selectedIndex ?? -1}
                onClose={closeContext}
                onDelete={onDelete}
                data={selectedItem}
            /> */}
            <ObjectList
                canNavigate={() => !isEditing}
                handleToggleSelect={handleToggleSelect}
                hideUpdateButton={isEditing}
                isSelecting={isSelecting}
                items={list}
                keyPrefix="directory-list-item"
                loading={loading}
                onAction={onAction}
                onClick={onClick}
                selectedItems={selectedData}
            />
            {/* Add item button */}
            {canUpdate && <Box sx={{
                maxWidth: "400px",
                margin: "auto",
                paddingTop: 5,
            }}>
                <Button
                    fullWidth onClick={openDialog}
                    startIcon={<AddIcon />}
                    variant="outlined"
                >{t("AddItem")}</Button>
            </Box>}
        </>
    );
};
