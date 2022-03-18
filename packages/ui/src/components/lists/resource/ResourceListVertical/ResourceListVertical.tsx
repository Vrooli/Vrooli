// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { ResourceDialog, ResourceListItem } from 'components';
import { ResourceListVerticalProps } from '../types';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { Resource } from 'types';
import { containerShadow } from 'styles';
import { Box, Button, Tooltip } from '@mui/material';
import { cardRoot } from 'components/cards/styles';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { resourceDeleteMany, resourceDeleteManyVariables } from 'graphql/generated/resourceDeleteMany';
import { useMutation } from '@apollo/client';
import { resourceDeleteManyMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { updateArray } from 'utils';

export const ResourceListVertical = ({
    title = '📌 Resources',
    canEdit = true,
    handleUpdate,
    mutate,
    list,
    session,
}: ResourceListVerticalProps) => {

    console.log('list data', list)

    const onAdd = useCallback((newResource: Resource) => {
        console.log('HANDE ADD', newResource, list)
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
                resources: updateArray(list.resources, updatedResource.index, updatedResource),
            });
        }
    }, [handleUpdate, list]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation<any>(resourceDeleteManyMutation);
    const onDelete = useCallback((resource: Resource) => {
        if (!list) return;
        if (mutate) {
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
    }, [handleUpdate, list, mutate]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selected, setSelected] = useState<any | null>(null);
    const contextId = useMemo(() => `resource-context-menu-${selected?.link}`, [selected]);
    const openContext = useCallback((ev: MouseEvent<HTMLButtonElement>, data: any) => {
        console.log('setting context anchor', ev.currentTarget, data);
        setContextAnchor(ev.currentTarget);
        setSelected(data);
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelected(null);
    }, []);

    // Add/update resource dialog
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true) }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false) }, []);
    const [editingIndex, setEditingIndex] = useState<number | undefined>(undefined);
    const openUpdateDialog = useCallback((index: number) => { 
        setEditingIndex(index);
        setIsDialogOpen(true) 
    }, []);
    const closeUpdateDialog = useCallback(() => { 
        setEditingIndex(undefined);
        setIsDialogOpen(false) 
    }, []);

    const dialog = useMemo(() => (
        list ? <ResourceDialog
            isAdd={editingIndex !== undefined}
            partialData={editingIndex ? list.resources[editingIndex as number] as any : undefined}
            listId={list.id}
            open={isDialogOpen}
            onClose={closeDialog}
            onCreated={onAdd}
            onUpdated={onUpdate}
            mutate={mutate}
        /> : null
    ), [onAdd, onUpdate, isDialogOpen, list, editingIndex, mutate]);

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
                        data={c}
                        index={index}
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