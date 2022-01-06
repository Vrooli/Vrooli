import { makeStyles } from '@mui/styles';
import { Stack, Theme } from '@mui/material';
import { useMemo } from 'react';
import { NodeGraphColumnProps } from '../types';
import { NodeType, RoutineListNodeData } from '@local/shared';
import { CombineNode, DecisionNode, EndNode, LoopNode, RedirectNode, RoutineListNode, StartNode } from '../../nodes';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        position: 'relative',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        backgroundColor: 'transparent',
    },
}));

export const NodeGraphColumn = ({
    scale = 1,
    columnNumber,
    nodes,
    labelVisible,
    isEditable,
}: NodeGraphColumnProps) => {
    const classes = useStyles();
    const padding = useMemo(() => `${scale*25}px`, [scale]);

    const nodeList = useMemo(() => nodes?.map((node, index) => {
        const commonProps = {
            key: `node-${columnNumber}-${index}`,
            scale,
            label: node?.title,
            labelVisible,
            isEditable
        }
        switch (node.type) {
            case NodeType.Combine:
                return <CombineNode {...commonProps} />;
            case NodeType.Decision:
                return <DecisionNode {...commonProps} />;
            case NodeType.End:
                return <EndNode {...commonProps} />;
            case NodeType.Loop:
                return <LoopNode {...commonProps} />;
            case NodeType.RoutineList:
                return <RoutineListNode {...commonProps} data={node.data as RoutineListNodeData} onAdd={() => {}} />;
            case NodeType.Redirect:
                return <RedirectNode {...commonProps} />;
            case NodeType.Start:
                return <StartNode {...commonProps} />;
            default:
                return null;
        }
    }) ?? [], [columnNumber, isEditable, labelVisible, nodes, scale])

    return (
        <Stack spacing={10} direction="column" className={classes.root} style={{padding}}>
            {nodeList}
        </Stack>
    )
}