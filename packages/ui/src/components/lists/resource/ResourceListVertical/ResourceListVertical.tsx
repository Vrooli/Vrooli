// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Count, DeleteManyInput, endpointPostDeleteMany, Resource } from "@local/shared";
import { Box, Button } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { ResourceDialog } from "components/dialogs/ResourceDialog/ResourceDialog";
import { AddIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { updateArray } from "utils/shape/general";
import { ResourceListItem } from "../ResourceListItem/ResourceListItem";
import { ResourceListItemContextMenu } from "../ResourceListItemContextMenu/ResourceListItemContextMenu";
import { ResourceListVerticalProps } from "../types";

export const ResourceListVertical = ({
    canUpdate = true,
    handleUpdate,
    mutate,
    list,
    loading,
    zIndex,
}: ResourceListVerticalProps) => {
    const { t } = useTranslation();

    const onAdd = useCallback((newResource: Resource) => {
        if (!list) return;
        if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: [...(list?.resources as any) ?? [], newResource],
            });
        }
    }, [handleUpdate, list]);

    const onUpdate = useCallback((index: number, updatedResource: Resource) => {
        if (!list) return;
        if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, index, updatedResource) as any[],
            });
        }
    }, [handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
    const onDelete = useCallback((index: number) => {
        if (!list) return;
        const resource = list.resources[index];
        if (mutate && resource.id) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { ids: [resource.id], objectType: "Resource" as any },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate({
                            ...list,
                            resources: list.resources.filter(r => r.id !== resource.id) as any,
                        });
                    }
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: list.resources.filter(r => r.id !== resource.id) as any,
            });
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
        setSelectedResource(resource as any);
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

    const dialog = useMemo(() => (
        list ? <ResourceDialog
            index={editingIndex}
            isOpen={isDialogOpen}
            partialData={(editingIndex >= 0) ? list.resources[editingIndex as number] as any : undefined}
            listId={list.id}
            onClose={closeDialog}
            onCreated={onAdd}
            onUpdated={onUpdate}
            mutate={mutate}
            zIndex={zIndex + 1}
        /> : null
    ), [list, editingIndex, isDialogOpen, closeDialog, onAdd, onUpdate, mutate, zIndex]);

    return (
        <>
            {/* Right-click context menu */}
            <ResourceListItemContextMenu
                canUpdate={canUpdate}
                id={contextId}
                anchorEl={contextAnchor}
                index={selectedIndex ?? -1}
                onClose={closeContext}
                onAddBefore={() => { }} //TODO
                onAddAfter={() => { }} //TODO
                onDelete={onDelete}
                onEdit={() => openUpdateDialog(selectedIndex ?? 0)}
                onMove={() => { }} //TODO
                resource={selectedResource}
                zIndex={zIndex + 1}
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
