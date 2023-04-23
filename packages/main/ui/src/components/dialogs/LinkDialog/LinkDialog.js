import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { NodeType } from "@local/consts";
import { uuid } from "@local/uuid";
import { Autocomplete, Box, DialogContent, Grid, Stack, TextField, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getTranslation } from "../../../utils/display/translationTools";
import { PubSub } from "../../../utils/pubsub";
import { GridSubmitButtons } from "../../buttons/GridSubmitButtons/GridSubmitButtons";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const helpText = "This dialog allows you create new links between nodes, which specifies the order in which the nodes are executed.\n\nIn the future, links will also be able to specify conditions, which must be true in order for the path to be available.";
const titleId = "link-dialog-title";
export const LinkDialog = ({ handleClose, handleDelete, isAdd, isOpen, language, link, nodeFrom, nodeTo, routineVersion, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [fromNode, setFromNode] = useState(nodeFrom ?? null);
    const handleFromSelect = useCallback((node) => {
        setFromNode(node);
    }, [setFromNode]);
    const [toNode, setToNode] = useState(nodeTo ?? null);
    const handleToSelect = useCallback((node) => {
        setToNode(node);
    }, [setToNode]);
    useEffect(() => { setFromNode(nodeFrom ?? null); }, [nodeFrom, setFromNode]);
    useEffect(() => { setToNode(nodeTo ?? null); }, [nodeTo, setToNode]);
    const errors = useMemo(() => {
        const errors = {};
        if (!fromNode) {
            errors.fromNode = t("NodeFromRequired", { ns: "error", defaultValue: "NodeFromRequired" });
        }
        if (!toNode) {
            errors.toNode = t("NodeToRequired", { ns: "error", defaultValue: "NodeToRequired" });
        }
        return errors;
    }, [fromNode, t, toNode]);
    const addLink = useCallback(() => {
        if (!fromNode || !toNode) {
            PubSub.get().publishSnack({ messageKey: "SelectFromAndToNodes", severity: "Error" });
            return;
        }
        handleClose({
            id: uuid(),
            from: { id: fromNode.id },
            operation: null,
            routineVersion: { id: routineVersion.id },
            to: { id: toNode.id },
            whens: [],
        });
        setFromNode(null);
        setToNode(null);
    }, [fromNode, toNode, handleClose, routineVersion.id]);
    const { fromOptions, toOptions } = useMemo(() => {
        if (!routineVersion)
            return { fromOptions: [], toOptions: [] };
        let fromNodes = routineVersion.nodes.filter((node) => node.nodeType !== NodeType.Start);
        let toNodes = routineVersion.nodes.filter((node) => node.nodeType !== NodeType.Start);
        const existingLinks = routineVersion.nodeLinks;
        if (fromNode) {
            toNodes = toNodes.filter(node => node.id !== fromNode.id);
            toNodes = toNodes.filter(node => !existingLinks.some(link => link.from.id === fromNode.id && link.to.id === node.id));
        }
        if (toNode) {
            fromNodes = fromNodes.filter(node => node.id !== toNode.id);
            fromNodes = fromNodes.filter(node => !existingLinks.some(link => link.from.id === node.id && link.to.id === toNode.id));
        }
        return { fromOptions: fromNodes, toOptions: toNodes };
    }, [fromNode, routineVersion, toNode]);
    const getNodeTitle = useCallback((node) => {
        const { name } = getTranslation(node, [language]);
        if (name)
            return name;
        if (node.nodeType === NodeType.Start)
            return t("Start");
        if (node.nodeType === NodeType.End)
            return t("End");
        return t("Untitled");
    }, [language, t]);
    const nodeSelections = useMemo(() => (_jsxs(Stack, { direction: "row", justifyContent: "center", alignItems: "center", pt: 2, children: [_jsx(Autocomplete, { disablePortal: true, id: "link-connect-from", options: fromOptions, getOptionLabel: (option) => getNodeTitle(option), onChange: (_, value) => handleFromSelect(value), value: fromNode, sx: {
                    minWidth: 200,
                    maxWidth: 350,
                }, renderInput: (params) => _jsx(TextField, { ...params, label: "From" }) }), _jsx(Box, { sx: {
                    width: "3em",
                    height: "3em",
                    color: "black",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                }, children: _jsx(Typography, { variant: "h6", textAlign: "center", color: palette.background.textPrimary, children: "\u2B95" }) }), _jsx(Autocomplete, { disablePortal: true, id: "link-connect-to", options: toOptions, getOptionLabel: (option) => getNodeTitle(option), onChange: (_, value) => handleToSelect(value), value: toNode, sx: {
                    minWidth: 200,
                    maxWidth: 350,
                }, renderInput: (params) => _jsx(TextField, { ...params, label: "To" }) })] })), [fromOptions, fromNode, palette.background.textPrimary, toOptions, toNode, getNodeTitle, handleFromSelect, handleToSelect]);
    const conditions = useMemo(() => (null), []);
    const deleteOption = useMemo(() => isAdd ? (_jsx(_Fragment, {})) : null, [isAdd]);
    const handleCancel = useCallback((_, reason) => {
        if ((fromNode !== nodeFrom || toNode !== nodeTo) && reason === "backdropClick")
            return;
        setFromNode(null);
        setToNode(null);
        handleClose();
    }, [fromNode, handleClose, nodeFrom, nodeTo, toNode]);
    return (_jsxs(LargeDialog, { id: "link-dialog", isOpen: isOpen, onClose: handleCancel, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, title: t(isAdd ? "LinkAdd" : "LinkEdit"), helpText: helpText, onClose: handleCancel }), _jsxs(DialogContent, { children: [nodeSelections, conditions, deleteOption, _jsx(Grid, { container: true, spacing: 2, mt: 2, mb: 8, children: _jsx(GridSubmitButtons, { display: "dialog", errors: errors, isCreate: isAdd, onCancel: handleCancel, onSubmit: addLink }) })] })] }));
};
//# sourceMappingURL=LinkDialog.js.map