import { Box, Tooltip, Typography, useTheme } from '@mui/material';
import { CSSProperties, useCallback, useMemo, useState } from 'react';
import { noSelect } from 'styles';
import { BuildAction } from 'utils/consts';
import usePress from 'utils/hooks/usePress';
import { calculateNodeSize, NodeContextMenu, NodeWidth } from '../..';
import { nodeLabel } from '../styles';
import { StartNodeProps } from '../types';

export const StartNode = ({
    handleAction,
    node,
    scale = 1,
    isEditing,
    label = 'Start',
    labelVisible = true,
    linksOut,
    zIndex,
}: StartNodeProps) => {
    const { palette } = useTheme();

    /**
     * Border color indicates status of node.
     * Red for not fully connected (missing in links)
     */
    const borderColor = useMemo<string | null>(() => {
        if (linksOut.length === 0) return 'red';
        return null;
    }, [linksOut.length]);

    const nodeSize = useMemo(() => calculateNodeSize(NodeWidth.Start, scale), [scale]);

    const labelObject = useMemo(() => labelVisible && nodeSize > 75 ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...nodeLabel,
                fontSize: `min(${nodeSize / 5}px, 2em)`
            } as CSSProperties}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, nodeSize, label]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: EventTarget) => {
        if (!isEditing) return;
        setContextAnchor(target)
    }, [isEditing]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({ 
        onLongPress: openContext,
        onRightClick: openContext,
    });

    return (
        <>
            {/* Right-click context menu */}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={[BuildAction.AddOutgoingLink]}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option as BuildAction.AddOutgoingLink, node.id) }}
                zIndex={zIndex + 1}
            />
            <Tooltip placement={'top'} title={label ?? ''}>
                <Box
                    id={`node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    {...pressEvents}
                    sx={{
                        boxShadow: borderColor ? `0px 0px 12px ${borderColor}` : 12,
                        width: `max(${nodeSize}px, 48px)`,
                        height: `max(${nodeSize}px, 48px)`,
                        position: 'relative',
                        display: 'block',
                        backgroundColor: palette.mode === 'light' ? '#259a17' : '#387e30',
                        color: 'white',
                        borderRadius: '100%',
                        '&:hover': {
                            filter: `brightness(120%)`,
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    {labelObject}
                </Box>
            </Tooltip>
        </>
    )
}