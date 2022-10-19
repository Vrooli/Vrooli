// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Box, CircularProgress, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { ResourceCard, ResourceListItemContextMenu } from 'components';
import { ResourceListHorizontalProps } from '../types';
import { useCallback, useMemo, useState } from 'react';
import { Resource } from 'types';
import { cardRoot } from 'components/cards/styles';
import { ResourceDialog } from 'components/dialogs';
import { updateArray } from 'utils';
import { resourceDeleteManyMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { resourceDeleteManyVariables, resourceDeleteMany_resourceDeleteMany } from 'graphql/generated/resourceDeleteMany';
import { AddIcon } from '@shared/icons';

export const ResourceListHorizontal = ({
    title = '📌 Resources',
    canEdit = true,
    handleUpdate,
    mutate = true,
    list,
    loading = false,
    session,
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

    const [deleteMutation] = useMutation(resourceDeleteManyMutation);
    const onDelete = useCallback((index: number) => {
        console.log('onDelete', index, list);
        if (!list) return;
        const resource = list.resources[index];
        if (mutate && resource.id) {
            mutationWrapper<resourceDeleteMany_resourceDeleteMany, resourceDeleteManyVariables>({
                mutation: deleteMutation,
                input: { ids: [resource.id] },
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
        console.log('openUpdateDialog', index);
        setEditingIndex(index);
        setIsDialogOpen(true)
    }, []);

    const dialog = useMemo(() => (
        list ? <ResourceDialog
            partialData={editingIndex >= 0 ? list.resources[editingIndex as number] : undefined}
            index={editingIndex}
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
        <Box>
            {/* Add resource dialog */}
            {dialog}
            {/* Right-click context menu */}
            <ResourceListItemContextMenu
                canEdit={canEdit}
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
            <Typography component="h2" variant="h5" textAlign="left">{title}</Typography>
            <Box
                sx={{
                    boxShadow: 12,
                    borderRadius: '16px',
                    background: palette.background.default,
                    border: `1px ${palette.text.primary}`,
                }}
            >
                <Stack direction="row" spacing={2} p={1} sx={{
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
                            key={`resource-card-${index}`}
                            index={index}
                            session={session}
                            data={c}
                            onContextMenu={openContext}
                            onEdit={openUpdateDialog}
                            onDelete={onDelete}
                            aria-owns={Boolean(selectedIndex) ? contextId : undefined}
                        />
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
                    {canEdit ? <Tooltip placement="top" title="Add resource">
                        <Box
                            onClick={openDialog}
                            aria-label="Add resource"
                            sx={{
                                ...cardRoot,
                                background: "#cad2e0",
                                width: '120px',
                                minWidth: '120px',
                                height: '120px',
                                minHeight: '120px',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                            }}
                        >
                            <AddIcon fill={palette.primary.main} width='50px' height='50px' />
                        </Box>
                    </Tooltip> : null}
                </Stack>
            </Box>
        </Box>
    )
}