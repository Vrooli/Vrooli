import { Box, Tooltip, Typography } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { EndNodeProps } from '../types';
import { calculateNodeSize, DraggableNode, EndNodeDialog, NodeContextMenu, NodeWidth } from '../..';
import { nodeLabel } from '../styles';
import { noSelect } from 'styles';
import { CSSProperties } from '@mui/styles';
import { BuildAction, firstString, usePress } from 'utils';

/**
 * Distance before a click is considered a drag
 */
const DRAG_THRESHOLD = 10;

export const EndNode = ({
    canDrag = true,
    handleAction,
    handleUpdate,
    isEditing,
    isLinked = true,
    label = 'End',
    labelVisible = true,
    language,
    linksIn,
    node,
    scale = 1,
    zIndex,
}: EndNodeProps) => {

    /**
     * Border color indicates status of node.
     * Red for not fully connected (missing in links)
     */
    const borderColor = useMemo<string | null>(() => {
        if (!isLinked) return null;
        if (linksIn.length === 0) return 'red';
        return null;
    }, [isLinked, linksIn.length]);

    const { innerCircleSize, outerCircleSize } = useMemo(() => {
        const nodeSize = calculateNodeSize(NodeWidth.End, scale);
        return {
            innerCircleSize: nodeSize / 2,
            outerCircleSize: nodeSize,
        }
    }, [scale]);

    const labelObject = useMemo(() => labelVisible && outerCircleSize > 75 ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...nodeLabel,
                pointerEvents: 'none',
                fontSize: `min(${outerCircleSize / 5}px, 2em)`
            } as CSSProperties}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, outerCircleSize, label]);

    // Normal click edit menu (title, wasSuccessful, etc.)
    const [editDialogOpen, setEditDialogOpen] = useState<boolean>(false);
    const openEditDialog = useCallback((event: any) => {
        if (isLinked) {
            setEditDialogOpen(!editDialogOpen);
        }
    }, [isLinked, editDialogOpen]);
    const handleEditDialogClose = useCallback((updatedNode?: NodeEnd) => {
        if (updatedNode) handleUpdate(updatedNode);
        setEditDialogOpen(false);
    }, [handleUpdate]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: EventTarget) => {
        // Ignore if not linked or not editing
        if (!canDrag || !isLinked || !isEditing) return;
        setContextAnchor(target)
    }, [canDrag, isLinked, isEditing]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: openEditDialog,
        onRightClick: openContext,
    });

    return (
        <DraggableNode
            className="handle"
            nodeId={node.id}
            canDrag={canDrag}
            dragThreshold={DRAG_THRESHOLD}
        >
            {/* Right-click context menu */}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={[BuildAction.AddListBeforeNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.DeleteNode]}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option, node.id) }}
                zIndex={zIndex + 1}
            />
            {/* Normal-click menu */}
            <EndNodeDialog
                handleClose={handleEditDialogClose}
                isEditing={isEditing}
                isOpen={editDialogOpen}
                language={language}
                node={node}
                zIndex={zIndex + 1}
            />
            <Tooltip placement={'top'} title={isEditing ? `Edit "${firstString(label, 'End')}"` : firstString(label, 'End')}>
                <Box
                    id={`${isLinked ? '' : 'unlinked-'}node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    {...pressEvents}
                    sx={{
                        width: `max(${outerCircleSize}px, 48px)`,
                        height: `max(${outerCircleSize}px, 48px)`,
                        position: 'relative',
                        display: 'block',
                        backgroundColor: node.nodeEnd?.wasSuccessful === false ? '#7c262a' : '#387e30',
                        color: 'white',
                        borderRadius: '100%',
                        boxShadow: borderColor ? `0px 0px 12px ${borderColor}` : 12,
                        '&:hover': {
                            filter: `brightness(120%)`,
                            transform: 'scale(1.1)',
                            transition: 'all 0.2s',
                        },
                    }}
                >
                    <Box
                        id={`${isLinked ? '' : 'unlinked-'}node-end-inner-circle-${node.id}`}
                        sx={{
                            width: `max(${innerCircleSize}px, 32px)`,
                            height: `max(${innerCircleSize}px, 32px)`,
                            position: 'absolute',
                            display: 'block',
                            margin: '0',
                            top: '50%',
                            left: '50%',
                            transform: 'translate(-50%, -50%)',
                            borderRadius: '100%',
                            border: `2px solid ${node.nodeEnd?.wasSuccessful === false ? '#e97691' : '#9ce793'}`,
                        } as const}
                    >
                    </Box>
                    {labelObject}
                </Box>
            </Tooltip>
        </DraggableNode>
    )
}