// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { ResourceDialog, ResourceListItem, ResourceListItemContextMenu } from 'components';
import { ResourceListVerticalProps } from '../types';
import { useCallback, useMemo, useState } from 'react';
import { Box, Button } from '@mui/material';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { updateArray } from 'utils';
import { AddIcon } from '@shared/icons';
import { Count, DeleteManyInput, Resource } from '@shared/consts';
import { deleteOneOrManyDeleteOne } from 'api/generated/endpoints/deleteOneOrMany';

export const ResourceListVertical = ({
    title = '📌 Resources',
    canUpdate = true,
    handleUpdate,
    mutate,
    list,
    loading,
    session,
    zIndex,
}: ResourceListVerticalProps) => {

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

    const [deleteMutation] = useCustomMutation<Count, DeleteManyInput>(deleteOneOrManyDeleteOne);
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
    const openDialog = useCallback(() => { console.log('open dialog'); setIsDialogOpen(true) }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1) }, []);
    const openUpdateDialog = useCallback((index: number) => {
        setEditingIndex(index);
        setIsDialogOpen(true)
    }, []);

    const dialog = useMemo(() => (
        list ? <ResourceDialog
            index={editingIndex}
            partialData={(editingIndex >= 0) ? list.resources[editingIndex as number] as any : undefined}
            listId={list.id}
            open={isDialogOpen}
            onClose={closeDialog}
            onCreated={onAdd}
            onUpdated={onUpdate}
            mutate={mutate}
            session={session}
            zIndex={zIndex + 1}
        /> : null
    ), [list, editingIndex, isDialogOpen, closeDialog, onAdd, onUpdate, mutate, session, zIndex]);

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
                overflow: 'overlay',
                borderRadius: '8px',
                maxWidth: '1000px',
                marginLeft: 'auto',
                marginRight: 'auto',
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
                        session={session}
                    />
                ))}
            </Box>}
            {/* Add resource button */}
            {canUpdate && <Box sx={{
                maxWidth: '400px',
                margin: 'auto',
                paddingTop: 5,
            }}>
                <Button fullWidth onClick={openDialog} startIcon={<AddIcon />}>Add Resource</Button>
            </Box>}
        </>
    )
}