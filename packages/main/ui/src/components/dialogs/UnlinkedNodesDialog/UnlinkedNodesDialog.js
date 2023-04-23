import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { NodeType } from "@local/consts";
import { DeleteIcon, ExpandLessIcon, ExpandMoreIcon, UnlinkedNodesIcon } from "@local/icons";
import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "../../../styles";
import { getTranslation } from "../../../utils/display/translationTools";
import { EndNode, RedirectNode, RoutineListNode } from "../../graphs/NodeGraph";
export const UnlinkedNodesDialog = ({ handleNodeDelete, handleToggleOpen, language, nodes, open, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const createNode = useCallback((node) => {
        const nodeProps = {
            canDrag: true,
            handleAction: () => { },
            isEditing: false,
            isLinked: false,
            key: `unlinked-node-${node.id}`,
            label: getTranslation(node, [language], false).name ?? null,
            labelVisible: false,
            scale: -0.5,
            zIndex,
        };
        switch (node.nodeType) {
            case NodeType.End:
                return _jsx(EndNode, { ...nodeProps, handleUpdate: () => { }, language: language, linksIn: [], node: node });
            case NodeType.Redirect:
                return _jsx(RedirectNode, { ...nodeProps, node: node });
            case NodeType.RoutineList:
                return _jsx(RoutineListNode, { ...nodeProps, canExpand: false, labelVisible: true, language: language, handleUpdate: () => { }, linksIn: [], linksOut: [], node: node });
            default:
                return null;
        }
    }, [language, zIndex]);
    return (_jsx(Tooltip, { title: t("NodeUnlinked", { count: 2 }), children: _jsxs(Box, { id: "unlinked-nodes-dialog", sx: {
                alignSelf: open ? "baseline" : "auto",
                borderRadius: 3,
                background: palette.secondary.main,
                color: palette.secondary.contrastText,
                paddingLeft: 1,
                paddingRight: 1,
                marginRight: 1,
                marginTop: open ? "4px" : "unset",
                maxHeight: { xs: "50vh", sm: "65vh", md: "72vh" },
                overflowX: "hidden",
                overflowY: "auto",
                width: open ? { xs: "100%", sm: "375px" } : "fit-content",
                transition: "height 1s ease-in-out",
                zIndex: 1500,
                "&::-webkit-scrollbar": {
                    width: 10,
                },
                "&::-webkit-scrollbar-track": {
                    backgroundColor: "#dae5f0",
                },
                "&::-webkit-scrollbar-thumb": {
                    borderRadius: "100px",
                    backgroundColor: "#409590",
                },
            }, children: [_jsxs(Stack, { direction: "row", onClick: handleToggleOpen, sx: {
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "space-between",
                        width: "100%",
                    }, children: [_jsx(UnlinkedNodesIcon, { fill: palette.secondary.contrastText }), _jsxs(Typography, { variant: "h6", sx: { ...noSelect, marginLeft: "8px" }, children: [open ? (t("Unlinked") + " ") : "", "(", nodes.length, ")"] }), _jsx(Tooltip, { title: t(open ? "Shrink" : "Expand"), children: _jsx(IconButton, { edge: "end", color: "inherit", "aria-label": t(open ? "Shrink" : "Expand"), children: open ? _jsx(ExpandLessIcon, { fill: palette.secondary.contrastText }) : _jsx(ExpandMoreIcon, { fill: palette.secondary.contrastText }) }) })] }), open ? (_jsx(Stack, { direction: "column", spacing: 1, children: nodes.map((node) => (_jsxs(Box, { sx: { display: "flex", alignItems: "center" }, children: [_jsx(Box, { children: createNode(node) }), node.nodeType === NodeType.RoutineList ? null : (_jsx(Typography, { variant: "body1", sx: { marginLeft: 1 }, children: getTranslation(node, [language], true).name })), _jsx(Tooltip, { title: t("NodeDeleteWithName", { nodeName: getTranslation(node, [language], true).name }), placement: "left", children: _jsx(Box, { sx: { marginLeft: "auto" }, children: _jsx(IconButton, { color: "inherit", onClick: () => handleNodeDelete(node.id), "aria-label": t("NodeUnlinkedDelete"), children: _jsx(DeleteIcon, { fill: palette.background.textPrimary }) }) }) })] }, node.id))) })) : null] }) }));
};
//# sourceMappingURL=UnlinkedNodesDialog.js.map