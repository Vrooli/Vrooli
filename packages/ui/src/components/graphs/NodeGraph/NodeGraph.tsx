import { makeStyles } from '@mui/styles';
import { Stack, Theme } from '@mui/material';
import { CombineNodeData, DecisionNodeData, DecisionNodeDataDecision, NodeData, NodeType } from '@local/shared';
import { NodeGraphColumn } from 'components';
import { useMemo } from 'react';
import { PUBS } from 'utils';
import { NodeGraphProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        border: '1px solid purple',
        minWidth: '100%',
        minHeight: '100%',
        overflowX: 'scroll',
        overflowY: 'scroll',
    }
}));

/**
 * Describes the data and general position of a node in the graph. 
 * A completely linear graph would have all nodes at the same level (i.e. one per column, each at row 0).
 * A column with a decision node adds ONE row to the following column FOR EACH possible decision. 
 * A column with a combine node removes ONE row from the following column FOR EACH possible combination, AND 
 * places itself in the vertical center of each combined node
 */
type NodePos = {
    column: number; // column in which node is displayed
    // rows: number; // number of rows in the column the node is displayed in
    // pos: number; // relative position in column. 0 is top, 1 is bottom
    node: NodeData;
}
export const NodeGraph = ({
    scale = 1,
    isEditable = true,
    labelVisible = true,
    nodes,
}: NodeGraphProps) => {
    const classes = useStyles();

    // Dictionary of node data and their columns
    const nodeDataMap: { [id: string]: NodePos } = useMemo(() => {
        // Position map for calculating node positions
        let posMap: { [id: string]: NodePos } = {};
        if (!nodes) return posMap;
        let startNodeId: string | null = null;
        console.log('node data map', nodes);
        // First pass of raw node data, to locate start node and populate position map
        for (let i = 0; i < nodes.length; i++) {
            console.log('raw data', nodes[i]);
            const currId = nodes[i].id;
            // If start node, must be in first column
            if (nodes[i].type === NodeType.Start) {
                startNodeId = currId;
                posMap[currId] = {
                    column: 0,
                    node: nodes[i],
                }
            }
            // Otherwise, node's column is currently unknown
            else {
                posMap[currId] = {
                    column: -1,
                    node: nodes[i],
                }
            }
        }
        // If no start node was found, throw error
        if (!startNodeId) {
            PubSub.publish(PUBS.Snack, { message: 'No start node found', severity: 'error' });
            return posMap;
        }
        // Helper function for recursively updating position map
        const addNode = (nodeId: string) => {
            // Find node data
            const curr: NodePos = posMap[nodeId];
            console.log('addNode', nodeId, curr);
            // If node not found or column already calculated (exception being the start node at column 0), skip
            if (!curr || curr.column > 0) return;
            console.log('a')
            // Calculate node's column (unless it is the start node). This is the same for all node types EXCEPT for combine nodes
            if (curr.node.type === NodeType.Combine) {
                console.log('b.1')
                const previousNodes = (curr.node.data as CombineNodeData | null)?.from ?? [];
                const farthestPreviousNode = Math.max(...previousNodes.map(prev => posMap[prev].column));
                posMap[nodeId].column = farthestPreviousNode === -1 ? -1 : farthestPreviousNode + 1;
            }
            else if (curr.node.type !== NodeType.Start) {
                console.log('b.2')
                const prevNode = posMap[curr.node.previous ?? ''];
                console.log('prevNode', prevNode);
                if (!prevNode) return;
                posMap[nodeId].column = prevNode.column === -1 ? -1 : prevNode.column + 1;
                console.log('calculated column', posMap[nodeId].column);
            }
            // Call addNode on each next node. Thiss is the same for all node types EXCEPT for decision nodes
            if (curr.node.type === NodeType.Decision) {
                console.log('in decision logic', curr?.node?.data)
                const decisions: DecisionNodeDataDecision[] | undefined = (curr.node.data as DecisionNodeData | undefined)?.decisions;
                if (!decisions) return;
                for (let i = 0; i < decisions.length; i++) {
                    addNode(decisions[i].next);
                }
            }
            else {
                if (curr.node.next) addNode(curr.node.next);
            }
        }
        // Starting with the start node, search for other nodes
        addNode(startNodeId);
        return posMap;
    }, [nodes]);

    // Node column objects
    const columns = useMemo(() => {
        console.log('calculating columns', nodeDataMap);
        // 2D node data array, ordered by column. 
        // Each column is ordered in a consistent way, so that the nodes in a column are always in the same order
        let list: NodeData[][] = [];
        // Iterate through node data map
        for (const value of Object.values(nodeDataMap)) {
            // Skips nodes that did not receive a column
            if (value.column < 0) continue;
            // Add new column(s) if necessary
            while (list.length <= value.column) {
                list.push([]);
            }
            // Add node to column
            list[value.column].push(value.node);
        }
        // Sort each column
        // TODO
        // return column objects
        return list.map((columnData, index) => <NodeGraphColumn
            key={`node-column-${index}`}
            columnNumber={index}
            nodes={columnData}
            isEditable={isEditable}
            scale={scale}
            labelVisible={labelVisible}
        />)
    }, [isEditable, labelVisible, nodeDataMap, scale]);

    return (
        <Stack spacing={0} direction="row" className={classes.root}>
            {columns}
        </Stack>
    )
};