import { makeStyles } from '@mui/styles';
import { Stack, Theme, Typography } from '@mui/material';
import { CombineNodeData, DecisionNodeData, DecisionNodeDataDecision, NodeData, NodeType, OrchestrationData } from '@local/shared';
import { NodeColumn } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { PUBS } from 'utils';

//TEMP
const data: OrchestrationData = {
    title: 'Validate business idea',
    description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.',
    // Unordered list of top-level nodes
    // Page renders nodes starting at the start node
    nodes: [
        // Start node
        {
            id: '1',
            type: NodeType.Start,
            title: null,
            description: null,
            // ID of previous node
            previous: null,
            // ID of next node
            next: '2',
            // Additional data specific to the node type
            data: null
        },
        // Routine List node
        {
            id: '2',
            type: NodeType.RoutineList,
            title: 'Provide Basic Info',
            description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
            previous: '1',
            next: '3',
            data: {
                isOrdered: false,
                isOptional: false,
                routines: [
                    {
                        id: '1',
                        title: 'Provide Basic Info',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: false,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    }
                ]
            }
        },
        // Routine List node
        {
            id: '3',
            type: NodeType.RoutineList,
            title: 'Knowledge management',
            description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
            previous: '2',
            next: '4',
            data: {
                isOrdered: false,
                isOptional: false,
                routines: [
                    {
                        id: '2',
                        title: 'Create task list',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    },
                    {
                        id: '3',
                        title: 'Generate forecasts',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    },
                    {
                        id: '4',
                        title: 'List objectives',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: false,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    },
                    {
                        id: '5',
                        title: 'Create business model',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    }
                ]
            }
        },
        // Decision node
        {
            id: '4',
            type: NodeType.Decision,
            title: 'Worth pursuing?',
            description: null,
            previous: '3',
            // Decision nodes have no definitive next node
            // Instead, next node is determined by the decision
            next: null,
            data: {
                decisions: [
                    {
                        id: '1',
                        title: 'Yes',
                        description: null,
                        next: '5',
                        // List of cases which must return true in order for this decision to be available
                        // If empty, decision is always available
                        when: []
                    },
                    {
                        id: '2',
                        title: 'No',
                        description: null,
                        next: '6',
                        when: []
                    }
                ]
            }
        },
        // Routine List node
        {
            id: '5',
            type: NodeType.RoutineList,
            title: 'todo',
            description: null,
            previous: '4',
            next: '7',
            data: {
                isOrdered: false,
                isOptional: false,
                routines: []
            },
        },
        // End node
        {
            id: '6',
            type: NodeType.End,
            title: 'Wasnt worth pursuing',
            description: 'afda',
            previous: '4',
            next: null,
            data: {
                wasSuccessful: true,
            }
        },
        // Decision node
        {
            id: '7',
            type: NodeType.Decision,
            title: 'Try again?',
            description: null,
            previous: '5',
            next: null,
            data: {
                decisions: [
                    {
                        id: '3',
                        title: 'Yes',
                        description: null,
                        next: '8',
                        // List of cases which must return true in order for this decision to be available
                        // If empty, decision is always available
                        when: []
                    },
                    {
                        id: '4',
                        title: 'No',
                        description: null,
                        next: '9',
                        when: []
                    }
                ]
            }
        },
        // Redirect node
        {
            id: '8',
            type: NodeType.Redirect,
            title: null,
            description: null,
            previous: '7',
            next: '1',
            data: null,
        },
        // End node
        {
            id: '9',
            type: NodeType.End,
            title: 'The good end',
            description: 'asdf',
            previous: '7',
            next: null,
            data: {
                wasSuccessful: true,
            }
        },
    ]
}

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        minWidth: '100%',
        minHeight: '100%',
        paddingTop: '10vh'
    },
    title: {
        textAlign: 'center',
    },
    // Horizontal scrolling container
    graphContainer: {
        height: '75vh',
        border: '1px solid red',
        // display: 'flex',
        // flexDirection: 'row',
        // flexWrap: 'nowrap',
        // justifyContent: 'flex-start',
        // alignItems: 'center',
        // alignContent: 'center',
    },
    graphStack: {
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

// /**
//  * Node data organized by row and column. This is accomplished 
//  * by using a 2D array, where the first dimension is the column, 
//  * and the second dimension is the row.
//  * A completely linear graph would have all nodes at the same level (i.e. one per column, each at row 0).
//  * A column with a decision node adds ONE row to the following column FOR EACH possible decision. 
//  * A column with a combine node removes ONE row from the following column FOR EACH possible combination, AND 
//  * places itself in the vertical center of each combined node
//  */
// type OrderedNodeList = Array<{
//     rows: number; // Number of node rows in this column (NOT necessarily the number of nodes in this column. Rather, used to determine the positioning of each node)
//     nodes: {
//         node: NodeData;
//         position: number; // Position of node in this column. Must be between 0 and #rows - 1 (CAN be non-integer)
//     }
// }>;

export const RoutineOrchestratorPage = () => {
    const classes = useStyles();
    // Queries routine data
    const { data: routineData } = useQuery<routine>(routineQuery, { variables: { input: { id: 'TODO' } } });
    // Routine mutator
    const [routineUpdate] = useMutation<any>(routineUpdateMutation);
    // The routine's title
    const [title, setTitle] = useState<string>('');
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);

    useEffect(() => {
        setTitle(routineData?.routine?.title ?? '');
    }, [routineData]);

    /**
     * Mutates routine data
     */
    const updateRoutine = useCallback(() => {
        if (!routineData?.routine?.id) {
            PubSub.publish(PUBS.Snack, { message: 'Cannot update: Invalid routine data', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: routineUpdate,
            input: {
                id: routineData.routine.id,
                title,
            },
            successMessage: () => 'Routine updated.',
        })
    }, [routineData, routineUpdate, title])

    // Unorganized graph data
    const nodeDataRaw = data.nodes;//useMemo<NodeData[]>(() => routineData?.routine?.nodes ?? [], [routineData]);
    // Dictionary of node data and their columns
    const nodeDataMap: { [id: string]: NodePos } = useMemo(() => {
        // Position map for calculating node positions
        let posMap: { [id: string]: NodePos } = {};
        let startNodeId: string | null = null;
        console.log('node data map', nodeDataRaw);
        // First pass of raw node data, to locate start node and populate position map
        for (let i = 0; i < nodeDataRaw.length; i++) {
            console.log('raw data', nodeDataRaw[i]);
            const currId = nodeDataRaw[i].id;
            // If start node, must be in first column
            if (nodeDataRaw[i].type === NodeType.Start) {
                startNodeId = currId;
                posMap[currId] = {
                    column: 0,
                    node: nodeDataRaw[i],
                }
            }
            // Otherwise, node's column is currently unknown
            else {
                posMap[currId] = {
                    column: -1,
                    node: nodeDataRaw[i],
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
    }, [nodeDataRaw]);

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
        return list.map((columnData, index) => <NodeColumn
            key={`node-column-${index}`}
            columnNumber={index}
            nodes={columnData}
            isEditable={true}
            scale={scale}
            labelVisible={true}
        />)
    }, [nodeDataMap, scale]);

    return (
        <div className={classes.root}>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <div className={classes.graphContainer}>
                <Stack spacing={10} direction="row" className={classes.graphStack}>
                    {columns}
                </Stack>
            </div>
        </div>
    )
};