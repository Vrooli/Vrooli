// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Box, CircularProgress, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { Count, DeleteManyInput, Resource } from '@shared/consts';
import { LinkIcon } from '@shared/icons';
import { deleteOneOrManyDeleteMany } from 'api/generated/endpoints/deleteOneOrMany_deleteMany';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { ResourceCard } from 'components/cards/ResourceCard/ResourceCard';
import { cardRoot } from 'components/cards/styles';
import { ResourceDialog } from 'components/dialogs/ResourceDialog/ResourceDialog';
import { useCallback, useMemo, useState } from 'react';
import { DragDropContext, Draggable, Droppable, DropResult } from 'react-beautiful-dnd';
import { updateArray } from 'utils/shape/general';
import { ResourceListItemContextMenu } from '../ResourceListItemContextMenu/ResourceListItemContextMenu';
import { ResourceListHorizontalProps } from '../types';

export const ResourceListHorizontal = ({
    title,
    canUpdate = true,
    handleUpdate,
    mutate = true,
    list,
    loading = false,
    zIndex,
}: ResourceListHorizontalProps) => {
    const { palette } = useTheme();

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

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!destination) return;
        if (source.index === destination.index) return;
        // Handle the reordering of the resources in the list
        if (handleUpdate && list) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, source.index, list.resources[destination.index]) as any[],
            });
        }
    }, [handleUpdate, list]);

    const [deleteMutation] = useCustomMutation<Count, DeleteManyInput>(deleteOneOrManyDeleteMany);
    const onDelete = useCallback((index: number) => {
        if (!list) return;
        const resource = list.resources[index];
        if (mutate && resource.id) {
            mutationWrapper<Count, DeleteManyInput>({
                mutation: deleteMutation,
                input: { ids: [resource.id], objectType: 'Resource' as any },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate({
                            ...list,
                            resources: list.resources.filter(r => r.id !== resource.id) as any,
                        });
                    }
                },
            })
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
    const openDialog = useCallback(() => { setIsDialogOpen(true) }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1) }, []);
    const openUpdateDialog = useCallback((index: number) => {
        setEditingIndex(index);
        setIsDialogOpen(true)
    }, []);

    const dialog = useMemo(() => (
        list ? <ResourceDialog
            partialData={editingIndex >= 0 ? list.resources[editingIndex as number] : undefined}
            index={editingIndex}
            isOpen={isDialogOpen}
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
            {/* Add resource dialog */}
            {dialog}
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
            {title && <Typography component="h2" variant="h5" textAlign="left">{title}</Typography>}
            <DragDropContext onDragEnd={onDragEnd}>
                <Droppable droppableId="resource-list" direction="horizontal">
                    {(provided) => (
                        <Stack
                            ref={provided.innerRef}
                            {...provided.droppableProps}
                            direction="row"
                            justifyContent="center"
                            alignItems="center"
                            spacing={2}
                            p={1}
                            sx={{
                                width: '100%',
                                maxWidth: '700px',
                                marginLeft: 'auto',
                                marginRight: 'auto',
                                // Custom scrollbar styling
                                overflowX: 'auto',
                                "&::-webkit-scrollbar": {
                                    width: 5,
                                },
                                "&::-webkit-scrollbar-track": {
                                    backgroundColor: 'transparent',
                                },
                                "&::-webkit-scrollbar-thumb": {
                                    borderRadius: '100px',
                                    backgroundColor: "#409590",
                                },
                            }}>
                            {/* Resources */}
                            {list?.resources?.map((c: Resource, index) => (
                                <Draggable key={`resource-card-${index}`} draggableId={`resource-card-${index}`} index={index}>
                                    {(provided) => (
                                        <ResourceCard
                                            ref={provided.innerRef}
                                            dragProps={provided.draggableProps}
                                            dragHandleProps={provided.dragHandleProps}
                                            canUpdate={canUpdate}
                                            key={`resource-card-${index}`}
                                            index={index}
                                            data={c}
                                            onContextMenu={openContext}
                                            onEdit={openUpdateDialog}
                                            onDelete={onDelete}
                                            aria-owns={Boolean(selectedIndex) ? contextId : undefined}
                                        />
                                    )}
                                </Draggable>
                            ))}
                            {
                                loading && (
                                    <CircularProgress sx={{
                                        position: 'absolute',
                                        top: '50%',
                                        left: '50%',
                                        transform: 'translate(-50%, -50%)',
                                        color: palette.mode === 'light' ? palette.secondary.light : 'white',
                                    }} />
                                )
                            }
                            {/* Add resource button */}
                            {canUpdate ? <Tooltip placement="top" title="Add resource">
                                <Box
                                    onClick={openDialog}
                                    aria-label="Add resource"
                                    sx={{
                                        ...cardRoot,
                                        background: palette.primary.light,
                                        width: '120px',
                                        minWidth: '120px',
                                        height: '120px',
                                        minHeight: '120px',
                                        display: 'flex',
                                        alignItems: 'center',
                                        justifyContent: 'center',
                                    }}
                                >
                                    <LinkIcon fill={palette.secondary.contrastText} width='56px' height='56px' />
                                </Box>
                            </Tooltip> : null}
                            {provided.placeholder}
                        </Stack>
                    )}
                </Droppable>
            </DragDropContext>
        </>
    )
}