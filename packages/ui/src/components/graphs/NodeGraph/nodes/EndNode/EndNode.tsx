import { Box, Tooltip, Typography } from '@mui/material';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { EndNodeProps } from '../types';
import { DraggableNode, NodeContextMenu, NodeWidth } from '../..';
import { nodeLabel } from '../styles';
import { noSelect } from 'styles';
import { CSSProperties } from '@mui/styles';
import { BuildAction } from 'utils';

export const EndNode = ({
    handleAction,
    isEditing,
    isLinked = true,
    node,
    scale = 1,
    label = 'End',
    labelVisible = true,
    canDrag = true,
    zIndex,
}: EndNodeProps) => {

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

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: MouseEvent<HTMLDivElement>) => {
        // Ignore if not linked or editing
        if (!canDrag || !isLinked) return;
        setContextAnchor(ev.currentTarget)
        ev.preventDefault();
    }, [canDrag, isLinked]);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    // Normal click edit menu (title, wasSuccessful, etc.)
    const [editMenuOpen, setEditMenuOpen] = useState<boolean>(false);
    const handleNodeMouseUp = useCallback((event: any) => {
        if (isLinked && !canDrag) {
            setEditMenuOpen(!editMenuOpen);
        }
    }, [canDrag, isLinked, editMenuOpen]);

    return (
        <DraggableNode
            className="handle"
            nodeId={node.id}
            canDrag={canDrag}
            onClick={handleNodeMouseUp}
        >
            {/* Right-click context menu */}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={[BuildAction.AddListBeforeNode, BuildAction.AddListAfterNode, BuildAction.AddEndAfterNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.DeleteNode]}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option, node.id) }}
                zIndex={zIndex + 1}
            />
            {/* Normal-click menu */}
            {/* <EndNodeMenu
                isOpen={contextOpen}
            > */}
            <Tooltip placement={'top'} title={isEditing ? `Edit "${label ?? 'End'}"` : (label ?? 'End')}>
                <Box
                    id={`${isLinked ? '' : 'unlinked-'}node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    onMouseUp={handleNodeMouseUp}
                    sx={{
                        width: outerCircleSize,
                        height: outerCircleSize,
                        fontSize: fontSize,
                        position: 'relative',
                        display: 'block',
                        backgroundColor: '#979696',
                        color: 'white',
                        borderRadius: '100%',
                        boxShadow: '0px 0px 12px gray',
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
                            border: '2px solid white',
                        } as const}
                    >
                        {labelObject}
                    </Box>
                </Box>
            </Tooltip>
        </DraggableNode>
    )
}