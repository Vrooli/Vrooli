import { Box, IconButton, Tooltip, Typography } from '@mui/material';
import { CSSProperties, MouseEvent, useCallback, useMemo, useState } from 'react';
import { RedirectNodeProps } from '../types';
import { UTurnLeft as RedirectIcon } from '@mui/icons-material';
import { NodeContextMenu, NodeWidth } from '../..';
import { nodeLabel } from '../styles';
import { noSelect } from 'styles';

export const RedirectNode = ({
    node,
    scale = 1,
    label = 'Redirect',
    labelVisible = true,
}: RedirectNodeProps) => {
    const [dialogOpen, setDialogOpen] = useState(false);
    const openDialog = () => setDialogOpen(true);
    const closeDialog = () => setDialogOpen(false);
    const dialog = useMemo(() => dialogOpen ? (
        <div>TODO</div>
    ) : null, [dialogOpen])

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

    const nodeSize = useMemo(() => `${NodeWidth.Redirect * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.Redirect * scale / 5}px, 2em)`, [scale]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: MouseEvent<HTMLButtonElement>) => {
        setContextAnchor(ev.currentTarget)
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    return (
        <Box className="handle">
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
            <Tooltip placement={'top'} title='Redirect'>
                <IconButton
                    onClick={openDialog}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    sx={{
                        width: nodeSize,
                        height: nodeSize,
                        fontSize: fontSize,
                        position: 'relative',
                        display: 'block',
                        backgroundColor: '#6daf72',
                        color: 'white',
                        boxShadow: '0px 0px 12px gray',
                        '&:hover': {
                            backgroundColor: '#6daf72',
                            filter: `brightness(120%)`,
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    <RedirectIcon
                        sx={{
                            width: '100%',
                            height: '100%',
                            color: '#00000044',
                            '&:hover': {
                                transform: 'scale(1.2)',
                                transition: 'scale .2s ease-in-out',
                            }
                        }}
                    />
                    {labelObject}
                </IconButton>
            </Tooltip>
        </Box>
    )
}