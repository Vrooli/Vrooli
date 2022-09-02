import { Box, Tooltip, Typography } from '@mui/material';
import React, { useCallback, useMemo, useState } from 'react';
import { EndNodeProps } from '../types';
import { DraggableNode, EndNodeDialog, NodeContextMenu, NodeWidth } from '../..';
import { nodeLabel } from '../styles';
import { noSelect } from 'styles';
import { CSSProperties } from '@mui/styles';
import { BuildAction, usePress } from 'utils';
import { NodeEnd } from 'types';

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
     * Default (grey) for valid or unlinked, 
     * Red for not fully connected (missing in links)
     */
    const borderColor = useMemo(() => {
        if (!isLinked) return 'gray';
        if (linksIn.length === 0) return 'red';
        return 'gray';
    }, [isLinked, linksIn.length]);

    const labelObject = useMemo(() => labelVisible && scale >= 0.5 ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...nodeLabel,
                pointerEvents: 'none',
            } as CSSProperties}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, label, scale]);

    const outerCircleSize = useMemo(() => `max(${NodeWidth.End * scale}px, 48px)`, [scale]);
    const innerCircleSize = useMemo(() => `max(${NodeWidth.End * scale / 1.5}px, 32px)`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.End * scale / 5}px, 2em)`, [scale]);

    // Normal click edit menu (title, wasSuccessful, etc.)
    const [editDialogOpen, setEditDialogOpen] = useState<boolean>(false);
    const openEditDialog = useCallback((event: any) => {
        console.log('handleNodeClick', isLinked, event);
        if (isLinked) {
            setEditDialogOpen(!editDialogOpen);
        }
    }, [isLinked, editDialogOpen]);
    // const handleNodeMouseUp = useCallback((event: any) => {
    //     console.log('handleNodeMouseUp', isLinked, event);
    //     if (isLinked && !canDrag) {
    //         setEditDialogOpen(!editDialogOpen);
    //     }
    // } , [isLinked, canDrag, editDialogOpen]);
    const handleEditDialogClose = useCallback((updatedNode?: NodeEnd) => {
        if (updatedNode) handleUpdate(updatedNode);
        setEditDialogOpen(false);
    }, [handleUpdate]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: React.MouseEvent['target']) => {
        console.log('opencontext')
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
            <Tooltip placement={'top'} title={isEditing ? `Edit "${label ?? 'End'}"` : (label ?? 'End')}>
                <Box
                    id={`${isLinked ? '' : 'unlinked-'}node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    {...pressEvents}
                    sx={{
                        width: outerCircleSize,
                        height: outerCircleSize,
                        fontSize: fontSize,
                        position: 'relative',
                        display: 'block',
                        backgroundColor: node.data?.wasSuccessful === false ? '#7c262a' : '#387e30',
                        color: 'white',
                        borderRadius: '100%',
                        boxShadow: `0px 0px 12px ${borderColor}`,
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
                            width: innerCircleSize,
                            height: innerCircleSize,
                            position: 'absolute',
                            display: 'block',
                            margin: '0',
                            top: '50%',
                            left: '50%',
                            transform: 'translate(-50%, -50%)',
                            borderRadius: '100%',
                            border: `2px solid ${node.data?.wasSuccessful === false ? '#e97691' : '#9ce793'}`,
                        } as const}
                    >
                        {labelObject}
                    </Box>
                </Box>
            </Tooltip>
        </DraggableNode>
    )
}