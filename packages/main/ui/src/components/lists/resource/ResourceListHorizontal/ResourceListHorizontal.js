import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LinkIcon } from "@local/icons";
import { Box, CircularProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { DragDropContext, Draggable, Droppable } from "react-beautiful-dnd";
import { deleteOneOrManyDeleteMany } from "../../../../api/generated/endpoints/deleteOneOrMany_deleteMany";
import { useCustomMutation } from "../../../../api/hooks";
import { mutationWrapper } from "../../../../api/utils";
import { updateArray } from "../../../../utils/shape/general";
import { ResourceDialog } from "../../../dialogs/ResourceDialog/ResourceDialog";
import { cardRoot } from "../../styles";
import { ResourceCard } from "../ResourceCard/ResourceCard";
import { ResourceListItemContextMenu } from "../ResourceListItemContextMenu/ResourceListItemContextMenu";
export const ResourceListHorizontal = ({ title, canUpdate = true, handleUpdate, mutate = true, list, loading = false, zIndex, }) => {
    const { palette } = useTheme();
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
    const onDragEnd = useCallback((result) => {
        const { source, destination } = result;
        if (!destination)
            return;
        if (source.index === destination.index)
            return;
        if (handleUpdate && list) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, source.index, list.resources[destination.index]),
            });
        }
    }, [handleUpdate, list]);
    const [deleteMutation] = useCustomMutation(deleteOneOrManyDeleteMany);
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
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((index) => {
        setEditingIndex(index);
        setIsDialogOpen(true);
    }, []);
    const dialog = useMemo(() => (list ? _jsx(ResourceDialog, { partialData: editingIndex >= 0 ? list.resources[editingIndex] : undefined, index: editingIndex, isOpen: isDialogOpen, listId: list.id, onClose: closeDialog, onCreated: onAdd, onUpdated: onUpdate, mutate: mutate, zIndex: zIndex + 1 }) : null), [list, editingIndex, isDialogOpen, closeDialog, onAdd, onUpdate, mutate, zIndex]);
    return (_jsxs(_Fragment, { children: [dialog, _jsx(ResourceListItemContextMenu, { canUpdate: canUpdate, id: contextId, anchorEl: contextAnchor, index: selectedIndex ?? -1, onClose: closeContext, onAddBefore: () => {
                    setEditingIndex(selectedIndex ?? 0);
                    openDialog();
                }, onAddAfter: () => {
                    setEditingIndex(selectedIndex ? selectedIndex + 1 : 0);
                    openDialog();
                }, onDelete: onDelete, onEdit: () => openUpdateDialog(selectedIndex ?? 0), onMove: (index) => {
                    if (handleUpdate && list) {
                        handleUpdate({
                            ...list,
                            resources: updateArray(list.resources, selectedIndex ?? 0, list.resources[index]),
                        });
                    }
                }, resource: selectedResource, zIndex: zIndex + 1 }), title && _jsx(Typography, { component: "h2", variant: "h5", textAlign: "left", children: title }), _jsx(DragDropContext, { onDragEnd: onDragEnd, children: _jsx(Droppable, { droppableId: "resource-list", direction: "horizontal", children: (provided) => (_jsxs(Stack, { ref: provided.innerRef, ...provided.droppableProps, direction: "row", justifyContent: "center", alignItems: "center", spacing: 2, p: 1, sx: {
                            width: "100%",
                            maxWidth: "700px",
                            marginLeft: "auto",
                            marginRight: "auto",
                            overflowX: "auto",
                            "&::-webkit-scrollbar": {
                                width: 5,
                            },
                            "&::-webkit-scrollbar-track": {
                                backgroundColor: "transparent",
                            },
                            "&::-webkit-scrollbar-thumb": {
                                borderRadius: "100px",
                                backgroundColor: "#409590",
                            },
                        }, children: [list?.resources?.map((c, index) => (_jsx(Draggable, { draggableId: `resource-card-${index}`, index: index, children: (provided) => (_jsx(ResourceCard, { ref: provided.innerRef, dragProps: provided.draggableProps, dragHandleProps: provided.dragHandleProps, canUpdate: canUpdate, index: index, data: c, onContextMenu: openContext, onEdit: openUpdateDialog, onDelete: onDelete, "aria-owns": Boolean(selectedIndex) ? contextId : undefined }, `resource-card-${index}`)) }, `resource-card-${index}`))), loading && (_jsx(CircularProgress, { sx: {
                                    position: "absolute",
                                    top: "50%",
                                    left: "50%",
                                    transform: "translate(-50%, -50%)",
                                    color: palette.mode === "light" ? palette.secondary.light : "white",
                                } })), canUpdate ? _jsx(Tooltip, { placement: "top", title: "Add resource", children: _jsx(Box, { onClick: openDialog, "aria-label": "Add resource", sx: {
                                        ...cardRoot,
                                        background: palette.primary.light,
                                        width: "120px",
                                        minWidth: "120px",
                                        height: "120px",
                                        minHeight: "120px",
                                        display: "flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                    }, children: _jsx(LinkIcon, { fill: palette.secondary.contrastText, width: '56px', height: '56px' }) }) }) : null, provided.placeholder] })) }) })] }));
};
//# sourceMappingURL=ResourceListHorizontal.js.map