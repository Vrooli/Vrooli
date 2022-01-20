import { makeStyles } from '@mui/styles';
import { Stack, Theme } from '@mui/material';
import { CombineNodeData, DecisionNodeData, DecisionNodeDataDecision, NodeData, NodeType } from '@local/shared';
import { NodeGraphColumn, NodeGraphEdge } from 'components';
import { useEffect, useMemo } from 'react';
import { Pubs } from 'utils';
import { NodeGraphProps, NodePos } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        cursor: 'move',
        minWidth: '100%',
        minHeight: '100%',
        overflowX: 'scroll',
        overflowY: 'scroll',
    }
}));

export const NodeGraphContainer = ({
    scale = 1,
    isEditable = true,
    labelVisible = true,
    nodes,
}: NodeGraphProps) => {
    const classes = useStyles();

    // Set event listeners for click-and-drag functionality
    useEffect(() => {
        // Mouse drag state
        let touched = false;
        let lastPosition: { x: number, y: number } = { x: 0, y: 0 };
        // Only drag if not pressing a node or edge
        const onMouseDown = (ev: MouseEvent) => {
            const targetId = (ev.target as any)?.id;
            if (!targetId || !targetId.startsWith('node-column')) return;
            touched = true;
            lastPosition = { x: ev.clientX, y: ev.clientY };
        }
        const onMouseUp = (ev: MouseEvent) => {
            touched = false;
        }
        const onMouseMove = (ev: MouseEvent) => {
            if (touched) {
                const deltaX = ev.clientX - lastPosition.x;
                const deltaY = ev.clientY - lastPosition.y;
                document.getElementById('graph-root')?.scrollBy(-deltaX, -deltaY);
                lastPosition = { x: ev.clientX, y: ev.clientY };
            }
        }
        // Add event listeners
        window.addEventListener('mousedown', onMouseDown);
        window.addEventListener('mouseup', onMouseUp);
        window.addEventListener('mousemove', onMouseMove);
        return () => {
            // Remove event listeners
            window.removeEventListener('mousedown', onMouseDown);
            window.removeEventListener('mouseup', onMouseUp);
            window.removeEventListener('mousemove', onMouseMove);
        }
    }, [])

    /**
     * Dictionary of node data and their columns
     */
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
            PubSub.publish(Pubs.Snack, { message: 'No start node found', severity: 'error' });
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
            id={`node-column-${index}`}
            columnNumber={index}
            nodes={columnData}
            isEditable={isEditable}
            scale={scale}
            labelVisible={labelVisible}
            onDrag={() => {}}
            onDrop={() => {}}
        />)
    }, [isEditable, labelVisible, nodeDataMap, scale]);

    // Edge objects
    const edges = useMemo(() => {
        if (!nodes) return [];
        return nodes.map(node => {
            const prevNode = nodeDataMap[node.previous ?? ''];
            const nextNode = nodeDataMap[node.next ?? ''];
            if (!prevNode || !nextNode) return null;
            return <NodeGraphEdge
                key={`edge-${node.id}`}
                from={prevNode}
                to={nextNode}
                isEditable={isEditable}
                scale={scale}
                onAdd={() => {}}
            />
        })
    }, [isEditable, nodeDataMap, nodes, scale]);

    return (
        <div id="graph-root" className={classes.root}>
            {/* Nodes */}
            <Stack spacing={0} direction="row">
                {columns}
            </Stack>
            {/* Edges */}
            {edges}
        </div>
    )
};