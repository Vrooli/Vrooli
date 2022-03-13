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
import { AddResourceDialog } from 'components/dialogs';
import { updateArray } from 'utils';

export const ResourceListHorizontal = ({
    title = '📌 Resources',
    canEdit = true,
    handleUpdate,
    mutate = true,
    list,
    session,
}: ResourceListHorizontalProps) => {

    const handleAdd = useCallback((newResource: Resource) => {
        if (!list) return;
        if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: [...(list?.resources as any) ?? [], newResource],
            });
        }
    }, [handleUpdate, list]);

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

    // Add resource dialog
    const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
    const openAddDialog = useCallback(() => { setIsAddDialogOpen(true) }, []);
    const closeAddDialog = useCallback(() => { setIsAddDialogOpen(false) }, []);

    const addDialog = useMemo(() => (
        list ? <AddResourceDialog
                listId={list.id}
                open={isAddDialogOpen}
                onClose={closeAddDialog}
                onCreated={handleAdd}
                mutate={mutate}
            /> : null
    ), [handleAdd, isAddDialogOpen, list]);

    return (
        <Box>
            {/* Add resource dialog */}
            {addDialog}
            {/* Right-click context menu */}
            <ResourceListItemContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                resource={selected}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={() => { }}
                onEdit={() => { }}
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
                            key={`resource-card-${index}`}
                            session={session}
                            data={c}
                            onRightClick={openContext}
                            aria-owns={Boolean(selected) ? contextId : undefined}
                        />
                    ))}
                    {/* Add resource button */}
                    {canEdit ? <Tooltip placement="top" title="Add resource">
                        <Box
                            onClick={openAddDialog}
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