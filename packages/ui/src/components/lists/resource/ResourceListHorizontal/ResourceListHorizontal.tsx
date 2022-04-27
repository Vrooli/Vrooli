// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Box, Stack, Tooltip, Typography } from '@mui/material';
import { ResourceCard, ResourceListItemContextMenu } from 'components';
import { ResourceListHorizontalProps } from '../types';
import { containerShadow } from 'styles';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { Resource } from 'types';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { cardRoot } from 'components/cards/styles';
import { ResourceDialog } from 'components/dialogs';
import { updateArray } from 'utils';
import { resourceDeleteManyMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/wrappers';

export const ResourceListHorizontal = ({
    title = '📌 Resources',
    canEdit = true,
    handleUpdate,
    mutate = true,
    list,
    session,
}: ResourceListHorizontalProps) => {

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
                resources: updateArray(list.resources, index, updatedResource),
            });
        }
    }, [handleUpdate, list]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation<any>(resourceDeleteManyMutation);
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
    }, [handleUpdate, list, mutate]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
    const contextId = useMemo(() => `resource-context-menu-${selectedIndex}`, [selectedIndex]);
    const openContext = useCallback((ev: MouseEvent<HTMLButtonElement>, index: number) => {
        setContextAnchor(ev.currentTarget);
        setSelectedIndex(index);
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelectedIndex(null);
    }, []);

    // Add/update resource dialog
    const [editingIndex, setEditingIndex] = useState<number>(-1);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true) }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1) }, []);
    const openUpdateDialog = useCallback((index: number) => { 
        console.log('open update dialog', index)
        setEditingIndex(index);
        setIsDialogOpen(true) 
    }, []);

    const dialog = useMemo(() => (
        list ? <ResourceDialog
            partialData={editingIndex >= 0 ? list.resources[editingIndex as number] as any : undefined}
            index={editingIndex}
            listId={list.id}
            open={isDialogOpen}
            onClose={closeDialog}
            onCreated={onAdd}
            onUpdated={onUpdate}
            mutate={mutate}
            session={session}
        /> : null
    ), [editingIndex, onAdd, onUpdate, isDialogOpen, list, editingIndex, mutate]);

    return (
        <Box>
            {/* Add resource dialog */}
            {dialog}
            {/* Right-click context menu */}
            <ResourceListItemContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                index={selectedIndex}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={onDelete}
                onEdit={() => openUpdateDialog(selectedIndex ?? 0)}
                onMove={() => { }}
            />
            <Typography component="h2" variant="h5" textAlign="left">{title}</Typography>
            <Box
                sx={{
                    ...containerShadow,
                    borderRadius: '16px',
                    background: (t) => t.palette.background.default,
                    border: (t) => `1px ${t.palette.text.primary}`,
                }}
            >
                <Stack direction="row" spacing={2} p={2} sx={{
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
                        <ResourceCard
                            canEdit={canEdit}
                            handleEdit={() => openUpdateDialog(index)}
                            handleDelete={onDelete}
                            key={`resource-card-${index}`}
                            index={index}
                            session={session}
                            data={c}
                            onRightClick={openContext}
                            aria-owns={Boolean(selectedIndex) ? contextId : undefined}
                        />
                    ))}
                    {/* Add resource button */}
                    {canEdit ? <Tooltip placement="top" title="Add resource">
                        <Box
                            onClick={openDialog}
                            aria-label="Add resource"
                            sx={{
                                ...cardRoot,
                                background: "#cad2e0",
                                width: '120px',
                                minWidth: '120px',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                            }}
                        >
                            <AddIcon color="primary" sx={{ width: '50px', height: '50px' }} />
                        </Box>
                    </Tooltip> : null}
                </Stack>
            </Box>
        </Box>
    )
}