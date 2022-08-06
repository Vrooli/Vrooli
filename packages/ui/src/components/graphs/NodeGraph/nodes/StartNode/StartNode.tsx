import { Box, Tooltip, Typography, useTheme } from '@mui/material';
import { CSSProperties, useCallback, useMemo, useState } from 'react';
import { StartNodeProps } from '../types';
import { nodeLabel } from '../styles';
import { noSelect } from 'styles';
import { NodeContextMenu, NodeWidth } from '../..';
import { BuildAction, useLongPress } from 'utils';

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
     * Default (grey) for valid or unlinked, 
     * Red for not fully connected (missing in links)
     */
    const borderColor = useMemo(() => {
        if (linksOut.length === 0) return 'red';
        return 'gray';
    }, [linksOut.length]);

    const labelObject = useMemo(() => labelVisible && scale >= 0.5 ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...nodeLabel,
            } as CSSProperties}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, label, scale]);

    const nodeSize = useMemo(() => `max(${NodeWidth.Start * scale}px, 48px)`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.Start * scale / 5}px, 2em)`, [scale]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        if (!isEditing) return;
        ev.preventDefault();
        setContextAnchor(ev.currentTarget ?? ev.target)
    }, [isEditing]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const longPressEvent = useLongPress({ onLongPress: openContext });

    return (
        <>
            {/* Right-click context menu */}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={[BuildAction.AddOutgoingLink]}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option, node.id) }}
                zIndex={zIndex + 1}
            />
            <Tooltip placement={'top'} title={label ?? ''}>
                <Box
                    id={`node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    {...longPressEvent}
                    sx={{
                        boxShadow: `0px 0px 12px ${borderColor}`,
                        width: nodeSize,
                        height: nodeSize,
                        fontSize: fontSize,
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