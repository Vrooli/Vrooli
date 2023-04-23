import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { AddIcon } from "@local/icons";
import { Box, Button } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { deleteOneOrManyDeleteOne } from "../../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { useCustomMutation } from "../../../../api/hooks";
import { mutationWrapper } from "../../../../api/utils";
import { updateArray } from "../../../../utils/shape/general";
import { ResourceDialog } from "../../../dialogs/ResourceDialog/ResourceDialog";
import { ResourceListItem } from "../ResourceListItem/ResourceListItem";
import { ResourceListItemContextMenu } from "../ResourceListItemContextMenu/ResourceListItemContextMenu";
export const ResourceListVertical = ({ canUpdate = true, handleUpdate, mutate, list, loading, zIndex, }) => {
    const onAdd = useCallback((newResource) => {
        if (!list)
            return;
        if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: [...list?.resources ?? [], newResource],
            });
        }
    }, [handleUpdate, list]);
    const onUpdate = useCallback((index, updatedResource) => {
        if (!list)
            return;
        if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, index, updatedResource),
            });
        }
    }, [handleUpdate, list]);
    const [deleteMutation] = useCustomMutation(deleteOneOrManyDeleteOne);
    const onDelete = useCallback((index) => {
        if (!list)
            return;
        const resource = list.resources[index];
        if (mutate && resource.id) {
            mutationWrapper({
                mutation: deleteMutation,
                input: { ids: [resource.id], objectType: "Resource" },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate({
                            ...list,
                            resources: list.resources.filter(r => r.id !== resource.id),
                        });
                    }
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: list.resources.filter(r => r.id !== resource.id),
            });
        }
    }, [deleteMutation, handleUpdate, list, mutate]);
    const [contextAnchor, setContextAnchor] = useState(null);
    const [selectedResource, setSelectedResource] = useState(null);
    const selectedIndex = useMemo(() => selectedResource ? list?.resources?.findIndex(r => r.id === selectedResource.id) : -1, [list, selectedResource]);
    const contextId = useMemo(() => `resource-context-menu-${selectedResource?.id}`, [selectedResource]);
    const openContext = useCallback((target, index) => {
        setContextAnchor(target);
        const resource = list?.resources[index];
        setSelectedResource(resource);
    }, [list?.resources]);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelectedResource(null);
    }, []);
    const [editingIndex, setEditingIndex] = useState(-1);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { console.log("open dialog"); setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((index) => {
        setEditingIndex(index);
        setIsDialogOpen(true);
    }, []);
    const dialog = useMemo(() => (list ? _jsx(ResourceDialog, { index: editingIndex, isOpen: isDialogOpen, partialData: (editingIndex >= 0) ? list.resources[editingIndex] : undefined, listId: list.id, onClose: closeDialog, onCreated: onAdd, onUpdated: onUpdate, mutate: mutate, zIndex: zIndex + 1 }) : null), [list, editingIndex, isDialogOpen, closeDialog, onAdd, onUpdate, mutate, zIndex]);
    return (_jsxs(_Fragment, { children: [_jsx(ResourceListItemContextMenu, { canUpdate: canUpdate, id: contextId, anchorEl: contextAnchor, index: selectedIndex ?? -1, onClose: closeContext, onAddBefore: () => { }, onAddAfter: () => { }, onDelete: onDelete, onEdit: () => openUpdateDialog(selectedIndex ?? 0), onMove: () => { }, resource: selectedResource, zIndex: zIndex + 1 }), dialog, list?.resources && list.resources.length > 0 && _jsx(Box, { sx: {
                    boxShadow: 12,
                    overflow: "overlay",
                    borderRadius: "8px",
                    maxWidth: "1000px",
                    marginLeft: "auto",
                    marginRight: "auto",
                }, children: list.resources.map((c, index) => (_jsx(ResourceListItem, { canUpdate: canUpdate, data: c, handleContextMenu: openContext, handleEdit: () => openUpdateDialog(index), handleDelete: onDelete, index: index, loading: loading }, `resource-card-${index}`))) }), canUpdate && _jsx(Box, { sx: {
                    maxWidth: "400px",
                    margin: "auto",
                    paddingTop: 5,
                }, children: _jsx(Button, { fullWidth: true, onClick: openDialog, startIcon: _jsx(AddIcon, {}), children: "Add Resource" }) })] }));
};
//# sourceMappingURL=ResourceListVertical.js.map