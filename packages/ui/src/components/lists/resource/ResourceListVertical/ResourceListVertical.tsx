// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Count, DeleteManyInput, DUMMY_ID, endpointPostDeleteMany, Resource } from "@local/shared";
import { Box, Button } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { updateArray } from "utils/shape/general";
import { NewResourceShape, resourceInitialValues, ResourceUpsert } from "views/objects/resource";
import { ResourceListItem } from "../ResourceListItem/ResourceListItem";
import { ResourceListItemContextMenu } from "../ResourceListItemContextMenu/ResourceListItemContextMenu";
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
    console.log("resourcelistvertical", list);

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
                inputs: { ids: [resource.id], objectType: "Resource" as any },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate(updatedList);
                    }
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate(updatedList);
        }
    }, [deleteMutation, handleUpdate, list, mutate]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selectedResource, setSelectedResource] = useState<Resource | null>(null);
    const selectedIndex = useMemo(() => selectedResource ? list?.resources?.findIndex(r => r.id === selectedResource.id) : -1, [list, selectedResource]);
    const contextId = useMemo(() => `resource-context-menu-${selectedResource?.id}`, [selectedResource]);
    const openContext = useCallback((target: EventTarget, index: number) => {
        setContextAnchor(target);
        const resource = list?.resources[index];
        setSelectedResource(resource ?? null);
    }, [list?.resources]);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelectedResource(null);
    }, []);

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

    const dialog = useMemo(() => {
        return <ResourceUpsert
            isCreate={editingIndex < 0}
            isOpen={isDialogOpen}
            isMutate={mutate}
            onCancel={closeDialog}
            onCompleted={onCompleted}
            overrideObject={editingIndex >= 0 && list?.resources ?
                { ...list.resources[editingIndex as number], index: editingIndex } as NewResourceShape :
                resourceInitialValues(undefined, {
                    index: 0,
                    list: list?.id && list.id !== DUMMY_ID ? { id: list.id } : { listFor: parent.__typename, listForId: parent.id },
                }) as NewResourceShape}
        />;
    }, [closeDialog, editingIndex, isDialogOpen, list, mutate, onCompleted, parent.__typename, parent.id]);

    return (
        <>
            {/* Right-click context menu */}
            <ResourceListItemContextMenu
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
            />
            {/* Add resource dialog */}
            {dialog}
            {list?.resources && list.resources.length > 0 && <Box sx={{
                boxShadow: 12,
                overflow: "overlay",
                borderRadius: "8px",
                maxWidth: "1000px",
                marginLeft: "auto",
                marginRight: "auto",
            }}>
                {/* Resource list */}
                {list.resources.map((c: Resource, index) => (
                    <ResourceListItem
                        key={`resource-card-${index}`}
                        canUpdate={canUpdate}
                        data={c}
                        handleContextMenu={openContext}
                        handleEdit={() => openUpdateDialog(index)}
                        handleDelete={onDelete}
                        index={index}
                        loading={loading}
                    />
                ))}
            </Box>}
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
