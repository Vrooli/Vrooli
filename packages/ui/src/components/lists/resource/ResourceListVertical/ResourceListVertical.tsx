// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Count, DeleteManyInput, DeleteType, DUMMY_ID, endpointPostDeleteMany, Resource } from "@local/shared";
import { Box, Button } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { useBulkObjectActions } from "hooks/useBulkObjectActions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSelectableList } from "hooks/useSelectableList";
import { AddIcon } from "icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ListObject } from "utils/display/listTools";
import { updateArray } from "utils/shape/general";
import { resourceInitialValues, ResourceUpsert } from "views/objects/resource";
import { ResourceListVerticalProps } from "../types";

export const ResourceListVertical = ({
    canUpdate = true,
    handleUpdate,
    list,
    loading,
    mutate,
    parent,
}: ResourceListVerticalProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
    const onDelete = useCallback((index: number) => {
        if (!list) return;
        const resource = list.resources[index];
        const updatedList = {
            ...list,
            resources: list.resources.filter(r => r.id !== resource.id),
        };
        if (mutate && resource.id) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { objects: [{ id: resource.id, objectType: DeleteType.Resource }] },
                onSuccess: () => {
                    handleUpdate?.(updatedList);
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate(updatedList);
        }
    }, [deleteMutation, handleUpdate, list, mutate]);

    // Add/update resource dialog
    const [editingIndex, setEditingIndex] = useState<number>(-1);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((index: number) => {
        setEditingIndex(index);
        setIsDialogOpen(true);
    }, []);
    const onCompleted = useCallback((resource: Resource) => {
        closeDialog();
        if (!list || !handleUpdate) return;
        const index = resource.index;
        if (index && index >= 0) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, index, resource) as Resource[],
            });
        }
        else {
            handleUpdate({
                ...list,
                resources: [...(list?.resources ?? []), resource],
            });
        }
    }, [closeDialog, handleUpdate, list]);
    const onDeleted = useCallback((resource: Resource) => {
        closeDialog();
        if (!list || !handleUpdate) return;
        handleUpdate({
            ...list,
            resources: list.resources.filter(r => r.id !== resource.id) as any,
        });
    }, [closeDialog, handleUpdate, list]);

    const dialog = useMemo(() => {
        return <ResourceUpsert
            display="dialog"
            isCreate={editingIndex < 0}
            isOpen={isDialogOpen}
            isMutate={mutate}
            onCancel={closeDialog}
            onClose={closeDialog}
            onCompleted={onCompleted}
            onDeleted={onDeleted}
            overrideObject={(editingIndex >= 0 && list?.resources ?
                { ...list.resources[editingIndex as number], index: editingIndex } :
                resourceInitialValues(undefined, {
                    index: 0,
                    list: {
                        __connect: true,
                        ...(list?.id && list.id !== DUMMY_ID ? list : { listFor: parent }),
                        id: list?.id ?? DUMMY_ID,
                        __typename: "ResourceList",
                    },
                }) as Resource)}
        />;
    }, [closeDialog, editingIndex, isDialogOpen, list, mutate, onCompleted, onDeleted, parent]);

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<Resource>();
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<Resource>({
        allData: list?.resources ?? [],
        selectedData,
        setAllData: (data) => {
            //TODO
        },
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });
    const onAction = useCallback((action: keyof ObjectListActions<Resource>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                //TODO
                break;
            }
            case "Updated": {
                //TODO
                break;
            }
        }
    }, []);
    const onClick = useCallback((data: ListObject) => {
        //TODO
    }, []);

    return (
        <>
            {/* Right-click context menu */}
            {/* <ResourceListItemContextMenu
                canUpdate={canUpdate}
                id={contextId}
                anchorEl={contextAnchor}
                index={selectedIndex ?? -1}
                onClose={closeContext}
                onAddBefore={() => {
                    setEditingIndex(selectedIndex ?? 0);
                    openDialog();
                }}
                onAddAfter={() => {
                    setEditingIndex(selectedIndex ? selectedIndex + 1 : 0);
                    openDialog();
                }}
                onDelete={onDelete}
                onEdit={() => openUpdateDialog(selectedIndex ?? 0)}
                onMove={(index: number) => {
                    if (handleUpdate && list) {
                        handleUpdate({
                            ...list,
                            resources: updateArray(list.resources, selectedIndex ?? 0, list.resources[index]) as any[],
                        });
                    }
                }}
                resource={selectedResource}
            /> */}
            {/* Add resource dialog */}
            {dialog}
            <ObjectList
                canNavigate={() => !isEditing}
                dummyItems={new Array(5).fill("Resource")}
                handleToggleSelect={handleToggleSelect}
                hideUpdateButton={isEditing}
                isSelecting={isSelecting}
                items={list?.resources ?? []}
                keyPrefix="resource-list-item"
                loading={loading}
                onAction={onAction}
                onClick={onClick}
                selectedItems={selectedData}
            />
            {/* Add resource button */}
            {canUpdate && <Box sx={{
                maxWidth: "400px",
                margin: "auto",
                paddingTop: 5,
            }}>
                <Button
                    fullWidth onClick={openDialog}
                    startIcon={<AddIcon />}
                    variant="outlined"
                >{t("CreateResource")}</Button>
            </Box>}
        </>
    );
};
