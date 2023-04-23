import { jsx as _jsx } from "react/jsx-runtime";
import { NodeType } from "@local/consts";
import { Box, Stack } from "@mui/material";
import { useMemo } from "react";
import { getTranslation } from "../../../../utils/display/translationTools";
import { calculateNodeSize, EndNode, RedirectNode, RoutineListNode, StartNode } from "../nodes";
export const NodeColumn = ({ handleAction, handleNodeUpdate, id, isEditing, columnIndex, labelVisible, language, links, nodes, scale = 1, zIndex, }) => {
    const padding = useMemo(() => calculateNodeSize(25, scale), [scale]);
    const nodeList = useMemo(() => {
        if (nodes.length === 0)
            return null;
        const nodesWithGaps = Array(Math.max(...nodes.map(node => (node.rowIndex ?? 0))) + 1).fill(null);
        nodes.forEach(node => {
            if (node.rowIndex === undefined || node.rowIndex === null)
                return;
            nodesWithGaps[node.rowIndex] = node;
        });
        return nodesWithGaps.map((node, index) => {
            if (node === null) {
                return (_jsx(Box, { sx: {
                        width: `${calculateNodeSize(100, scale)}px`,
                        height: `${calculateNodeSize(350, scale)}px`,
                    } }, `node-placeholder-${columnIndex}-${index}`));
            }
            const nodeProps = {
                key: `node-${columnIndex}-${index}`,
                handleAction,
                isLinked: true,
                scale,
                label: getTranslation(node, [language], false).name ?? undefined,
                labelVisible,
                isEditing,
                canDrag: isEditing,
                zIndex,
            };
            switch (node.nodeType) {
                case NodeType.End:
                    return _jsx(EndNode, { ...nodeProps, handleUpdate: handleNodeUpdate, language: language, linksIn: links.filter(l => l.to.id === node.id), node: node });
                case NodeType.Redirect:
                    return _jsx(RedirectNode, { ...nodeProps, node: node });
                case NodeType.RoutineList:
                    return (_jsx(RoutineListNode, { ...nodeProps, canExpand: true, handleUpdate: handleNodeUpdate, language: language, linksIn: links.filter(l => l.to.id === node.id), linksOut: links.filter(l => l.from.id === node.id), node: node }));
                case NodeType.Start:
                    return _jsx(StartNode, { ...nodeProps, linksOut: links.filter(l => l.from.id === node.id), node: node });
                default:
                    return null;
            }
        });
    }, [columnIndex, handleAction, handleNodeUpdate, isEditing, labelVisible, language, links, nodes, scale, zIndex]);
    return (_jsx(Stack, { id: id, direction: "column", padding: `${padding}px`, paddingTop: "100px", paddingBottom: "100px", position: "relative", display: "flex", justifyContent: "center", alignItems: "center", sx: {
            gap: `${padding * 4}px`,
            width: nodes.length === 0 ? "-webkit-fill-available" : "auto",
        }, children: nodeList }));
};
//# sourceMappingURL=NodeColumn.js.map