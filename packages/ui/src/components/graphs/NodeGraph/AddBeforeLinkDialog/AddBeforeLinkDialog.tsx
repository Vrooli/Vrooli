/**
 * Prompts user to select which link the new node should be added on
 */
import { AddBeforeLinkDialogProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import { useCallback, useMemo } from 'react';
import { NodeLink } from 'types';
import { Box, Dialog, IconButton, List, ListItem, ListItemText, Typography } from '@mui/material';
import {
    Close as CloseIcon,
} from '@mui/icons-material';
import { noSelect } from 'styles';
import { getTranslation, getUserLanguages } from 'utils';

export const AddBeforeLinkDialog = ({
    isOpen,
    handleClose,
    handleSelect,
    nodeId,
    nodes,
    links,
    session,
}: AddBeforeLinkDialogProps) => {

    /**
     * Gets the name of a node from its id
     */
    const getNodeName = useCallback((nodeId: string) => {
        const node = nodes.find(n => n.id === nodeId);
        return getTranslation(node, 'title', getUserLanguages(session), true);
    }, [nodes, nodeId, session]);

    /**
     * Find links where the "toId" is the nodeId
     */
    const linkOptions = useMemo<NodeLink[]>(() => links.filter(l => l.toId === nodeId), [links, nodeId]);

    const listOptions: ListMenuItemData<NodeLink>[] = linkOptions.map(o => ({
        label: `${getNodeName(o.toId)} ⟵ ${getNodeName(o.fromId)}`,
        value: o as NodeLink,
    }));

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
        >
            <Box
                sx={{
                    ...noSelect,
                    display: 'flex',
                    alignItems: 'center',
                    padding: 1,
                    background: (t) => t.palette.primary.dark
                }}
            >
                <Typography
                    variant="h6"
                    textAlign="center"
                    sx={{ width: '-webkit-fill-available', color: (t) => t.palette.primary.contrastText }}
                >
                    Select Link
                </Typography>
                <IconButton
                    edge="end"
                    onClick={handleClose}
                >
                    <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                </IconButton>
            </Box>
            <List>
                {listOptions.map(({ label, value }, index) => (
                    <ListItem button onClick={() => { handleSelect(value); handleClose(); }} key={index}>
                        <ListItemText primary={label} />
                    </ListItem>
                ))}
            </List>
        </Dialog>
    )
}