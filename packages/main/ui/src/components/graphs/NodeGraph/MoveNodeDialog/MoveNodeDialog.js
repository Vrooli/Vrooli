import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Autocomplete, Box, DialogContent, Grid, Stack, TextField, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../../buttons/GridSubmitButtons/GridSubmitButtons";
import { DialogTitle } from "../../../dialogs/DialogTitle/DialogTitle";
import { LargeDialog } from "../../../dialogs/LargeDialog/LargeDialog";
const titleId = "move-node-dialog-title";
export const MoveNodeMenu = ({ handleClose, isOpen, language, node, routineVersion, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [fromColumnIndex, setFromColumnIndex] = useState(node?.columnIndex ?? 0);
    const [fromRowIndex, setFromRowIndex] = useState(node?.rowIndex ?? 0);
    const [toColumnIndex, setToColumnIndex] = useState(node?.columnIndex ?? 0);
    const [toRowIndex, setToRowIndex] = useState(node?.rowIndex ?? 0);
    const handleToColumnSelect = useCallback((columnIndex) => {
        setFromColumnIndex(columnIndex ?? 0);
    }, [setFromColumnIndex]);
    const handleToRowSelect = useCallback((rowIndex) => {
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
        const lowestColumn = 1;
        const highestColumn = routineVersion.nodes.reduce((highest, node) => Math.max(highest, (node.columnIndex ?? 0)), 0);
        return new Array(highestColumn - lowestColumn + 1).fill(0).map((_, index) => index + lowestColumn);
    }, [routineVersion.nodes]);
    const availableRows = useMemo(() => {
        const nodesInColumn = routineVersion.nodes.filter(node => node.columnIndex === fromColumnIndex);
        const highestRowNumber = nodesInColumn.reduce((highest, node) => Math.max(highest, (node.rowIndex ?? 0)), 0);
        const availableRows = new Array(highestRowNumber + 2)
            .fill(0).map((_, index) => index)
            .filter(rowIndex => nodesInColumn.every(node => node.rowIndex !== rowIndex));
        return availableRows;
    }, [fromColumnIndex, routineVersion.nodes]);
    useEffect(() => {
        if (availableRows.length > 0) {
            setToRowIndex(availableRows[0]);
        }
        else {
            setToRowIndex(0);
        }
    }, [availableRows, setToRowIndex]);
    const onClose = useCallback((newPosition) => {
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
    const formContent = useMemo(() => (_jsxs(Stack, { direction: "row", justifyContent: "center", alignItems: "center", sx: { color: palette.background.textPrimary }, children: [_jsxs(Stack, { direction: "column", spacing: 2, justifyContent: "center", alignItems: "center", children: [_jsx(Typography, { variant: "h6", children: t("From") }), _jsx(TextField, { fullWidth: true, disabled: true, id: "node-from-column", label: "Column", value: fromColumnIndex + 1 }), _jsx(TextField, { fullWidth: true, disabled: true, id: "node-from-row", label: "Row", value: fromRowIndex + 1 })] }), _jsx(Box, { sx: {
                    width: "3em",
                    height: "3em",
                    color: "black",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                }, children: _jsx(Typography, { variant: "h6", textAlign: "center", color: palette.background.textPrimary, children: "\u2B95" }) }), _jsxs(Stack, { direction: "column", spacing: 2, justifyContent: "center", alignItems: "center", children: [_jsx(Typography, { variant: "h6", children: t("To") }), _jsx(Autocomplete, { disablePortal: true, id: "node-to-column", options: availableColumns, getOptionLabel: (option) => (option + 1) + "", onChange: (_, value) => handleToColumnSelect(value), value: toColumnIndex, sx: {
                            minWidth: 200,
                            maxWidth: 350,
                        }, renderInput: (params) => _jsx(TextField, { ...params, label: "Column" }) }), _jsx(Autocomplete, { disablePortal: true, id: "node-to-row", options: availableRows, getOptionLabel: (option) => (option + 1) + "", onChange: (_, value) => handleToRowSelect(value), value: toRowIndex, sx: {
                            minWidth: 200,
                            maxWidth: 350,
                        }, renderInput: (params) => _jsx(TextField, { ...params, label: "Row" }) })] })] })), [palette.background.textPrimary, t, fromColumnIndex, fromRowIndex, availableColumns, toColumnIndex, availableRows, toRowIndex, handleToColumnSelect, handleToRowSelect]);
    return (_jsxs(LargeDialog, { id: "move-node-dialog", isOpen: isOpen, onClose: () => { handleClose(); }, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, helpText: t("NodeMoveDialogHelp"), onClose: onClose, title: t("NodeMove") }), _jsxs(DialogContent, { children: [formContent, _jsx(Grid, { container: true, spacing: 1, sx: { padding: 0, paddingTop: "24px" }, children: _jsx(GridSubmitButtons, { display: "dialog", isCreate: false, onCancel: closeDialog, onSubmit: moveNode }) })] })] }));
};
//# sourceMappingURL=MoveNodeDialog.js.map