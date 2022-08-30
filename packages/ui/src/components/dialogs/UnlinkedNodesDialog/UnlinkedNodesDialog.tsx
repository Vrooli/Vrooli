/**
 * Displays nodes associated with a routine, but that are not linked to any other nodes.
 */
import {
    ExpandLess as ShrinkIcon,
    ExpandMore as ExpandIcon,
    Delete as DeleteIcon,
} from '@mui/icons-material';
import {
    Box,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { UnlinkedNodesDialogProps } from '../types';
import { noSelect } from 'styles';
import { useCallback } from 'react';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node } from 'types';
import { EndNode, RedirectNode, RoutineListNode } from 'components';
import { getTranslation } from 'utils';
import { UnlinkedNodesIcon } from '@shared/icons';

export const UnlinkedNodesDialog = ({
    handleNodeDelete,
    handleToggleOpen,
    language,
    nodes,
    open,
    zIndex,
}: UnlinkedNodesDialogProps) => {
    const { palette } = useTheme();

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
            label: getTranslation(node, 'title', [language], false) ?? null,
            labelVisible: false,
            node,
            scale: 0.5,
            zIndex,
        }
        // Determine node to display based on node type
        switch (node.type) {
            case NodeType.End:
                return <EndNode {...nodeProps} linksIn={[]} />
            case NodeType.Redirect:
                return <RedirectNode {...nodeProps} />
            case NodeType.RoutineList:
                return <RoutineListNode
                    {...nodeProps}
                    canExpand={false}
                    labelVisible={true}
                    language={language}
                    handleUpdate={() => { }} // Intentionally blank
                    linksIn={[]}
                    linksOut={[]}
                />
            default:
                return null;
        }
    }, [language, zIndex])

    return (
        <Tooltip title="Unlinked nodes">
            <Box id="unlinked-nodes-dialog" sx={{
                alignSelf: open ? 'baseline' : 'auto',
                borderRadius: 3,
                background: palette.mode === 'light' ? '#c7dee2' : '#315672',
                color: palette.text.primary,
                paddingLeft: 1,
                paddingRight: 1,
                marginRight: 1,
                marginTop: open ? '4px' : 'unset',
                maxHeight: { xs: '62vh', sm: '65vh', md: '72vh' },
                overflowX: 'hidden',
                overflowY: 'auto',
                width: open ? '250px' : 'fit-content',
                transition: 'height 1s ease-in-out',
                zIndex: 1500,
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
                <Stack direction="row" onClick={handleToggleOpen} sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    width: '100%'
                }}>
                    <UnlinkedNodesIcon fill={palette.background.textPrimary} />
                    <Typography variant="h6" sx={{ ...noSelect, marginLeft: '8px' }}>{open ? 'Unlinked ' : ''}({nodes.length})</Typography>
                    <Tooltip title={open ? 'Shrink' : 'Expand'}>
                        <IconButton edge="end" color="inherit" aria-label={open ? 'Shrink' : 'Expand'}>
                            {open ? <ShrinkIcon sx={{ fill: palette.background.textPrimary }} /> : <ExpandIcon sx={{ fill: palette.background.textPrimary }} />}
                        </IconButton>
                    </Tooltip>
                </Stack>
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
                                {node.type === NodeType.RoutineList ? null : (<Typography variant="body1" sx={{ marginLeft: 1 }}>{getTranslation(node, 'title', [language], true)}</Typography>)}
                                {/* Delete node icon */}
                                <Tooltip title={`Delete ${getTranslation(node, 'title', [language], true)} node`} placement="left">
                                    <Box sx={{ marginLeft: 'auto' }}>
                                        <IconButton
                                            color="inherit"
                                            onClick={() => handleNodeDelete(node.id)}
                                            aria-label={'Delete unlinked node'}
                                        >
                                            <DeleteIcon sx={{
                                                fill: palette.background.textPrimary,
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
        </Tooltip>
    );
}