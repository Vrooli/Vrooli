/**
 * Prompts user to select which link the new node should be added on
 */
import { DialogContent, List, ListItem, ListItemText } from '@mui/material';
import { NodeLink } from '@shared/consts';
import { DialogTitle } from 'components/dialogs/DialogTitle/DialogTitle';
import { LargeDialog } from 'components/dialogs/LargeDialog/LargeDialog';
import { ListMenuItemData } from 'components/dialogs/types';
import { useCallback, useContext, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { getTranslation, getUserLanguages } from 'utils/display/translationTools';
import { SessionContext } from 'utils/SessionContext';
import { AddAfterLinkDialogProps } from '../types';

const titleId = 'add-after-link-dialog-title';

export const AddAfterLinkDialog = ({
    isOpen,
    handleClose,
    handleSelect,
    nodeId,
    nodes,
    links,
    zIndex,
}: AddAfterLinkDialogProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    /**
     * Gets the name of a node from its id
     */
    const getNodeName = useCallback((nodeId: string) => {
        const node = nodes.find(n => n.id === nodeId);
        return getTranslation(node, getUserLanguages(session), true).name;
    }, [nodes, session]);

    /**
     * Find links where the "from.id" is the nodeId
     */
    const linkOptions = useMemo<NodeLink[]>(() => links.filter(l => l.from.id === nodeId), [links, nodeId]);

    const listOptions: ListMenuItemData<NodeLink>[] = linkOptions.map(o => ({
        label: `${getNodeName(o.from.id)} ⟶ ${getNodeName(o.to.id)}`,
        value: o as NodeLink,
    }));

    return (
        <LargeDialog
            id="add-link-after-dialog"
            onClose={handleClose}
            isOpen={isOpen}
            titleId={titleId}
            zIndex={zIndex}
        >
            <DialogTitle
                id={titleId}
                onClose={handleClose}
                title={t('LinkSelect')}
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
        </LargeDialog>
    )
}