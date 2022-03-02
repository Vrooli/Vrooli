import { Box, Tooltip, Typography } from '@mui/material';
import { CSSProperties, MouseEvent, useCallback, useMemo, useState } from 'react';
import { EndNodeProps } from '../types';
import { DraggableNode, NodeContextMenu, NodeWidth } from '../..';
import { nodeLabel } from '../styles';
import { noSelect } from 'styles';

export const EndNode = ({
    isLinked = true,
    node,
    scale = 1,
    label = 'End',
    labelVisible = true,
    canDrag = true,
}: EndNodeProps) => {

    const labelObject = useMemo(() => labelVisible ? (
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
    ) : null, [labelVisible, label]);

    const outerCircleSize = useMemo(() => `${NodeWidth.End * scale}px`, [scale]);
    const innerCircleSize = useMemo(() => `${NodeWidth.End * scale / 1.5}px`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.End * scale / 5}px, 2em)`, [scale]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: MouseEvent<HTMLDivElement>) => {
        setContextAnchor(ev.currentTarget)
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    return (
        <DraggableNode nodeId={node.id} canDrag={canDrag}>
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                node={node}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={() => { }}
                onEdit={() => { }}
                onMove={() => { }}
            />
            <Tooltip placement={'top'} title={'End'}>
                <Box
                    id={`${isLinked ? '' : 'unlinked-'}node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    onClick={() => { }}
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
                            transition: 'filter 0.2s',
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
                        } as CSSProperties}
                    >
                        {labelObject}
                    </Box>
                </Box>
            </Tooltip>
        </DraggableNode>
    )
}