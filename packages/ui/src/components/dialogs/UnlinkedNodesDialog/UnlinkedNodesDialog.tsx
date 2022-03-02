/**
 * Displays nodes associated with a routine, but that are not linked to any other nodes.
 */
import {
    CloseFullscreen as ShrinkIcon,
    Delete as DeleteIcon,
    OpenInFull as ExpandIcon,
} from '@mui/icons-material';
import {
    Box,
    IconButton,
    Stack,
    Tooltip,
    Typography,
} from '@mui/material';
import { UnlinkedNodesDialogProps } from '../types';
import { containerShadow } from 'styles';
import { useCallback } from 'react';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node } from 'types';
import { EndNode, LoopNode, RedirectNode, RoutineListNode } from 'components';

export const UnlinkedNodesDialog = ({
    open,
    nodes,
    handleToggleOpen,
    handleDeleteNode,
}: UnlinkedNodesDialogProps) => {

    /**
     * Generates a simple node from a node type
     */
    const createNode = useCallback((node: Node) => {
        console.log('in unlinkedlist createnode', node);
        // Common node props
        const nodeProps = {
            key: `unlinked-node-${node.id}`,
            node,
            scale: 0.5,
            label: '',
            labelVisible: false,
            isEditable: false,
        }
        // Determine node to display based on node type
        switch (node.type) {
            case NodeType.End:
                return <EndNode {...nodeProps} />
            case NodeType.Loop:
                return <LoopNode {...nodeProps} />
            case NodeType.Redirect:
                return <RedirectNode {...nodeProps} />
            case NodeType.RoutineList:
                return <RoutineListNode {...nodeProps} canExpand={false} labelVisible={true} label={node.title} onAdd={() => { }} onResize={() => { }} handleDialogOpen={() => { }} />
            default:
                return null;
        }
    }, [])

    return (
        <Box sx={{
            borderRadius: 3,
            background: '#c7dee2',
            color: 'black',
            padding: 1,
            maxheight: '70vh',
            width: 'fit-content',
            transition: 'height 1s ease-in-out',
            ...containerShadow
        }}>
            <Box sx={{
                display: 'flex',
                alignItems: 'center',
                width: '100%'
            }}>
                <Tooltip title={open ? 'Shrink' : 'Expand'}>
                    <IconButton edge="start" color="inherit" onClick={handleToggleOpen} aria-label={open ? 'Shrink' : 'Expand'}>
                        {open ? <ShrinkIcon sx={{ fill: 'black' }} /> : <ExpandIcon sx={{ fill: 'black' }} />}
                    </IconButton>
                </Tooltip>
                <Typography variant="h5">Unlinked Nodes ({nodes.length})</Typography>
            </Box>
            {open ? (
                <Stack direction="column" spacing={1}>
                    {nodes.map((node) => (
                        <Box key={node.id} sx={{ display: 'flex', alignItems: 'center' }}>
                            {/* Miniature version of node */}
                            <Box sx={{
                                width: '50px',
                                height: '50px',
                            }}>
                                {createNode(node)}
                            </Box>
                            {/* Node title */}
                            {node.type === NodeType.RoutineList ? null : (<Typography variant="body1" sx={{marginLeft: 1}}>{node.title}</Typography>)}
                            {/* Delete node icon */}
                            <Tooltip title={`Delete ${node.title} node`} placement="left">
                                <Box sx={{ marginLeft: 'auto' }}>
                                    <IconButton
                                        color="inherit"
                                        onClick={() => handleDeleteNode(node)}
                                        aria-label={'Delete unlinked node'}
                                    >
                                        <DeleteIcon sx={{ 
                                            fill: 'black',
                                            '&:hover': {
                                                fill: '#ff6a6a'
                                            },
                                            transition: 'fill 1s ease-in-out',
                                        }} />
                                    </IconButton>
                                </Box>
                            </Tooltip>
                        </Box>
                    ))}
                </Stack>
            ) : null}

        </Box>
    );
}