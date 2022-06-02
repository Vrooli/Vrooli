/**
 * Prompts user to select which link the new node should be added on
 */
import { AddAfterLinkDialogProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import { useCallback, useMemo } from 'react';
import { NodeLink } from 'types';
import { Box, Dialog, IconButton, List, ListItem, ListItemText, Typography, useTheme } from '@mui/material';
import {
    Close as CloseIcon,
} from '@mui/icons-material';
import { noSelect } from 'styles';
import { getTranslation, getUserLanguages } from 'utils';

export const AddAfterLinkDialog = ({
    isOpen,
    handleClose,
    handleSelect,
    nodeId,
    nodes,
    links,
    session,
    zIndex,
}: AddAfterLinkDialogProps) => {
    const { palette } = useTheme();

    /**
     * Gets the name of a node from its id
     */
    const getNodeName = useCallback((nodeId: string) => {
        const node = nodes.find(n => n.id === nodeId);
        return getTranslation(node, 'title', getUserLanguages(session), true);
    }, [nodes, session]);

    /**
     * Find links where the "fromId" is the nodeId
     */
    const linkOptions = useMemo<NodeLink[]>(() => links.filter(l => l.fromId === nodeId), [links, nodeId]);

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
                    display: 'flex',
                    alignItems: 'center',
                    padding: 1,
                    background: palette.primary.dark
                }}
            >
                <Typography
                    variant="h6"
                    textAlign="center"
                    sx={{ width: '-webkit-fill-available', color: palette.primary.contrastText }}
                >
                    Select Link
                </Typography>
                <IconButton
                    edge="end"
                    onClick={handleClose}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
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