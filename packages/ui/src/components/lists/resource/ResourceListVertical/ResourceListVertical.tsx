// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { AddResourceDialog, ResourceListItem } from 'components';
import { ResourceListVerticalProps } from '../types';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { Resource } from 'types';
import { containerShadow } from 'styles';
import { Box, Button, Tooltip } from '@mui/material';
import { cardRoot } from 'components/cards/styles';
import {
    Add as AddIcon,
} from '@mui/icons-material';

export const ResourceListVertical = ({
    title = '📌 Resources',
    canEdit = true,
    handleUpdate,
    mutate,
    list,
    session,
}: ResourceListVerticalProps) => {

    console.log('list data', list)

    const handleAdd = useCallback((newResource: Resource) => {
        console.log('HANDE ADD', newResource, list)
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
                {addDialog}
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
                <Button fullWidth onClick={openAddDialog} startIcon={<AddIcon />}>Add Resource</Button>
            </Box>}
        </>
    )
}