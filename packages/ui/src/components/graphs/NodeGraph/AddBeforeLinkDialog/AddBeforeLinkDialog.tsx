/**
 * Prompts user to select which link the new node should be added on
 */
import { AddBeforeLinkDialogProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import { useCallback, useMemo } from 'react';
import { Dialog, DialogContent, List, ListItem, ListItemText } from '@mui/material';
import { getTranslation, getUserLanguages } from 'utils';
import { DialogTitle } from 'components/dialogs';
import { useTranslation } from 'react-i18next';
import { NodeLink } from '@shared/consts';

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
    const { t } = useTranslation();

    /**
     * Gets the name of a node from its id
     */
    const getNodeName = useCallback((nodeId: string) => {
        const node = nodes.find(n => n.id === nodeId);
        return getTranslation(node, getUserLanguages(session), true).name;
    }, [nodes, session]);

    /**
     * Find links where the "to.id" is the nodeId
     */
    const linkOptions = useMemo<NodeLink[]>(() => links.filter(l => l.to.id === nodeId), [links, nodeId]);

    const listOptions: ListMenuItemData<NodeLink>[] = linkOptions.map(o => ({
        label: `${getNodeName(o.from.id)} ⟶ ${getNodeName(o.to.id)}`,
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
                title={t('common:LinkSelect', { lng: getUserLanguages(session)[0] })}
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