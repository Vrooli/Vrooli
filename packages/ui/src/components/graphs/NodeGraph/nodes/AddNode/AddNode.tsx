import { makeStyles } from '@mui/styles';
import { Box, IconButton, Tooltip } from '@mui/material';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { AddNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';
import { ListMenu } from 'components';
import {
    Add as AddIcon,
    AltRoute as DecisionIcon,
    Done as EndIcon,
    List as RoutineListIcon,
    Loop as LoopIcon,
    MergeType as CombineIcon,
    SvgIconComponent,
    UTurnLeft as RedirectIcon
} from '@mui/icons-material';
import { NodeType } from '@local/shared';

const optionsMap: { [x: string]: [string, SvgIconComponent] } = {
    [NodeType.Combine]: ['Combine', CombineIcon],
    [NodeType.Decision]: ['Decision', DecisionIcon],
    [NodeType.End]: ['End', EndIcon],
    [NodeType.Loop]: ['Loop', LoopIcon],
    [NodeType.RoutineList]: ['Routine List', RoutineListIcon],
    [NodeType.Redirect]: ['Redirect', RedirectIcon],
}

export const AddNode = ({
    scale = 1,
    options = Object.values(NodeType).filter(o => o !== NodeType.Start),
    onAdd,
}: AddNodeProps) => {
    // Add node menu popup
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = 'add-node-menu';
    const openDialog = useCallback((ev: MouseEvent<HTMLButtonElement>) => {
        setContextAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeDialog = useCallback(() => setContextAnchor(null), []);

    const listOptions = useMemo(() => options.map(o => ({
        label: optionsMap[o][0],
        value: o,
        Icon: optionsMap[o][1]
    })), [options]);

    const nodeSize = useMemo(() => `${100 * scale}px`, [scale]);

    return (
        <Box>
            <ListMenu
                id={contextId}
                anchorEl={contextAnchor}
                title='Add Step'
                data={listOptions}
                onSelect={onAdd}
                onClose={closeDialog}
            />
            <Tooltip placement={'top'} title='Insert step'>
                <IconButton
                    onClick={openDialog}
                    sx={{
                        width: nodeSize,
                        height: nodeSize,
                        position: 'relative',
                        display: 'flex',
                        alignItems: 'center',
                        padding: '0',
                        backgroundColor: '#6daf72',
                        color: 'white',
                        borderRadius: '100%',
                        boxShadow: '0px 0px 12px gray',
                        '&:hover': {
                            backgroundColor: '#6daf72',
                            filter: `brightness(120%)`,
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    <AddIcon
                        sx={{
                            width: '80%',
                            height: '80%',
                        }}
                    />
                </IconButton>
            </Tooltip>
        </Box>
    )
}