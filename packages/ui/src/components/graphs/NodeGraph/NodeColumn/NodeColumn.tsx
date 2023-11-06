/**
 * Displays a list of nodes vertically.
 */
import { Node, NodeRoutineList, NodeType } from "@local/shared";
import { Box, Stack } from "@mui/material";
import { useMemo } from "react";
import { getTranslation } from "utils/display/translationTools";
import { NodeShape } from "utils/shape/models/node";
import { NodeWithEnd } from "views/objects/node/types";
import { calculateNodeSize, EndNode, RedirectNode, RoutineListNode, StartNode } from "../nodes";
import { NodeColumnProps } from "../types";

export const NodeColumn = ({
    handleAction,
    handleNodeUpdate,
    id,
    isEditing,
    columnIndex,
    columnsLength,
    labelVisible,
    language,
    links,
    nodes,
    scale = 1,
}: NodeColumnProps) => {
    // Padding between cells
    const padding = useMemo(() => calculateNodeSize(50, scale), [scale]);

    /**
     * Create a node component for the given node data. 
     * Each node is wrapped in a cell that accepts drag and drop. 
     */
    const nodeList = useMemo(() => {
        // Sort nodes by their row index
        if (nodes.length === 0) return null;
        // There may be gaps between the nodes. For each missing rowIndex,
        // add a placeholder node to fill the gap.
        // Create an array that's the length of the largest rowIndex + 1
        const nodesWithGaps = Array(Math.max(...nodes.map(node => (node.rowIndex ?? 0))) + 1).fill(null);
        // Loop through the nodes and add them to the array
        nodes.forEach(node => {
            if (node.rowIndex === undefined || node.rowIndex === null) return;
            nodesWithGaps[node.rowIndex as number] = node;
        });
        // Now that we have a complete array, create a list of nodes
        return nodesWithGaps.map((node: NodeShape | null, index) => {
            // If a placeholder, return a placeholder node
            if (node === null) {
                return (
                    <Box key={`node-placeholder-${columnIndex}-${index}`} sx={{
                        width: `${calculateNodeSize(100, scale)}px`,
                        height: `${calculateNodeSize(350, scale)}px`,
                    }} />
                );
            }
            // Otherwise, return correct node
            // Common node props
            const nodeProps = {
                key: `node-${columnIndex}-${index}`,
                handleAction,
                isLinked: true,
                scale,
                label: getTranslation(node, [language], false).name ?? undefined,
                labelVisible,
                isEditing,
                canDrag: isEditing,
            };
            // Determine node to display based on nodeType
            switch (node.nodeType) {
                case NodeType.End:
                    return <EndNode
                        {...nodeProps}
                        handleUpdate={handleNodeUpdate}
                        language={language}
                        linksIn={links.filter(l => l.to.id === node.id)}
                        node={node as NodeWithEnd}
                    />;
                case NodeType.Redirect:
                    return <RedirectNode
                        {...nodeProps}
                        node={node}
                    />;
                case NodeType.RoutineList:
                    return (<RoutineListNode
                        {...nodeProps}
                        canExpand={true}
                        handleUpdate={handleNodeUpdate}
                        language={language}
                        linksIn={links.filter(l => l.to.id === node.id)}
                        linksOut={links.filter(l => l.from.id === node.id)}
                        node={node as Node & { routineList: NodeRoutineList }}
                    />);
                case NodeType.Start:
                    return <StartNode
                        {...nodeProps}
                        linksOut={links.filter(l => l.from.id === node.id)}
                        node={node}
                    />;
                default:
                    return null;
            }
        });
    }, [columnIndex, handleAction, handleNodeUpdate, isEditing, labelVisible, language, links, nodes, scale]);

    return (
        <Stack
            id={id}
            direction="column"
            padding={`${padding}px`}
            paddingTop={"100px"}
            paddingBottom={"100px"}
            position="relative"
            display="flex"
            justifyContent="center"
            alignItems="center"
            sx={{
                // pointerEvents: 'none',
                gap: `${Math.max(padding * 4, 30)}px`,
                // Fill available if column is empty
                width: nodes.length === 0 ? "-webkit-fill-available" : "auto",
                paddingLeft: "0px",
                paddingRight: "0px",
                marginLeft: `${Math.max(padding * 4, 30)}px`,
                marginRight: `${Math.max(padding * 4, 30)} px`,
            }}
        >
            {nodeList}
        </Stack >
    );
};
