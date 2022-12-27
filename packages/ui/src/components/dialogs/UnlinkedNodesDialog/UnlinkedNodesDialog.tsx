/**
 * Displays nodes associated with a routine, but that are not linked to any other nodes.
 */
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
import { Node, NodeEnd, NodeRoutineList } from 'types';
import { EndNode, RedirectNode, RoutineListNode } from 'components';
import { getTranslation } from 'utils';
import { DeleteIcon, ExpandLessIcon, ExpandMoreIcon, UnlinkedNodesIcon } from '@shared/icons';

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
            label: getTranslation(node, [language], false).title ?? null,
            labelVisible: false,
            scale: 0.8,
            zIndex,
        }
        // Determine node to display based on node type
        switch (node.type) {
            case NodeType.End:
                return <EndNode
                    {...nodeProps}
                    handleUpdate={() => { }} // Intentionally blank
                    language={language}
                    linksIn={[]}
                    node={node as NodeEnd}
                />
            case NodeType.Redirect:
                return <RedirectNode
                    {...nodeProps}
                    node={node as Node}//as NodeRedirect}
                />
            case NodeType.RoutineList:
                return <RoutineListNode
                    {...nodeProps}
                    canExpand={false}
                    labelVisible={true}
                    language={language}
                    handleUpdate={() => { }} // Intentionally blank
                    linksIn={[]}
                    linksOut={[]}
                    node={node as NodeRoutineList}
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
                background: palette.secondary.main,
                color: palette.secondary.contrastText,
                paddingLeft: 1,
                paddingRight: 1,
                marginRight: 1,
                marginTop: open ? '4px' : 'unset',
                maxHeight: { xs: '50vh', sm: '65vh', md: '72vh' },
                overflowX: 'hidden',
                overflowY: 'auto',
                width: open ? { xs: '100%', sm: '375px' } : 'fit-content',
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
                    <UnlinkedNodesIcon fill={palette.secondary.contrastText} />
                    <Typography variant="h6" sx={{ ...noSelect, marginLeft: '8px' }}>{open ? 'Unlinked ' : ''}({nodes.length})</Typography>
                    <Tooltip title={open ? 'Shrink' : 'Expand'}>
                        <IconButton edge="end" color="inherit" aria-label={open ? 'Shrink' : 'Expand'}>
                            {open ? <ExpandLessIcon fill={palette.secondary.contrastText} /> : <ExpandMoreIcon fill={palette.secondary.contrastText} />}
                        </IconButton>
                    </Tooltip>
                </Stack>
                {open ? (
                    <Stack direction="column" spacing={1}>
                        {nodes.map((node) => (
                            <Box key={node.id} sx={{ display: 'flex', alignItems: 'center' }}>
                                {/* Miniature version of node */}
                                <Box sx={{
                                    height: '50px',
                                }}>
                                    {createNode(node)}
                                </Box>
                                {/* Node title */}
                                {node.type === NodeType.RoutineList ? null : (<Typography variant="body1" sx={{ marginLeft: 1 }}>{getTranslation(node, [language], true).title}</Typography>)}
                                {/* Delete node icon */}
                                <Tooltip title={`Delete ${getTranslation(node, [language], true).title} node`} placement="left">
                                    <Box sx={{ marginLeft: 'auto' }}>
                                        <IconButton
                                            color="inherit"
                                            onClick={() => handleNodeDelete(node.id)}
                                            aria-label={'Delete unlinked node'}
                                        >
                                            <DeleteIcon fill={palette.background.textPrimary} />
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