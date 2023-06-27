/**
 * Prompts user to select which link the new node should be added on
 */
import { NodeLink } from "@local/shared";
import { DialogContent, List, ListItem, ListItemText } from "@mui/material";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { ListMenuItemData } from "components/dialogs/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { SessionContext } from "utils/SessionContext";
import { AddBeforeLinkDialogProps } from "../types";

const titleId = "add-before-link-dialog-title";

export const AddBeforeLinkDialog = ({
    isOpen,
    handleClose,
    handleSelect,
    nodeId,
    nodes,
    links,
    zIndex,
}: AddBeforeLinkDialogProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

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
        <LargeDialog
            id="add-link-before-dialog"
            onClose={handleClose}
            isOpen={isOpen}
            titleId={titleId}
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={handleClose}
                title={t("LinkSelect")}
                titleId={titleId}
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
    );
};
