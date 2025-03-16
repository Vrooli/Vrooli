/**
 * Prompts user to select which link the new node should be added on
 */
import { getTranslation, NodeLink } from "@local/shared";
import { Autocomplete, Box, DialogContent, Grid, List, ListItem, ListItemText, Stack, Typography, useTheme } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons.js";
import { DialogTitle } from "components/dialogs/DialogTitle/DialogTitle.js";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog.js";
import { ListMenuItemData } from "components/dialogs/types.js";
import { TextInput } from "components/inputs/TextInput/TextInput.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools.js";
import { SessionContext } from "../../../contexts.js";
import { AddAfterLinkDialogProps, AddBeforeLinkDialogProps, MoveNodeMenuProps } from "./types.js";

const addBeforeLinkDialogTitleId = "add-before-link-dialog-title";

export function AddBeforeLinkDialog({
    isOpen,
    handleClose,
    handleSelect,
    nodeId,
    nodes,
    links,
}: AddBeforeLinkDialogProps) {
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
            titleId={addBeforeLinkDialogTitleId}
        >
            <TopBar
                display="dialog"
                onClose={handleClose}
                title={t("LinkSelect")}
                titleId={addBeforeLinkDialogTitleId}
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
}

const addAfterLinkDialogTitleId = "add-after-link-dialog-title";

export function AddAfterLinkDialog({
    isOpen,
    handleClose,
    handleSelect,
    nodeId,
    nodes,
    links,
}: AddAfterLinkDialogProps) {
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
            titleId={addAfterLinkDialogTitleId}
        >
            <TopBar
                display="dialog"
                onClose={handleClose}
                title={t("LinkSelect")}
                titleId={addAfterLinkDialogTitleId}
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
}

const moveNodeDialogTitleId = "move-node-dialog-title";

