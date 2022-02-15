import { Box, Tooltip, Typography } from '@mui/material';
import { CSSProperties, useCallback, useMemo, useState } from 'react';
import { CombineNodeProps } from '../types';
import { ArrowRightIcon } from 'assets/img';
import { NodeContextMenu, NodeWidth } from '../..';
import { nodeLabel } from '../styles';
import { noSelect } from 'styles';
import { DraggableNode } from '../';

export const CombineNode = ({
    node,
    scale = 1,
    label = 'Combine',
    labelVisible = true,
}: CombineNodeProps) => {
    const [dialogOpen, setDialogOpen] = useState(false);
    const openDialog = () => setDialogOpen(true);
    const closeDialog = () => setDialogOpen(false);
    const dialog = useMemo(() => dialogOpen ? (
        <div>TODO</div>
    ) : null, [dialogOpen])

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    const labelObject = useMemo(() => labelVisible ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...nodeLabel,
                marginLeft: '-20px' as any,
                pointerEvents: 'none' as any,
            } as CSSProperties}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, label]);

    const nodeSize = useMemo(() => `${NodeWidth.Combine * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.Combine * scale / 5}px, 2em)`, [scale]);

    return (
        <DraggableNode nodeId={node.id}>
            {dialog}
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
            <Tooltip placement={'top'} title='Combine'>
                <Box
                    className="handle"
                    aria-owns={contextOpen ? contextId : undefined}
                    sx={{
                        width: nodeSize,
                        height: nodeSize,
                        fontSize: fontSize,
                        position: 'relative',
                        display: 'block',
                        color: 'white',
                        '&:hover': {
                            filter: `brightness(120%)`,
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    <ArrowRightIcon
                        style={{
                            width: '100%',
                            height: '100%',
                            fill: '#6daf72',
                            filter: 'drop-shadow(0px 0px 12px gray)'
                        }}
                        onClick={openDialog}
                    />
                    {labelObject}
                </Box>
            </Tooltip>
        </DraggableNode>
    )
}