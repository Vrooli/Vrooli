// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { ResourceDialog, ResourceListItem } from 'components';
import { ResourceListVerticalProps } from '../types';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { NewObject, Resource } from 'types';
import { containerShadow } from 'styles';
import { Box, Button } from '@mui/material';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { useMutation } from '@apollo/client';
import { resourceDeleteManyMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { updateArray } from 'utils';
import { resourceDeleteMany, resourceDeleteManyVariables } from 'graphql/generated/resourceDeleteMany';

export const ResourceListVertical = ({
    title = '📌 Resources',
    canEdit = true,
    handleUpdate,
    mutate,
    list,
    loading,
    session,
    zIndex,
}: ResourceListVerticalProps) => {

    const onAdd = useCallback((newResource: NewObject<Resource>) => {
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
                resources: updateArray(list.resources, index, updatedResource),
            });
        }
    }, [handleUpdate, list]);

    const [deleteMutation] = useMutation<resourceDeleteMany, resourceDeleteManyVariables>(resourceDeleteManyMutation);
    const onDelete = useCallback((index: number) => {
        if (!list) return;
        const resource = list.resources[index];
        if (mutate && resource.id) {
            mutationWrapper({
                mutation: deleteMutation,
                input: { ids: [resource.id] },
                onSuccess: (response) => {
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
    const [selected, setSelected] = useState<any | null>(null);
    const contextId = useMemo(() => `resource-context-menu-${selected?.link}`, [selected]);
    const openContext = useCallback((ev: MouseEvent<HTMLButtonElement>, data: any) => {
        setContextAnchor(ev.currentTarget);
        setSelected(data);
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelected(null);
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
            <Box sx={{
                ...containerShadow,
                overflow: 'overlay',
                borderRadius: '8px',
                maxWidth: '1000px',
                marginLeft: 'auto',
                marginRight: 'auto',
            }}>
                {/* Add resource dialog */}
                {dialog}
                {/* Resource list */}
                {list?.resources?.map((c: Resource, index) => (
                    <ResourceListItem
                        key={`resource-card-${index}`}
                        canEdit={canEdit}
                        data={c}
                        handleEdit={() => openUpdateDialog(index)}
                        handleDelete={onDelete}
                        index={index}
                        loading={loading}
                        session={session}
                    />
                ))}
            </Box>
            {/* Add resource button */}
            {canEdit && <Box sx={{
                maxWidth: '400px',
                margin: 'auto',
                paddingTop: 5,
            }}>
                <Button fullWidth onClick={openDialog} startIcon={<AddIcon />}>Add Resource</Button>
            </Box>}
        </>
    )
}