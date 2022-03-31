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
import { EndNode, RedirectNode, RoutineListNode } from 'components';
import { getTranslation } from 'utils';

export const UnlinkedNodesDialog = ({
    open,
    nodes,
    handleNodeDelete,
    handleToggleOpen,
}: UnlinkedNodesDialogProps) => {

    /**
     * Generates a simple node from a node type
     */
    const createNode = useCallback((node: Node) => {
        // Common node props
        const nodeProps = {
            canDrag: true,
            handleAction: () => { },
            isEditing: false,
            isLinked: false,
            key: `unlinked-node-${node.id}`,
            label: '',
            labelVisible: false,
            node,
            scale: 0.5,
        }
        // Determine node to display based on node type
        switch (node.type) {
            case NodeType.End:
                return <EndNode {...nodeProps} />
            case NodeType.Redirect:
                return <RedirectNode {...nodeProps} />
            case NodeType.RoutineList:
                return <RoutineListNode
                    {...nodeProps}
                    canExpand={false}
                    labelVisible={true}
                    handleUpdate={() => { }} // Intentionally blank
                />
            default:
                return null;
        }
    }, [])

    return (
        <Box id="unlinked-nodes-dialog" sx={{
            borderRadius: 3,
            background: '#c7dee2',
            color: 'black',
            padding: 1,
            maxHeight: { xs: '62vh', sm: '65vh', md: '72vh' },
            overflowY: 'auto',
            width: open ? '250px' : 'fit-content',
            transition: 'height 1s ease-in-out',
            ...containerShadow,
            "&::-webkit-scrollbar": {
                width: 10,
            },
            "&::-webkit-scrollbar-track": {
                backgroundColor: '#dae5f0',
            },
            "&::-webkit-scrollbar-thumb": {
                borderRadius: '100px',
                backgroundColor: "#409590",
            },
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
                <Typography variant="h6">Unlinked ({nodes.length})</Typography>
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
                            {node.type === NodeType.RoutineList ? null : (<Typography variant="body1" sx={{ marginLeft: 1 }}>{getTranslation(node, 'title', ['en'], true)}</Typography>)}
                            {/* Delete node icon */}
                            <Tooltip title={`Delete ${getTranslation(node, 'title', ['en'], true)} node`} placement="left">
                                <Box sx={{ marginLeft: 'auto' }}>
                                    <IconButton
                                        color="inherit"
                                        onClick={() => handleNodeDelete(node.id)}
                                        aria-label={'Delete unlinked node'}
                                    >
                                        <DeleteIcon sx={{
                                            fill: 'black',
                                            '&:hover': {
                                                fill: '#ff6a6a'
                                            },
                                            transition: 'fill 0.5s ease-in-out',
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