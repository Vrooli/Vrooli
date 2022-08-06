/**
 * Prompts user to select which link the new node should be added on
 */
import { AddBeforeLinkDialogProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import { useCallback, useMemo } from 'react';
import { NodeLink } from 'types';
import { Box, Dialog, DialogContent, IconButton, List, ListItem, ListItemText, Typography, useTheme } from '@mui/material';
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
    zIndex,
}: AddBeforeLinkDialogProps) => {
    const { palette } = useTheme();

    /**
     * Gets the name of a node from its id
     */
    const getNodeName = useCallback((nodeId: string) => {
        const node = nodes.find(n => n.id === nodeId);
        return getTranslation(node, 'title', getUserLanguages(session), true);
    }, [nodes, session]);

    /**
     * Find links where the "toId" is the nodeId
     */
    const linkOptions = useMemo<NodeLink[]>(() => links.filter(l => l.toId === nodeId), [links, nodeId]);

    const listOptions: ListMenuItemData<NodeLink>[] = linkOptions.map(o => ({
        label: `${getNodeName(o.fromId)} ⟶ ${getNodeName(o.toId)}`,
        value: o as NodeLink,
    }));

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
            sx={{
                zIndex,
            }}
        >
            <Box
                sx={{
                    ...noSelect,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    display: 'flex',
                    alignItems: 'left',
                    justifyContent: 'space-between',
                    padding: 1,
                }}
            >
                <Typography
                    variant="h6"
                    alignSelf='center'
                    sx={{ marginLeft: 2, marginRight: 'auto' }}
                >
                    Select Link
                </Typography>
                <Box sx={{ marginLeft: 'auto' }}>
                    <IconButton
                        edge="start"
                        onClick={handleClose}
                    >
                        <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                    </IconButton>
                </Box>
            </Box>
            <DialogContent>
                <List>
                    {listOptions.map(({ label, value }, index) => (
                        <ListItem button onClick={() => { handleSelect(value); handleClose(); }} key={index}>
                            <ListItemText primary={label} />
                        </ListItem>
                    ))}
                </List>
            </DialogContent>
        </Dialog>
    )
}