export function MoveNodeMenu({
    handleClose,
    isOpen,
    language,
    node,
    routineVersion,
}: MoveNodeMenuProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [fromColumnIndex, setFromColumnIndex] = useState<number>(node?.columnIndex ?? 0);
    const [fromRowIndex, setFromRowIndex] = useState<number>(node?.rowIndex ?? 0);
    const [toColumnIndex, setToColumnIndex] = useState<number>(node?.columnIndex ?? 0);
    const [toRowIndex, setToRowIndex] = useState<number>(node?.rowIndex ?? 0);

    const handleToColumnSelect = useCallback((columnIndex: number | null) => {
        setFromColumnIndex(columnIndex ?? 0);
    }, [setFromColumnIndex]);
    const handleToRowSelect = useCallback((rowIndex: number | null) => {
        setFromRowIndex(rowIndex ?? 0);
    }, [setFromRowIndex]);

    useEffect(() => {
        if (node) {
            const { columnIndex, rowIndex } = node;
            setFromColumnIndex(columnIndex ?? 0);
            setToColumnIndex((columnIndex ?? 0) + 1);
            setFromRowIndex(rowIndex ?? 0);
            setToRowIndex(rowIndex ?? 0);
        }
    }, [node]);

    const availableColumns = useMemo(() => {
        const lowestColumn = 1; // Can't put in first column
        const highestColumn = routineVersion.nodes.reduce((highest, node) => Math.max(highest, (node.columnIndex ?? 0)), 0);
        return new Array(highestColumn - lowestColumn + 1).fill(0).map((_, index) => index + lowestColumn);
    }, [routineVersion.nodes]);

    const availableRows = useMemo(() => {
        const nodesInColumn = routineVersion.nodes.filter(node => node.columnIndex === fromColumnIndex);
        const highestRowNumber = nodesInColumn.reduce((highest, node) => Math.max(highest, (node.rowIndex ?? 0)), 0);
        // Find all available numbers between 0 and highestRowNumber + 1
        const availableRows = new Array(highestRowNumber + 2)
            .fill(0).map((_, index) => index)
            .filter(rowIndex => nodesInColumn.every(node => node.rowIndex !== rowIndex));
        return availableRows;
    }, [fromColumnIndex, routineVersion.nodes]);

    // Update to row index when available rows change
    useEffect(() => {
        if (availableRows.length > 0) {
            setToRowIndex(availableRows[0]);
        } else {
            setToRowIndex(0);
        }
    }, [availableRows, setToRowIndex]);

    /**
     * Before closing, clear inputs
     */
    const onClose = useCallback((newPosition?: { columnIndex: number, rowIndex: number }) => {
        setFromColumnIndex(0);
        setToColumnIndex(0);
        setFromRowIndex(0);
        setToRowIndex(0);
        handleClose(newPosition);
    }, [handleClose]);

    const moveNode = useCallback(() => {
        if (toColumnIndex === fromColumnIndex && toRowIndex === fromRowIndex) {
            onClose();
        }
        onClose({ columnIndex: toColumnIndex, rowIndex: toRowIndex });
    }, [toColumnIndex, fromColumnIndex, toRowIndex, fromRowIndex, onClose]);

    const closeDialog = useCallback(() => { onClose(); }, [onClose]);

    /**
     * Container that displays "From" and "To" sections, with right arrow inbetween
     */
    const formContent = useMemo(() => (
        <Stack direction="row" justifyContent="center" alignItems="center" sx={{ color: palette.background.textPrimary }}>
            {/* "From" stack */}
            <Stack direction="column" spacing={2} justifyContent="center" alignItems="center">
                <Typography variant="h6">
                    {t("From")}
                </Typography>
                {/* Column TextInput (Disabled) */}
                <TextInput
                    fullWidth
                    disabled
                    id="node-from-column"
                    label="Column"
                    value={fromColumnIndex + 1}
                />
                {/* Row TextInput (Disabled) */}
                <TextInput
                    fullWidth
                    disabled
                    id="node-from-row"
                    label="Row"
                    value={fromRowIndex + 1}
                />
            </Stack>
            {/* Right arrow */}
            <Box sx={{
                width: "3em",
                height: "3em",
                color: "black",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
            }}>
                <Typography variant="h6" textAlign="center" color={palette.background.textPrimary}>
                    ⮕
                </Typography>
            </Box>
            {/* "To" stack */}
            <Stack direction="column" spacing={2} justifyContent="center" alignItems="center">
                <Typography variant="h6">
                    {t("To")}
                </Typography>
                {/* Column selector */}
                <Autocomplete
                    disablePortal
                    id="node-to-column"
                    options={availableColumns}
                    getOptionLabel={(option: number) => (option + 1) + ""}
                    onChange={(_, value) => handleToColumnSelect(value)}
                    value={toColumnIndex}
                    sx={{
                        minWidth: 200,
                        maxWidth: 350,
                    }}
                    renderInput={(params) => <TextInput {...params} label="Column" />}
                />
                {/* Row selector */}
                <Autocomplete
                    disablePortal
                    id="node-to-row"
                    options={availableRows}
                    getOptionLabel={(option: number) => (option + 1) + ""}
                    onChange={(_, value) => handleToRowSelect(value)}
                    value={toRowIndex}
                    sx={{
                        minWidth: 200,
                        maxWidth: 350,
                    }}
                    renderInput={(params) => <TextInput {...params} label="Row" />}
                />
            </Stack>
        </Stack>
    ), [palette.background.textPrimary, t, fromColumnIndex, fromRowIndex, availableColumns, toColumnIndex, availableRows, toRowIndex, handleToColumnSelect, handleToRowSelect]);

    return (
        <LargeDialog
            id="move-node-dialog"
            isOpen={isOpen}
            onClose={() => { handleClose(); }}
            titleId={moveNodeDialogTitleId}
        >
            <DialogTitle
                id={moveNodeDialogTitleId}
                help={t("NodeMoveDialogHelp")}
                onClose={onClose}
                title={t("NodeMove")}
            />
            <DialogContent>
                {formContent}
            </DialogContent>
            {/* Action buttons */}
            <Grid container spacing={1} sx={{ padding: 0, paddingTop: "24px" }}>
                <BottomActionsButtons
                    display="dialog"
                    isCreate={false}
                    onCancel={closeDialog}
                    onSubmit={moveNode}
                />
            </Grid>
        </LargeDialog>
    );
}
