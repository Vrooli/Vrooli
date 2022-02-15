import { Box, Tooltip, Typography } from '@mui/material';
import { CSSProperties, MouseEvent, useCallback, useMemo, useState } from 'react';
import { DecisionNodeProps } from '../types';
import { NodeContextMenu, NodeWidth } from '../..';
import { nodeLabel } from '../styles';
import { noSelect, textShadow } from 'styles';
import { DraggableNode } from '../';

export const DecisionNode = ({
    node,
    scale = 1,
    label = 'Continue?',
    text = 'Would you like to continue?',
    labelVisible = true,
}: DecisionNodeProps) => {

    const labelObject = useMemo(() => labelVisible ? (
        <Typography
            sx={{
                ...noSelect,
                ...nodeLabel,
            } as CSSProperties}
        >{label}</Typography>
    ) : null, [labelVisible, label]);

    const nodeSize = useMemo(() => `${NodeWidth.Decision * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.Decision * scale / 5}px, 2em)`, [scale]);

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
        <DraggableNode nodeId={node.id}>
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
            <Tooltip placement={'top'} title={text}>
                <Box
                    className="handle"
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    onClick={() => { }}
                    sx={{
                        width: nodeSize, 
                        height: nodeSize, 
                        lineHeight: nodeSize, 
                        fontSize: fontSize,
                        position: 'relative' as any,
                        textAlign: 'center',
                        margin: '10px 40px',
                        color: 'white',
                        '&:before': {
                            ...textShadow,
                            position: 'absolute',
                            content: '""',
                            top: '0',
                            left: '0',
                            width: '100%',
                            height: '100%',
                            backgroundColor: '#68b4c5',
                            transform: 'rotateX(45deg) rotateZ(45deg)',
                        },
                        '&:after': {
                            position: 'absolute',
                            content: '""',
                            top: '10px',
                            left: '10px',
                            height: 'calc(100% - 22px)',
                            width: 'calc(100% - 22px)',
                            border: '1px solid organge',
                            transform: 'rotateX(45deg) rotateZ(45deg)',
                        },
                        '&:hover': {
                            filter: `brightness(120%)`,
                            transition: 'filter 0.2s',
                        },
                    } as CSSProperties}
                >
                    {labelObject}
                </Box>
            </Tooltip>
        </DraggableNode>
    )
}