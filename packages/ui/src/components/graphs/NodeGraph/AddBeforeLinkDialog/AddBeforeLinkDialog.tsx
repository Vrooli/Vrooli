/**
 * Prompts user to select which link the new node should be added on
 */
import { AddBeforeLinkDialogProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import { useCallback, useMemo } from 'react';
import { NodeLink } from 'types';
import { Dialog, DialogContent, List, ListItem, ListItemText } from '@mui/material';
import { getTranslation, getUserLanguages } from 'utils';
import { DialogTitle } from 'components/dialogs';

const titleAria = 'add-before-link-dialog-title';

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
            aria-labelledby={titleAria}
            sx={{
                zIndex,
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                onClose={handleClose}
                title="Select Link"
            />
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