/**
 * Prompts user to select which link the new node should be added on
 */
import { AddAfterLinkDialogProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import { useCallback, useMemo } from 'react';
import { NodeLink } from 'types';
import { Dialog, DialogContent, List, ListItem, ListItemText } from '@mui/material';
import { getTranslation, getUserLanguages } from 'utils';
import { DialogTitle } from 'components/dialogs';

const titleAria = 'add-after-link-dialog-title';

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

    /**
     * Gets the name of a node from its id
     */
    const getNodeName = useCallback((nodeId: string) => {
        const node = nodes.find(n => n.id === nodeId);
        return getTranslation(node, getUserLanguages(session), true).title;
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
        </Dialog >
    )
}